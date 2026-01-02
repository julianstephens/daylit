package postgres

import (
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
			connStr:  "host=localhost search_path=public,daylit",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasSearchPathParam(tt.connStr); got != tt.expected {
				t.Errorf("hasSearchPathParam() = %v, want %v", got, tt.expected)
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
			connStr:  "postgres://user:pass@localhost:5432/db",
			expected: false,
		},
		{
			name:     "sslmode in URL query",
			connStr:  "postgres://user:pass@localhost:5432/db?sslmode=disable",
			expected: true,
		},
		{
			name:     "sslmode in URL query mixed case",
			connStr:  "postgres://user:pass@localhost:5432/db?SSLMODE=require",
			expected: true,
		},
		{
			name:     "sslmode in DSN",
			connStr:  "host=localhost user=user dbname=db sslmode=disable",
			expected: true,
		},
		{
			name:     "sslmode in DSN mixed case",
			connStr:  "host=localhost user=user dbname=db SSLMODE=verify-full",
			expected: true,
		},
		{
			name:     "sslmode as value not key",
			connStr:  "host=localhost user=sslmode dbname=db",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasSSLMode(tt.connStr); got != tt.expected {
				t.Errorf("hasSSLMode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestValidateConnString(t *testing.T) {
	tests := []struct {
		name      string
		connStr   string
		wantValid bool
		wantErr   bool
	}{
		{
			name:      "valid URL",
			connStr:   "postgres://user@localhost:5432/db?sslmode=disable",
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "valid DSN",
			connStr:   "host=localhost user=user dbname=db sslmode=disable",
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "URL with password",
			connStr:   "postgres://user:password@localhost:5432/db",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "DSN with password",
			connStr:   "host=localhost user=user password=secret dbname=db",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "empty string",
			connStr:   "",
			wantValid: false,
			wantErr:   true,
		},
		{
			name:      "invalid URL format",
			connStr:   "://invalid",
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := ValidateConnString(tt.connStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConnString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if valid != tt.wantValid {
				t.Errorf("ValidateConnString() = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}
