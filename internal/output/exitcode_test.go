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
		{"CodeInterrupted", CodeInterrupted, 130},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.want)
			}
		})
	}
}
