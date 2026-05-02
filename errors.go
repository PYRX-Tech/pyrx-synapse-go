package synapse

import "fmt"

// SynapseError is the base error for all Synapse API errors.
type SynapseError struct {
	Message   string
	Status    int
	Code      string
	RequestID string
}

func (e *SynapseError) Error() string { return e.Message }

// SynapseAuthError is raised on 401 Unauthorized or 403 Forbidden.
type SynapseAuthError struct {
	SynapseError
}

// SynapseRateLimitError is raised on 429 Too Many Requests.
type SynapseRateLimitError struct {
	SynapseError
	RetryAfter float64
}

// SynapsePlanLimitError is raised on 403 with code "plan_limit_reached".
type SynapsePlanLimitError struct {
	SynapseError
	LimitType string
	Current   int
	Maximum   int
	Plan      string
}

// SynapseValidationError is raised on 422 Unprocessable Entity.
type SynapseValidationError struct {
	SynapseError
	Errors []ValidationFieldError
}

// ValidationFieldError represents a single field validation error.
type ValidationFieldError struct {
	Field   string
	Message string
}

// MapError maps an HTTP status code and response body to the appropriate error type.
func MapError(status int, body map[string]interface{}, retryAfter float64) error {
	if body == nil {
		body = map[string]interface{}{}
	}

	message := extractMessage(body, status)
	code := stringFromMap(body, "code")
	requestID := stringFromMap(body, "request_id")

	base := SynapseError{
		Message:   message,
		Status:    status,
		Code:      code,
		RequestID: requestID,
	}

	if status == 429 {
		ra := retryAfter
		if ra <= 0 {
			ra = 60.0
		}
		return &SynapseRateLimitError{SynapseError: base, RetryAfter: ra}
	}

	if status == 403 && code == "plan_limit_reached" {
		return &SynapsePlanLimitError{
			SynapseError: base,
			LimitType:    stringFromMap(body, "limit_type"),
			Current:      intFromMap(body, "current"),
			Maximum:      intFromMap(body, "maximum"),
			Plan:         stringFromMap(body, "plan"),
		}
	}

	if status == 401 || status == 403 {
		return &SynapseAuthError{SynapseError: base}
	}

	if status == 422 {
		var fieldErrors []ValidationFieldError
		if rawErrors, ok := body["errors"]; ok {
			if arr, ok := rawErrors.([]interface{}); ok {
				for _, entry := range arr {
					if m, ok := entry.(map[string]interface{}); ok {
						msg := stringFromMap(m, "message")
						if msg == "" {
							msg = stringFromMap(m, "msg")
						}
						fieldErrors = append(fieldErrors, ValidationFieldError{
							Field:   stringFromMap(m, "field"),
							Message: msg,
						})
					}
				}
			}
		}
		return &SynapseValidationError{SynapseError: base, Errors: fieldErrors}
	}

	return &SynapseError{
		Message:   message,
		Status:    status,
		Code:      code,
		RequestID: requestID,
	}
}

func extractMessage(body map[string]interface{}, status int) string {
	if detail, ok := body["detail"]; ok {
		if s, ok := detail.(string); ok && s != "" {
			return s
		}
	}
	if msg, ok := body["message"]; ok {
		if s, ok := msg.(string); ok && s != "" {
			return s
		}
	}
	return fmt.Sprintf("HTTP %d", status)
}

func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intFromMap(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}
