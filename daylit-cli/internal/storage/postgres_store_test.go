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
			name:     "sslmode in password",
			connStr:  "host=localhost password=sslmode123",
			expected: true, // This is expected behavior - we use contains for simplicity
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
