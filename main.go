package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	importer "location-service/cmd/importer"
	"location-service/infrastructure/database"
	"location-service/infrastructure/media"
	"location-service/internal/bootstrap/locationseed"
	"location-service/internal/router"
	"location-service/utils"
)

func main() {
	utils.LoadEnvFile(".env")

	cmd := command()
	switch cmd {
	case "serve":
		if err := serve(); err != nil {
			log.Fatal(err)
		}
	case "import":
		if err := importData(); err != nil {
			log.Fatal(err)
		}
	case "import-boundaries":
		if err := importBoundaries(); err != nil {
			log.Fatal(err)
		}
	case "import-islands":
		if err := importIslands(); err != nil {
			log.Fatal(err)
		}
	case "import-population":
		if err := importPopulation(); err != nil {
			log.Fatal(err)
		}
	case "import-areas":
		if err := importAreas(); err != nil {
			log.Fatal(err)
		}
	case "import-postal-codes":
		if err := importPostalCodes(); err != nil {
			log.Fatal(err)
		}
	case "migrate":
		if err := migrate(); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown command %q; use serve, import, import-boundaries, import-islands, import-population, import-areas, import-postal-codes, or migrate", cmd)
	}
}

func command() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	if len(strings.TrimSpace(utils.Env("COMMAND", ""))) > 0 {
		return utils.Env("COMMAND", "")
	}
	return "serve"
}

func serve() error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.String("port", utils.Env("PORT", "8080"), "HTTP port")
	if err := fs.Parse(commandArgs()); err != nil {
		return err
	}

	db, err := database.Open()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	if err := locationseed.Run(context.Background(), db); err != nil {
		return fmt.Errorf("seed data: %w", err)
	}

	redisClient, err := database.OpenRedis()
	if err != nil {
		log.Printf("redis disabled: %v", err)
	}
	if redisClient != nil {
		defer redisClient.Close()
	}
	storageProvider, err := media.InitStorage()
	if err != nil {
		return fmt.Errorf("initialize storage: %w", err)
	}

	addr := ":" + strings.TrimPrefix(*port, ":")
	log.Printf("location-service listening on %s", addr)
	server := &http.Server{
		Addr:              addr,
		Handler:           router.New(db, redisClient, storageProvider),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	return server.ListenAndServe()
}

func importData() error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	path := fs.String("file", "../wilayah.sql", "path to wilayah.sql")
	truncate := fs.Bool("truncate", true, "truncate normalized tables before import")
	if err := fs.Parse(commandArgs()); err != nil {
		return err
	}

	db, err := database.Open()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	stats, err := importer.Import(context.Background(), db, *path, *truncate)
	if err != nil {
		return err
	}
	invalidateCache("location:*")
	log.Printf("import done: raw=%d provinces=%d regencies=%d districts=%d villages=%d", stats.Raw, stats.Provinces, stats.Regencies, stats.Districts, stats.Villages)
	return nil
}

type pathList []string

func (p *pathList) String() string {
	return strings.Join(*p, ",")
}

func (p *pathList) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("boundary file path is required")
	}
	*p = append(*p, value)
	return nil
}

func importBoundaries() error {
	fs := flag.NewFlagSet("import-boundaries", flag.ExitOnError)
	var files pathList
	fs.Var(&files, "file", "gzip SQL boundary file; repeat for multiple files")
	dir := fs.String("dir", "", "directory containing boundary gzip files")
	if err := fs.Parse(commandArgs()); err != nil {
		return err
	}

	paths := append([]string(nil), files...)
	if strings.TrimSpace(*dir) != "" {
		boundaryFiles, err := importer.BoundaryFiles(*dir)
		if err != nil {
			return err
		}
		paths = append(paths, boundaryFiles...)
	}
	if len(paths) == 0 {
		return fmt.Errorf("provide -dir or at least one -file")
	}

	db, err := database.Open()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	storageProvider, err := media.RequiredStorage()
	if err != nil {
		return err
	}
	stats, err := importer.ImportBoundaries(context.Background(), db, storageProvider, paths)
	if err != nil {
		return err
	}
	invalidateCache("location:boundary:*")
	log.Printf("boundary import done: read=%d imported=%d skipped_unknown=%d", stats.Read, stats.Imported, stats.SkippedUnknown)
	return nil
}

