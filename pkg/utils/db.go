package utils

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"time"
)

func CreatePostgresConnection(host string, port string, user string, password string, dbname string, sslmode string) *sql.DB {
	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to open database connection: %w", err))
	}

	if err := db.Ping(); err != nil {
		db.Close()
		log.Fatal(fmt.Errorf("failed to ping database: %w", err))
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Successfully connected to PostgreSQL database")
	return db
}
