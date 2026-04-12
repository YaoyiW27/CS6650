package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func New(databaseURL string) (*DB, error) {
	conn, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}
	conn.SetMaxOpenConns(50)
	conn.SetMaxIdleConns(25)
	conn.SetConnMaxLifetime(30 * time.Minute)
	conn.SetConnMaxIdleTime(5 * time.Minute)
	return &DB{conn}, nil
}

func (d *DB) Migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS albums (
		album_id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		description TEXT NOT NULL,
		owner TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS photos (
		photo_id TEXT PRIMARY KEY,
		album_id TEXT NOT NULL REFERENCES albums(album_id),
		seq INT NOT NULL,
		status TEXT NOT NULL DEFAULT 'processing',
		url TEXT,
		UNIQUE(album_id, seq)
	);

	CREATE TABLE IF NOT EXISTS album_seq (
		album_id TEXT PRIMARY KEY REFERENCES albums(album_id),
		next_seq INT NOT NULL DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS photo_data (
		photo_id TEXT PRIMARY KEY,
		data BYTEA NOT NULL
	);
	`
	_, err := d.Exec(schema)
	return err
}