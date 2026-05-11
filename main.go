package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	importer "location-service/cmd/importer"
	"location-service/infrastructure/database"
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
	case "migrate":
		if err := migrate(); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown command %q; use serve, import, or migrate", cmd)
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
	port := fs.String("port", utils.Env("PORT", "8088"), "HTTP port")
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

	addr := ":" + strings.TrimPrefix(*port, ":")
	log.Printf("location-service listening on %s", addr)
	return http.ListenAndServe(addr, router.New(db))
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
	log.Printf("import done: raw=%d provinces=%d regencies=%d districts=%d villages=%d", stats.Raw, stats.Provinces, stats.Regencies, stats.Districts, stats.Villages)
	return nil
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
