package main

import (
	"flag"
	"log"

	"github.com/Igorjr19/go-shorty/internal/config"
	"github.com/Igorjr19/go-shorty/internal/migrate"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	direction := flag.String("direction", "up", "Migration direction: up or down")
	steps := flag.Int("steps", 0, "Number of migrations to run (0 = all)")
	flag.Parse()

	db := config.ConnectDB()
	defer db.Close()

	migrator := migrate.NewMigrator(db, "migrations")

	var err error
	switch *direction {
	case "up":
		if *steps > 0 {
			err = migrator.UpSteps(*steps)
		} else {
			err = migrator.Up()
		}
	case "down":
		if *steps > 0 {
			err = migrator.DownSteps(*steps)
		} else {
			err = migrator.Down()
		}
	default:
		log.Fatalf("Invalid direction: %s. Use 'up' or 'down'", *direction)
	}

	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully!")
}
