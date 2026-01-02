package system

import (
	"strings"
	"testing"

	"github.com/julianstephens/daylit/daylit-cli/internal/cli"
	"github.com/julianstephens/daylit/daylit-cli/internal/keyring"
	gokeyring "github.com/zalando/go-keyring"
)

func TestKeyringSetCmd(t *testing.T) {
	gokeyring.MockInit()
	defer func() { _ = keyring.DeleteConnectionString() }()

	tests := []struct {
		name      string
		connStr   string
		wantError bool
	}{
		{
			name:      "valid postgres URL",
			connStr:   "postgres://user@localhost:5432/daylit?sslmode=disable",
			wantError: false,
		},
		{
			name:      "valid postgresql URL",
			connStr:   "postgresql://user@localhost:5432/daylit",
			wantError: false,
		},
		{
			name:      "valid DSN format",
			connStr:   "host=localhost port=5432 dbname=daylit user=testuser",
			wantError: false,
		},
		{
			name:      "invalid connection string",
			connStr:   "not-a-valid-connection-string",
			wantError: true,
		},
		{
			name:      "postgres URL with password (warning but succeeds)",
			connStr:   "postgres://user:password@localhost:5432/daylit",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &KeyringSetCmd{
				ConnectionString: tt.connStr,
			}
			ctx := &cli.Context{}

			err := cmd.Run(ctx)
			if (err != nil) != tt.wantError {
				t.Errorf("KeyringSetCmd.Run() error = %v, wantError %v", err, tt.wantError)
			}

			// If successful, verify it was stored
			if err == nil {
				stored, getErr := keyring.GetConnectionString()
				if getErr != nil {
					t.Errorf("Failed to retrieve stored connection string: %v", getErr)
				}
				if stored != tt.connStr {
					t.Errorf("Stored connection string = %q, want %q", stored, tt.connStr)
				}
			}
		})
	}
}

func TestKeyringGetCmd(t *testing.T) {
	gokeyring.MockInit()
	defer func() { _ = keyring.DeleteConnectionString() }()

	testConnStr := "postgres://user@localhost:5432/daylit"

	// Test when nothing is stored
	t.Run("not found", func(t *testing.T) {
		_ = keyring.DeleteConnectionString()
		cmd := &KeyringGetCmd{}
		ctx := &cli.Context{}

		err := cmd.Run(ctx)
		if err == nil {
			t.Error("KeyringGetCmd.Run() should return error when no credentials stored")
		}
	})

	// Test when credentials are stored
	t.Run("found", func(t *testing.T) {
		err := keyring.SetConnectionString(testConnStr)
		if err != nil {
			t.Fatalf("Failed to set connection string: %v", err)
		}

		cmd := &KeyringGetCmd{}
		ctx := &cli.Context{}

		err = cmd.Run(ctx)
		if err != nil {
			t.Errorf("KeyringGetCmd.Run() error = %v, want nil", err)
		}
	})
}

func TestKeyringDeleteCmd(t *testing.T) {
	gokeyring.MockInit()
	defer func() { _ = keyring.DeleteConnectionString() }()

	testConnStr := "postgres://user@localhost:5432/daylit"

	// Test deleting when nothing is stored
	t.Run("not found", func(t *testing.T) {
		_ = keyring.DeleteConnectionString()
		cmd := &KeyringDeleteCmd{}
		ctx := &cli.Context{}

		err := cmd.Run(ctx)
		if err == nil {
			t.Error("KeyringDeleteCmd.Run() should return error when no credentials stored")
		}
	})

	// Test deleting stored credentials
	t.Run("delete success", func(t *testing.T) {
		err := keyring.SetConnectionString(testConnStr)
		if err != nil {
			t.Fatalf("Failed to set connection string: %v", err)
		}

		cmd := &KeyringDeleteCmd{}
		ctx := &cli.Context{}

		err = cmd.Run(ctx)
		if err != nil {
			t.Errorf("KeyringDeleteCmd.Run() error = %v, want nil", err)
		}

		// Verify it's deleted
		_, err = keyring.GetConnectionString()
		if err != keyring.ErrNotFound {
			t.Error("Connection string should be deleted from keyring")
		}
	})
}

func TestKeyringStatusCmd(t *testing.T) {
	gokeyring.MockInit()

	cmd := &KeyringStatusCmd{}
	ctx := &cli.Context{}

	err := cmd.Run(ctx)
	if err != nil {
		t.Errorf("KeyringStatusCmd.Run() error = %v, want nil", err)
	}
}

func TestMaskPassword(t *testing.T) {
	tests := []struct {
		name     string
		connStr  string
		expected string
	}{
		{
			name:     "URL with password",
			connStr:  "postgres://user:secret123@localhost:5432/daylit",
			expected: "postgres://user:****@localhost:5432/daylit",
		},
		{
			name:     "URL without password",
			connStr:  "postgres://user@localhost:5432/daylit",
			expected: "postgres://user@localhost:5432/daylit",
		},
		{
			name:     "DSN with password",
			connStr:  "host=localhost port=5432 user=test password=secret dbname=daylit",
			expected: "host=localhost port=5432 user=test password=**** dbname=daylit",
		},
		{
			name:     "DSN without password",
			connStr:  "host=localhost port=5432 user=test dbname=daylit",
			expected: "host=localhost port=5432 user=test dbname=daylit",
		},
		{
			name:     "postgresql URL with password",
			connStr:  "postgresql://admin:p@ssw0rd@db.example.com:5432/mydb",
			expected: "postgresql://admin:****@db.example.com:5432/mydb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskPassword(tt.connStr)
			// Normalize whitespace for DSN comparisons
			normalizedResult := strings.Join(strings.Fields(result), " ")
			normalizedExpected := strings.Join(strings.Fields(tt.expected), " ")

			if normalizedResult != normalizedExpected {
				t.Errorf("maskPassword() = %q, want %q", result, tt.expected)
			}
		})
	}
}
