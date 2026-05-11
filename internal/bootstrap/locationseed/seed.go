package locationseed

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	importer "location-service/cmd/importer"
	"location-service/utils"
)

func Run(ctx context.Context, db *sql.DB) error {
	if !envBool("AUTO_SEED", true) {
		return nil
	}

	locked, err := tryLock(ctx, db)
	if err != nil {
		return err
	}
	if !locked {
		log.Print("seed skipped: another instance is handling auto seed")
		return nil
	}
	defer releaseLock(db)

	existing, err := countRawLocations(ctx, db)
	if err != nil {
		return err
	}
	if existing > 0 {
		log.Printf("seed skipped: raw_locations already has %d rows", existing)
		return nil
	}

	seedFile, ok, err := resolveSeedFile()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	log.Printf("seed started: importing %s", seedFile)
	stats, err := importer.Import(ctx, db, seedFile, true)
	if err != nil {
		return err
	}
	log.Printf("seed done: raw=%d provinces=%d regencies=%d districts=%d villages=%d", stats.Raw, stats.Provinces, stats.Regencies, stats.Districts, stats.Villages)
	return nil
}

func tryLock(ctx context.Context, db *sql.DB) (bool, error) {
	var locked bool
	err := db.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(hashtext('location-service:auto-seed'))`).Scan(&locked)
	return locked, err
}

func releaseLock(db *sql.DB) {
	var unlocked bool
	if err := db.QueryRowContext(context.Background(), `SELECT pg_advisory_unlock(hashtext('location-service:auto-seed'))`).Scan(&unlocked); err != nil {
		log.Printf("seed lock release failed: %v", err)
	}
}

func countRawLocations(ctx context.Context, db *sql.DB) (int, error) {
	var existing int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM raw_locations`).Scan(&existing)
	return existing, err
}

func resolveSeedFile() (string, bool, error) {
	seedFile := utils.Env("SEED_FILE", "data/wilayah.sql")
	if _, err := os.Stat(seedFile); err == nil {
		return seedFile, true, nil
	}

	fallback := "../wilayah.sql"
	if _, err := os.Stat(fallback); err == nil {
		return fallback, true, nil
	}

	if envBool("AUTO_SEED_REQUIRED", false) {
		return "", false, fmt.Errorf("seed file not found: %s", seedFile)
	}

	log.Printf("seed skipped: seed file not found: %s", seedFile)
	return "", false, nil
}

func envBool(key string, def bool) bool {
	value := strings.ToLower(strings.TrimSpace(utils.Env(key, "")))
	if value == "" {
		return def
	}
	return value == "true" || value == "1" || value == "yes" || value == "on"
}
