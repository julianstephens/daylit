package keyring

import (
	"errors"
	"fmt"

	"github.com/julianstephens/daylit/daylit-cli/internal/constants"
	"github.com/zalando/go-keyring"
)

var (
	// ErrNotFound is returned when no credentials are found in the keyring
	ErrNotFound = errors.New("credentials not found in keyring")
	// ErrKeyringUnavailable is returned when the OS keyring is not available
	ErrKeyringUnavailable = errors.New("OS keyring is not available")
)

// GetConnectionString retrieves the database connection string from the OS keyring.
// Returns ErrNotFound if no credentials are stored.
func GetConnectionString() (string, error) {
	connStr, err := keyring.Get(constants.AppName, constants.DefaultKeyringUser)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", ErrNotFound
		}
		// Wrap other keyring errors as unavailable
		return "", fmt.Errorf("%w: %v", ErrKeyringUnavailable, err)
	}
	return connStr, nil
}

// SetConnectionString stores the database connection string in the OS keyring.
func SetConnectionString(connStr string) error {
	if connStr == "" {
		return errors.New("connection string cannot be empty")
	}
	err := keyring.Set(constants.AppName, constants.DefaultKeyringUser, connStr)
	if err != nil {
		return fmt.Errorf("failed to store credentials in keyring: %w", err)
	}
	return nil
}

// DeleteConnectionString removes the database connection string from the OS keyring.
func DeleteConnectionString() error {
	err := keyring.Delete(constants.AppName, constants.DefaultKeyringUser)
	if err != nil {
		if err == keyring.ErrNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("failed to delete credentials from keyring: %w", err)
	}
	return nil
}

// IsAvailable checks if the OS keyring is available on the current system.
// This is a best-effort check and may not catch all failure scenarios.
func IsAvailable() bool {
	// Try to perform a read operation to test availability
	// We don't care about the result, just whether the operation succeeds or fails
	_, err := keyring.Get(constants.AppName, "test-availability")
	// If the error is ErrNotFound, the keyring is available but empty
	// Any other error likely indicates the keyring is not available
	return err == nil || err == keyring.ErrNotFound
}
