package config

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/Igorjr19/go-shorty/internal/logger"
	_ "github.com/lib/pq"
)

func ConnectDB() *sql.DB {
	dbHost := os.Getenv("PGHOST")
	dbPort := os.Getenv("PGPORT")
	dbUser := os.Getenv("PGUSER")
	dbPassword := os.Getenv("PGPASSWORD")
	dbName := os.Getenv("PGDATABASE")
	dbUseSSL := os.Getenv("PGSSLMODE")
	dbConnectTimeout := os.Getenv("PGCONNECT_TIMEOUT")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s connect_timeout=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbUseSSL, dbConnectTimeout)

	ctx := context.Background()
	logger.Info(ctx, "Connecting to database",
		slog.String("host", dbHost),
		slog.String("port", dbPort),
		slog.String("database", dbName),
		slog.String("user", dbUser),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		logger.Error(ctx, "Failed to connect to database", slog.String("error", err.Error()))
		panic(fmt.Sprintf("Unable to connect to database: %v", err))
	}

	if err := db.Ping(); err != nil {
		logger.Error(ctx, "Failed to ping database", slog.String("error", err.Error()))
		panic(fmt.Sprintf("Unable to ping database: %v", err))
	}

	logger.Info(ctx, "Database connection established successfully")

	return db
}
