package system

import (
	"errors"
	"fmt"
	"strings"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/keyring"
	"github.com/julianstephens/daylit/daylit-cli/internal/storage/postgres"
)

// KeyringSetCmd stores database connection credentials in the OS keyring
type KeyringSetCmd struct {
	ConnectionString string `arg:"" help:"PostgreSQL connection string to store in keyring"`
}

func (cmd *KeyringSetCmd) Run(ctx *cli.Context) error {
	// Validate the connection string format
	if !strings.HasPrefix(cmd.ConnectionString, "postgres://") &&
		!strings.HasPrefix(cmd.ConnectionString, "postgresql://") &&
		!strings.Contains(cmd.ConnectionString, "host=") {
		return errors.New("connection string must be a valid PostgreSQL connection string")
	}

	// Validate connection string for security
	_, err := postgres.ValidateConnString(cmd.ConnectionString)
	if err != nil {
		if errors.Is(err, postgres.ErrEmbeddedCredentials) {
			// Warn about embedded credentials but allow storage in keyring (it's encrypted)
			fmt.Println("⚠️  Warning: Connection string contains embedded credentials.")
			fmt.Println("   It will be stored as-is in the encrypted OS keyring, which is a secure place for credentials.")
			fmt.Println("   If you prefer to keep passwords separate from connection strings, consider using .pgpass or environment variables instead.")
		} else {
			// Other validation errors should fail the operation
			return fmt.Errorf("invalid connection string: %w", err)
		}
	}

	// Store in keyring
	if err := keyring.SetConnectionString(cmd.ConnectionString); err != nil {
		return fmt.Errorf("failed to store connection string in keyring: %w", err)
	}

	fmt.Println("✓ Connection string stored successfully in OS keyring")
	fmt.Println("  You can now use daylit without the --config flag")
	return nil
}

// KeyringGetCmd retrieves database connection credentials from the OS keyring
type KeyringGetCmd struct{}

func (cmd *KeyringGetCmd) Run(ctx *cli.Context) error {
	connStr, err := keyring.GetConnectionString()
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return errors.New("no connection string found in keyring. Use 'daylit keyring set' to store one")
		}
		return fmt.Errorf("failed to retrieve connection string from keyring: %w", err)
	}

	fmt.Println("Connection string retrieved from keyring:")
	// Mask the password in the output for security
	maskedConnStr := maskPassword(connStr)
	fmt.Println(maskedConnStr)
	return nil
}

// KeyringDeleteCmd removes database connection credentials from the OS keyring
type KeyringDeleteCmd struct{}

func (cmd *KeyringDeleteCmd) Run(ctx *cli.Context) error {
	err := keyring.DeleteConnectionString()
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return errors.New("no connection string found in keyring")
		}
		return fmt.Errorf("failed to delete connection string from keyring: %w", err)
	}

	fmt.Println("✓ Connection string deleted from OS keyring")
	return nil
}

// KeyringStatusCmd checks the availability of the OS keyring
type KeyringStatusCmd struct{}

func (cmd *KeyringStatusCmd) Run(ctx *cli.Context) error {
	if keyring.IsAvailable() {
		fmt.Println("✓ OS keyring is available")

		// Check if credentials are stored
		_, err := keyring.GetConnectionString()
		if err == nil {
			fmt.Println("✓ Connection string is stored in keyring")
		} else if errors.Is(err, keyring.ErrNotFound) {
			fmt.Println("ℹ No connection string stored in keyring")
		}
	} else {
		fmt.Println("❌ OS keyring is not available on this system")
		return errors.New("keyring unavailable")
	}
	return nil
}

// maskPassword masks passwords in connection strings for display
func maskPassword(connStr string) string {
	// Handle URL format (postgres://user:password@host:port/db)
	if strings.HasPrefix(connStr, "postgres://") || strings.HasPrefix(connStr, "postgresql://") {
		// Find password section
		if idx := strings.Index(connStr, "://"); idx != -1 {
			remaining := connStr[idx+3:]
			// Find the last @ which separates user info from host
			if atIdx := strings.LastIndex(remaining, "@"); atIdx != -1 {
				userInfo := remaining[:atIdx]
				if colonIdx := strings.Index(userInfo, ":"); colonIdx != -1 {
					// Has password, mask it
					return connStr[:idx+3] + userInfo[:colonIdx] + ":****" + connStr[idx+3+atIdx:]
				}
			}
		}
	}

	// Handle DSN format (host=... user=... password=... dbname=...)
	if strings.Contains(connStr, "password=") {
		parts := strings.Fields(connStr)
		var masked []string
		for _, part := range parts {
			if strings.HasPrefix(part, "password=") {
				masked = append(masked, "password=****")
			} else {
				masked = append(masked, part)
			}
		}
		return strings.Join(masked, " ")
	}

	return connStr
}
