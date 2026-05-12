package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"

	"location-service/utils"
)

func Open() (*sql.DB, error) {
	dsn := utils.Env("DATABASE_URL", "")
	if dsn == "" {
		dsn = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			utils.Env("DB_USERNAME", "location"),
			utils.Env("DB_PASS", "location"),
			utils.Env("DB_HOST", "localhost"),
			utils.Env("DB_PORT", "5438"),
			utils.Env("DB_NAME", "location"),
			utils.Env("DB_SSLMODE", "disable"),
		)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func Migrate(db *sql.DB) error {
	path := utils.Env("PATH_MIGRATE", "migrations/000001_init.sql")
	query, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = db.Exec(string(query))
	return err
}
