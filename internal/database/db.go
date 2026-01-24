package database

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/thomaskoefod/newsreadr/pkg/models"
	_ "modernc.org/sqlite"
)

var (
	ErrFeedExists = errors.New("feed already exists")
)

type DB struct {
	*sql.DB
}

// New creates a new database connection and initializes schema
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	d := &DB{db}
	if err := d.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initializing schema: %w", err)
	}

	return d, nil
}

// initSchema creates database tables if they don't exist
func (db *DB) initSchema() error {
	schema := `
		CREATE TABLE IF NOT EXISTS feeds (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feed_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			url TEXT NOT NULL UNIQUE,
			content TEXT,
			description TEXT,
			published_at TIMESTAMP NOT NULL,
			fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			relevance_score REAL DEFAULT 0,
			FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS user_interests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			description TEXT NOT NULL,
			weight REAL NOT NULL DEFAULT 1.0,
			embedding BLOB
		);

		CREATE TABLE IF NOT EXISTS read_articles (
			article_id INTEGER PRIMARY KEY,
			read_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE
		);

		CREATE INDEX IF NOT EXISTS idx_articles_published_at ON articles(published_at);
		CREATE INDEX IF NOT EXISTS idx_articles_relevance_score ON articles(relevance_score);
		CREATE INDEX IF NOT EXISTS idx_articles_feed_id ON articles(feed_id);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}

	return nil
}

// AddFeed adds a feed by URL and name (convenience method)
func (db *DB) AddFeed(url, name string) error {
	feed := &models.Feed{
		URL:       url,
		Name:      name,
		Enabled:   true,
		CreatedAt: time.Now(),
	}

	err := db.AddFeedModel(feed)
	if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return ErrFeedExists
	}
	return err
}

// AddFeedModel inserts a new feed using a Feed model
func (db *DB) AddFeedModel(feed *models.Feed) error {
	result, err := db.Exec(
		"INSERT INTO feeds (url, name, enabled, created_at) VALUES (?, ?, ?, ?)",
		feed.URL, feed.Name, feed.Enabled, feed.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting feed: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert id: %w", err)
	}

	feed.ID = id
	return nil
}
