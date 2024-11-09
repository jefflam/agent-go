package db

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/gorm"
)

// findProjectRoot looks for go.mod file to determine project root
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (go.mod)")
		}
		dir = parent
	}
}

// constructDBURL creates the database URL from environment variables
func constructDBURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
}

// ensureTweetCategoryEnum ensures the tweet_category enum type exists
func ensureTweetCategoryEnum(db *gorm.DB) error {
	var exists bool
	err := db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM pg_type 
			WHERE typname = 'tweet_category'
		);
	`).Scan(&exists).Error

	if err != nil {
		return err
	}

	if !exists {
		err := db.Exec(`
			CREATE TYPE tweet_category AS ENUM (
				'mention',
				'reply',
				'quote',
				'retweet',
				'dm',
				'conversation'
			);
		`).Error
		if err != nil {
			return err
		}
	}

	return nil
}
