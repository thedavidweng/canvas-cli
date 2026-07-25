package output

// Exit codes matching the taxonomy in docs/json-contract.md.
const (
	CodeSuccess          = 0
	CodeGenericError     = 1
	CodeValidationError  = 2
	CodeAuthError        = 3
	CodePermissionDenied = 4
	CodeRateLimit        = 5
	CodeNetworkError     = 6
	CodeSafetyBlocked    = 7
	CodePartialFailure   = 8
)

// ExitCodeForCategory maps a JSON error category to a process exit code.
// Unknown categories map to CodeGenericError.
func ExitCodeForCategory(category string) int {
	switch category {
	case "validation":
		return CodeValidationError
	case "auth":
		return CodeAuthError
	case "permission":
		return CodePermissionDenied
	case "rate_limit":
		return CodeRateLimit
	case "network":
		return CodeNetworkError
	case "partial_failure":
		return CodePartialFailure
	default:
		return CodeGenericError
	}
}
