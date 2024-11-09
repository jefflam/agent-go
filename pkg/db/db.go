package db

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/lisanmuaddib/agent-go/pkg/db/models"
)

// SetupDatabase initializes the database connection and runs migrations
func SetupDatabase(logger *logrus.Logger) (*gorm.DB, error) {
	logger.Debug("Starting database setup")

	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root: %w", err)
	}

	// Run migrations
	if err := RunMigrations(logger, projectRoot); err != nil {
		return nil, err
	}

	// Construct DSN
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	logger.Debug("Establishing GORM database connection")

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: NewGormLogrusLogger(logger),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Ensure enum type exists
	if err := ensureTweetCategoryEnum(db); err != nil {
		return nil, fmt.Errorf("failed to ensure tweet_category enum: %w", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&models.Tweet{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database schema: %w", err)
	}

	logger.Info("Database setup completed successfully")
	return db, nil
}