func importIslands() error {
	fs := flag.NewFlagSet("import-islands", flag.ExitOnError)
	path := fs.String("file", "../wilayah-indonesia-api/init-db/02-data.sql", "path to SQL containing wilayah_pulau")
	if err := fs.Parse(commandArgs()); err != nil {
		return err
	}

	db, err := database.Open()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	stats, err := importer.ImportIslands(context.Background(), db, *path)
	if err != nil {
		return err
	}
	invalidateCache("location:islands:*")
	log.Printf("island import done: read=%d imported=%d skipped=%d duplicate_codes=%d", stats.RowsRead, stats.RowsImported, stats.RowsSkipped, stats.DuplicateCodes)
	return nil
}

func importPopulation() error {
	fs := flag.NewFlagSet("import-population", flag.ExitOnError)
	path := fs.String("file", "../wilayah-indonesia-api/init-db/02-data.sql", "path to SQL containing wilayah_penduduk")
	if err := fs.Parse(commandArgs()); err != nil {
		return err
	}

	db, err := database.Open()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	stats, err := importer.ImportPopulation(context.Background(), db, *path)
	if err != nil {
		return err
	}
	invalidateCache("location:population:*")
	log.Printf("population import done: read=%d imported=%d skipped=%d national=%d unknown=%d", stats.RowsRead, stats.RowsImported, stats.RowsSkipped, stats.NationalRows, stats.UnknownCodes)
	return nil
}

func importAreas() error {
	fs := flag.NewFlagSet("import-areas", flag.ExitOnError)
	path := fs.String("file", "../wilayah-indonesia-api/init-db/02-data.sql", "path to SQL containing wilayah_luas")
	if err := fs.Parse(commandArgs()); err != nil {
		return err
	}

	db, err := database.Open()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	stats, err := importer.ImportAreas(context.Background(), db, *path)
	if err != nil {
		return err
	}
	invalidateCache("location:area:*")
	log.Printf("area import done: read=%d imported=%d unknown=%d code_corrections=%d name_corrections=%d", stats.RowsRead, stats.RowsImported, stats.RowsSkippedUnknown, stats.CodeCorrections, stats.NameCorrections)
	return nil
}

func importPostalCodes() error {
	fs := flag.NewFlagSet("import-postal-codes", flag.ExitOnError)
	path := fs.String("file", "data/kodepos.sql", "path to postal-code SQL seed")
	if err := fs.Parse(commandArgs()); err != nil {
		return err
	}

	db, err := database.Open()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	count, err := importer.ImportPostalCodes(context.Background(), db, *path)
	if err != nil {
		return err
	}
	invalidateCache("location:*")
	log.Printf("postal-code import done: villages=%d", count)
	return nil
}

func invalidateCache(pattern string) {
	client, err := database.OpenRedis()
	if err != nil {
		log.Printf("cache invalidation skipped: %v", err)
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	iterator := client.Scan(ctx, 0, pattern, 1000).Iterator()
	for iterator.Next(ctx) {
		if err := client.Del(ctx, iterator.Val()).Err(); err != nil {
			log.Printf("cache invalidation failed: %v", err)
			return
		}
	}
	if err := iterator.Err(); err != nil {
		log.Printf("cache invalidation failed: %v", err)
	}
}

func migrate() error {
	db, err := database.Open()
	if err != nil {
		return err
	}
	defer db.Close()
	return database.Migrate(db)
}

func commandArgs() []string {
	if len(os.Args) <= 2 {
		return nil
	}
	return os.Args[2:]
}
