package storage

import (
	"strings"
	"testing"
)

func TestHasSearchPathParam(t *testing.T) {
	tests := []struct {
		name     string
		connStr  string
		expected bool
	}{
		{
			name:     "empty string",
			connStr:  "",
			expected: false,
		},
		{
			name:     "no search_path",
			connStr:  "host=localhost port=5432 dbname=daylit user=postgres password=secret",
			expected: false,
		},
		{
			name:     "has search_path lowercase",
			connStr:  "host=localhost search_path=daylit dbname=daylit",
			expected: true,
		},
		{
			name:     "has search_path uppercase",
			connStr:  "host=localhost SEARCH_PATH=daylit dbname=daylit",
			expected: true,
		},
		{
			name:     "has search_path mixed case",
			connStr:  "host=localhost Search_Path=daylit dbname=daylit",
			expected: true,
		},
		{
			name:     "search_path in password should not match",
			connStr:  "host=localhost password=search_path_123 dbname=daylit",
			expected: false,
		},
		{
			name:     "search_path at start",
			connStr:  "search_path=public,daylit host=localhost",
			expected: true,
		},
		{
			name:     "search_path at end",
			connStr:  "host=localhost dbname=daylit search_path=daylit",
			expected: true,
		},
		{
			name:     "substring match should not trigger",
			connStr:  "host=localhost dbname=daylit_search_path",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasSearchPathParam(tt.connStr)
			if result != tt.expected {
				t.Errorf("hasSearchPathParam(%q) = %v, want %v", tt.connStr, result, tt.expected)
			}
		})
	}
}

func TestHasSSLMode(t *testing.T) {
	tests := []struct {
		name     string
		connStr  string
		expected bool
	}{
		{
			name:     "empty string",
			connStr:  "",
			expected: false,
		},
		{
			name:     "no sslmode",
			connStr:  "host=localhost port=5432 dbname=daylit",
			expected: false,
		},
		{
			name:     "has sslmode lowercase",
			connStr:  "host=localhost sslmode=disable",
			expected: true,
		},
		{
			name:     "has sslmode uppercase",
			connStr:  "host=localhost SSLMODE=disable",
			expected: true,
		},
		{
			name:     "has sslmode mixed case",
			connStr:  "host=localhost SslMode=disable",
			expected: true,
		},
		{
			name:     "has sslmode in URL format",
			connStr:  "postgres://user:pass@localhost/db?sslmode=disable",
			expected: true,
		},
		{
			name:     "sslmode in password should not match",
			connStr:  "host=localhost password=sslmode123",
			expected: false, // sslmode appearing only inside a value (e.g., password) should not be treated as an sslmode parameter
		},
		{
			name:     "sslmode in database name should not match",
			connStr:  "host=localhost dbname=test_sslmode",
			expected: false,
		},
		{
			name:     "URL with uppercase SSLMODE",
			connStr:  "postgres://localhost/db?SSLMODE=require",
			expected: true,
		},
		{
			name:     "URL with sslmode in password",
			connStr:  "postgres://user:sslmode@localhost/db",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasSSLMode(tt.connStr)
			if result != tt.expected {
				t.Errorf("hasSSLMode(%q) = %v, want %v", tt.connStr, result, tt.expected)
			}
		})
	}
}

func TestEnsureSearchPath(t *testing.T) {
	tests := []struct {
		name          string
		inputConnStr  string
		expectedMatch string // substring that should be present in result
	}{
		{
			name:          "URL format without search_path",
			inputConnStr:  "postgres://user:pass@localhost/db",
			expectedMatch: "search_path=daylit",
		},
		{
			name:          "URL format with existing search_path",
			inputConnStr:  "postgres://user:pass@localhost/db?search_path=public",
			expectedMatch: "search_path=public", // should not be modified
		},
		{
			name:          "DSN format without search_path",
			inputConnStr:  "host=localhost port=5432 dbname=daylit",
			expectedMatch: "search_path=daylit",
		},
		{
			name:          "DSN format with existing search_path",
			inputConnStr:  "host=localhost search_path=public dbname=daylit",
			expectedMatch: "search_path=public", // should not be modified
		},
		{
			name:          "PostgreSQL URL prefix",
			inputConnStr:  "postgresql://user:pass@localhost/db",
			expectedMatch: "search_path=daylit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewPostgresStore(tt.inputConnStr)
			store.ensureSearchPath()

			if !strings.Contains(store.connStr, tt.expectedMatch) {
				t.Errorf("ensureSearchPath() result %q does not contain expected substring %q", store.connStr, tt.expectedMatch)
			}
		})
	}
}

func TestHasEmbeddedCredentials(t *testing.T) {
	tests := []struct {
		name     string
		connStr  string
		expected bool
	}{
		// URL format tests
		{
			name:     "URL with username and password",
			connStr:  "postgres://user:pass@localhost:5432/daylit",
			expected: true,
		},
		{
			name:     "URL with username but no password",
			connStr:  "postgres://user@localhost:5432/daylit",
			expected: false,
		},
		{
			name:     "URL with no user info",
			connStr:  "postgres://localhost:5432/daylit",
			expected: false,
		},
		{
			name:     "URL with empty password",
			connStr:  "postgres://user:@localhost:5432/daylit",
			expected: false,
		},
		{
			name:     "URL with password and query params",
			connStr:  "postgres://user:pass@localhost:5432/daylit?sslmode=disable",
			expected: true,
		},
		{
			name:     "postgresql:// prefix with password",
			connStr:  "postgresql://user:secret@localhost/db",
			expected: true,
		},
		{
			name:     "postgresql:// prefix without password",
			connStr:  "postgresql://user@localhost/db",
			expected: false,
		},
		{
			name:     "URL with special characters in password",
			connStr:  "postgres://user:p@ssw0rd!@localhost:5432/daylit",
			expected: true,
		},
		{
			name:     "URL with encoded password",
			connStr:  "postgres://user:p%40ss@localhost:5432/daylit",
			expected: true,
		},
		// DSN format tests
		{
			name:     "DSN with password parameter",
			connStr:  "host=localhost port=5432 dbname=daylit user=postgres password=secret",
			expected: true,
		},
		{
			name:     "DSN with empty password parameter",
			connStr:  "host=localhost port=5432 dbname=daylit user=postgres password=",
			expected: false,
		},
		{
			name:     "DSN without password parameter",
			connStr:  "host=localhost port=5432 dbname=daylit user=postgres",
			expected: false,
		},
		{
			name:     "DSN with PASSWORD uppercase",
			connStr:  "host=localhost PASSWORD=secret dbname=daylit",
			expected: true,
		},
		{
			name:     "DSN with Password mixed case",
			connStr:  "host=localhost Password=secret dbname=daylit",
			expected: true,
		},
		{
			name:     "DSN with service name (no password)",
			connStr:  "service=daylit",
			expected: false,
		},
		// Edge cases
		{
			name:     "Empty string",
			connStr:  "",
			expected: false,
		},
		{
			name:     "Invalid URL format",
			connStr:  "postgres://invalid@@@url",
			expected: false, // url.Parse succeeds but doesn't find valid password
		},
		{
			name:     "Plain text (not URL or DSN)",
			connStr:  "some random text password=hidden",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasEmbeddedCredentials(tt.connStr)
			if result != tt.expected {
				t.Errorf("HasEmbeddedCredentials(%q) = %v, want %v", tt.connStr, result, tt.expected)
			}
		})
	}
}
