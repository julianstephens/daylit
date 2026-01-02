package keyring

import (
	"testing"

	gokeyring "github.com/zalando/go-keyring"
)

func TestSetAndGetConnectionString(t *testing.T) {
	// Use mock keyring for testing
	gokeyring.MockInit()

	testConnStr := "postgres://testuser@localhost:5432/testdb?sslmode=disable"

	// Test Set
	err := SetConnectionString(testConnStr)
	if err != nil {
		t.Fatalf("SetConnectionString() failed: %v", err)
	}

	// Test Get
	retrieved, err := GetConnectionString()
	if err != nil {
		t.Fatalf("GetConnectionString() failed: %v", err)
	}

	if retrieved != testConnStr {
		t.Errorf("GetConnectionString() = %q, want %q", retrieved, testConnStr)
	}
}

func TestSetConnectionStringEmpty(t *testing.T) {
	gokeyring.MockInit()

	err := SetConnectionString("")
	if err == nil {
		t.Error("SetConnectionString(\"\") should return an error")
	}
}

func TestGetConnectionStringNotFound(t *testing.T) {
	gokeyring.MockInit()

	// Ensure nothing is stored
	_ = DeleteConnectionString()

	_, err := GetConnectionString()
	if err != ErrNotFound {
		t.Errorf("GetConnectionString() error = %v, want %v", err, ErrNotFound)
	}
}

func TestDeleteConnectionString(t *testing.T) {
	gokeyring.MockInit()

	testConnStr := "postgres://testuser@localhost:5432/testdb"

	// First, set a connection string
	err := SetConnectionString(testConnStr)
	if err != nil {
		t.Fatalf("SetConnectionString() failed: %v", err)
	}

	// Delete it
	err = DeleteConnectionString()
	if err != nil {
		t.Fatalf("DeleteConnectionString() failed: %v", err)
	}

	// Verify it's gone
	_, err = GetConnectionString()
	if err != ErrNotFound {
		t.Errorf("After DeleteConnectionString(), GetConnectionString() error = %v, want %v", err, ErrNotFound)
	}
}

func TestDeleteConnectionStringNotFound(t *testing.T) {
	gokeyring.MockInit()

	// Ensure nothing is stored
	_ = DeleteConnectionString()

	err := DeleteConnectionString()
	if err != ErrNotFound {
		t.Errorf("DeleteConnectionString() error = %v, want %v", err, ErrNotFound)
	}
}

func TestIsAvailable(t *testing.T) {
	gokeyring.MockInit()

	available := IsAvailable()
	// In mock mode, keyring should be available
	if !available {
		t.Error("IsAvailable() = false, want true in mock mode")
	}
}
