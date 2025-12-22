package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/lib/pq"
	"log"
	"os"
	"redifu-example/internal/dbconn"
)

type MigrationConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
	Help     bool
}

func ParseMigrationArgs() *MigrationConfig {
	config := &MigrationConfig{}

	flag.StringVar(&config.Host, "host", "localhost", "Database host")
	flag.StringVar(&config.Port, "port", "5432", "Database port")
	flag.StringVar(&config.User, "user", "", "Database user (required)")
	flag.StringVar(&config.Password, "password", "", "Database password (required)")
	flag.StringVar(&config.Database, "database", "", "Database name (required)")
	flag.StringVar(&config.SSLMode, "sslmode", "disable", "SSL mode (disable, require, verify-ca, verify-full)")
	flag.BoolVar(&config.Help, "help", false, "Show help message")
	flag.BoolVar(&config.Help, "h", false, "Show help message")

	flag.Parse()

	return config
}

func ShowHelp() {
	fmt.Println("Redifu Example Migration Tool")
	fmt.Println("=======================")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run main.go migrate [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -host string       Database host (default: localhost)")
	fmt.Println("  -port string       Database port (default: 5432)")
	fmt.Println("  -user string       Database user (required)")
	fmt.Println("  -password string   Database password (required)")
	fmt.Println("  -database string   Database name (required)")
	fmt.Println("  -sslmode string    SSL mode: disable, require, verify-ca, verify-full (default: disable)")
	fmt.Println("  -help, -h          Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run main.go migrate -user myuser -password mypass -database mydb")
	fmt.Println("  go run main.go migrate -host db.example.com -port 5432 -user admin -password secret -database tickets -sslmode require")
	fmt.Println()
	fmt.Println("Tables created:")
	fmt.Println("  - account: User account information")
	fmt.Println("  - ticket:  Support tickets with foreign key to account")
}

func ValidateConfig(config *MigrationConfig) error {
	if config.User == "" {
		return fmt.Errorf("database user is required (use -user flag)")
	}
	if config.Password == "" {
		return fmt.Errorf("database password is required (use -password flag)")
	}
	if config.Database == "" {
		return fmt.Errorf("database name is required (use -database flag)")
	}
	return nil
}

func CreateTables(config *MigrationConfig) {
	db := dbconn.CreatePostgresConnection(config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode)
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
	}()

	log.Printf("Connecting to database: %s@%s:%s/%s", config.User, config.Host, config.Port, config.Database)

	createAccountTable(db)
	createTicketTable(db)

	log.Println("Migration completed successfully")
}

func createAccountTable(db *sql.DB) {
	createAccountTable := `
		CREATE TABLE IF NOT EXISTS account (
		    uuid varchar(36) PRIMARY KEY,
		    randid varchar(16) UNIQUE NOT NULL,
		    created_at timestamp NOT NULL DEFAULT NOW(),
		    updated_at timestamp NOT NULL DEFAULT NOW(),
		    name varchar(255) NOT NULL,
		    email varchar(255) NOT NULL
	  	);
	`

	_, errCreateAccountTable := db.Exec(createAccountTable)
	if errCreateAccountTable != nil {
		log.Fatal("Failed to create account table:", errCreateAccountTable)
	}

	log.Println("Account table created successfully")
}

func createTicketTable(db *sql.DB) {
	createTicketTable := `
		CREATE TABLE IF NOT EXISTS ticket (
		    uuid varchar(36) PRIMARY KEY,
		    randid varchar(16) UNIQUE NOT NULL,
		    created_at timestamp NOT NULL DEFAULT NOW(),
		    updated_at timestamp NOT NULL DEFAULT NOW(),
		    account_uuid varchar(36) NOT NULL,
		    description text NOT NULL,
		    resolved boolean NOT NULL DEFAULT false,
		    security_risk bigint NOT NULL DEFAULT 0,
		    FOREIGN KEY (account_uuid) REFERENCES account(uuid) ON DELETE CASCADE
	  	);
	`

	_, errCreateTicketTable := db.Exec(createTicketTable)
	if errCreateTicketTable != nil {
		log.Fatal("Failed to create ticket table:", errCreateTicketTable)
	}

	log.Println("Ticket table created successfully")
}

func StartMigration() {
	config := ParseMigrationArgs()

	if config.Help {
		ShowHelp()
		return
	}

	if err := ValidateConfig(config); err != nil {
		fmt.Printf("Error: %v\n\n", err)
		ShowHelp()
		os.Exit(1)
	}

	CreateTables(config)
}

func main() {
	StartMigration()
}
