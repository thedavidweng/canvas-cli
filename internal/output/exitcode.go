package output

// Exit codes matching the JSON contract in docs/json-contract.md.
// All codes are defined here for the contract spec and external consumers,
// even if not all are currently referenced in production code.
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
	CodeInterrupted      = 130
)
