package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

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

	log.Printf("Connecting to database: host=%s port=%s dbname=%s user=%s", dbHost, dbPort, dbName, dbUser)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}

	log.Println("Database connection established successfully")

	return db
}
