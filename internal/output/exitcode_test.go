package output

import "testing"

func TestExitCodes(t *testing.T) {
	tests := []struct {
		name string
		code int
		want int
	}{
		{"CodeSuccess", CodeSuccess, 0},
		{"CodeGenericError", CodeGenericError, 1},
		{"CodeValidationError", CodeValidationError, 2},
		{"CodeAuthError", CodeAuthError, 3},
		{"CodePermissionDenied", CodePermissionDenied, 4},
		{"CodeRateLimit", CodeRateLimit, 5},
		{"CodeNetworkError", CodeNetworkError, 6},
		{"CodeSafetyBlocked", CodeSafetyBlocked, 7},
		{"CodePartialFailure", CodePartialFailure, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.want)
			}
		})
	}
}

func TestExitCodeForCategory(t *testing.T) {
	tests := []struct {
		category string
		want     int
	}{
		{"validation", CodeValidationError},
		{"auth", CodeAuthError},
		{"permission", CodePermissionDenied},
		{"rate_limit", CodeRateLimit},
		{"network", CodeNetworkError},
		{"partial_failure", CodePartialFailure},
		{"api", CodeGenericError},
		{"not_found", CodeGenericError},
		{"server", CodeGenericError},
		{"", CodeGenericError},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			if got := ExitCodeForCategory(tt.category); got != tt.want {
				t.Errorf("ExitCodeForCategory(%q) = %d, want %d", tt.category, got, tt.want)
			}
		})
	}
}
