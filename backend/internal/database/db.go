package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

type DB struct {
	*sql.DB
	config *Config
}

func New(config *Config) (*DB, error) {
	db, err := sql.Open("postgres", config.ConnectionString())

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failedll to ping database: %w", err)
	}

	log.Printf("Connected to PostgreSQL database:%s", config.DBname)

	return &DB{
		DB:     db,
		config: config,
	}, nil
}

func (db *DB) Close() error {
	log.Println("Closing database connection")
	return db.DB.Close()
}

func (db *DB) Stats() sql.DBStats {
	return db.DB.Stats()
}

func (db *DB) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}
