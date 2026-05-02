package synapse

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Error hierarchy (type assertions)
// ---------------------------------------------------------------------------

func TestErrorHierarchy_SynapseError(t *testing.T) {
	var err error = &SynapseError{Message: "boom", Status: 500}
	if err.Error() != "boom" {
		t.Errorf("expected boom, got %s", err.Error())
	}
}

func TestErrorHierarchy_AuthError(t *testing.T) {
	err := &SynapseAuthError{SynapseError: SynapseError{Message: "auth", Status: 401}}
	// AuthError embeds SynapseError, so it has all its fields and methods
	if err.Error() != "auth" {
		t.Errorf("expected auth, got %s", err.Error())
	}
	if err.Status != 401 {
		t.Errorf("expected 401, got %d", err.Status)
	}
	// Verify it satisfies the error interface
	var _ error = err
}

func TestErrorHierarchy_RateLimitError(t *testing.T) {
	err := &SynapseRateLimitError{SynapseError: SynapseError{Message: "slow", Status: 429}}
	if err.Error() != "slow" {
		t.Errorf("expected slow, got %s", err.Error())
	}
	if err.Status != 429 {
		t.Errorf("expected 429, got %d", err.Status)
	}
	var _ error = err
}

func TestErrorHierarchy_PlanLimitError(t *testing.T) {
	err := &SynapsePlanLimitError{SynapseError: SynapseError{Message: "limit", Status: 403}}
	if err.Error() != "limit" {
		t.Errorf("expected limit, got %s", err.Error())
	}
	if err.Status != 403 {
		t.Errorf("expected 403, got %d", err.Status)
	}
	var _ error = err
}

func TestErrorHierarchy_ValidationError(t *testing.T) {
	err := &SynapseValidationError{SynapseError: SynapseError{Message: "invalid", Status: 422}}
	if err.Error() != "invalid" {
		t.Errorf("expected invalid, got %s", err.Error())
	}
	if err.Status != 422 {
		t.Errorf("expected 422, got %d", err.Status)
	}
	var _ error = err
}

// ---------------------------------------------------------------------------
// SynapseError attributes
// ---------------------------------------------------------------------------

func TestSynapseError_Attributes(t *testing.T) {
	err := &SynapseError{Message: "boom", Status: 500, Code: "internal", RequestID: "req_1"}
	if err.Error() != "boom" {
		t.Errorf("expected boom, got %s", err.Error())
	}
	if err.Status != 500 {
		t.Errorf("expected 500, got %d", err.Status)
	}
	if err.Code != "internal" {
		t.Errorf("expected internal, got %s", err.Code)
	}
	if err.RequestID != "req_1" {
		t.Errorf("expected req_1, got %s", err.RequestID)
	}
}

// ---------------------------------------------------------------------------
// SynapseRateLimitError
// ---------------------------------------------------------------------------

func TestRateLimitError_DefaultRetryAfter(t *testing.T) {
	err := MapError(429, map[string]interface{}{"message": "slow down"}, 0)
	rle, ok := err.(*SynapseRateLimitError)
	if !ok {
		t.Fatalf("expected SynapseRateLimitError, got %T", err)
	}
	if rle.RetryAfter != 60.0 {
		t.Errorf("expected default 60.0, got %f", rle.RetryAfter)
	}
}

func TestRateLimitError_CustomRetryAfter(t *testing.T) {
	err := MapError(429, map[string]interface{}{"message": "slow down"}, 120.0)
	rle, ok := err.(*SynapseRateLimitError)
	if !ok {
		t.Fatalf("expected SynapseRateLimitError, got %T", err)
	}
	if rle.RetryAfter != 120.0 {
		t.Errorf("expected 120.0, got %f", rle.RetryAfter)
	}
}

// ---------------------------------------------------------------------------
// SynapsePlanLimitError
// ---------------------------------------------------------------------------

func TestPlanLimitError_Attributes(t *testing.T) {
	body := map[string]interface{}{
		"message":    "limit reached",
		"code":       "plan_limit_reached",
		"limit_type": "contacts",
		"current":    float64(1000),
		"maximum":    float64(1000),
		"plan":       "starter",
	}
	err := MapError(403, body, 0)
	ple, ok := err.(*SynapsePlanLimitError)
	if !ok {
		t.Fatalf("expected SynapsePlanLimitError, got %T", err)
	}
	if ple.LimitType != "contacts" {
		t.Errorf("expected contacts, got %s", ple.LimitType)
	}
	if ple.Current != 1000 {
		t.Errorf("expected 1000, got %d", ple.Current)
	}
	if ple.Maximum != 1000 {
		t.Errorf("expected 1000, got %d", ple.Maximum)
	}
	if ple.Plan != "starter" {
		t.Errorf("expected starter, got %s", ple.Plan)
	}
}

// ---------------------------------------------------------------------------
// SynapseValidationError
// ---------------------------------------------------------------------------

func TestValidationError_Errors(t *testing.T) {
	body := map[string]interface{}{
		"message": "invalid",
		"errors": []interface{}{
			map[string]interface{}{"field": "email", "message": "is required"},
		},
	}
	err := MapError(422, body, 0)
	ve, ok := err.(*SynapseValidationError)
	if !ok {
		t.Fatalf("expected SynapseValidationError, got %T", err)
	}
	if len(ve.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(ve.Errors))
	}
	if ve.Errors[0].Field != "email" {
		t.Errorf("expected email, got %s", ve.Errors[0].Field)
	}
	if ve.Errors[0].Message != "is required" {
		t.Errorf("expected is required, got %s", ve.Errors[0].Message)
	}
}

func TestValidationError_EmptyErrors(t *testing.T) {
	body := map[string]interface{}{"message": "invalid"}
	err := MapError(422, body, 0)
	ve, ok := err.(*SynapseValidationError)
	if !ok {
		t.Fatalf("expected SynapseValidationError, got %T", err)
	}
	if ve.Errors != nil && len(ve.Errors) != 0 {
		t.Errorf("expected nil or empty errors, got %d", len(ve.Errors))
	}
}

// ---------------------------------------------------------------------------
// MapError routing
// ---------------------------------------------------------------------------

func TestMapError_429(t *testing.T) {
	err := MapError(429, map[string]interface{}{"message": "too fast"}, 30.0)
	rle, ok := err.(*SynapseRateLimitError)
	if !ok {
		t.Fatalf("expected SynapseRateLimitError, got %T", err)
	}
	if rle.RetryAfter != 30.0 {
		t.Errorf("expected 30.0, got %f", rle.RetryAfter)
	}
	if rle.Message != "too fast" {
		t.Errorf("expected too fast, got %s", rle.Message)
	}
}

func TestMapError_429_DefaultRetryAfter(t *testing.T) {
	err := MapError(429, map[string]interface{}{"message": "too fast"}, 0)
	rle := err.(*SynapseRateLimitError)
	if rle.RetryAfter != 60.0 {
		t.Errorf("expected 60.0, got %f", rle.RetryAfter)
	}
}

func TestMapError_403_PlanLimit(t *testing.T) {
	body := map[string]interface{}{
		"message":    "Contact limit reached",
		"code":       "plan_limit_reached",
		"limit_type": "contacts",
		"current":    float64(500),
		"maximum":    float64(500),
		"plan":       "free",
	}
	err := MapError(403, body, 0)
	ple, ok := err.(*SynapsePlanLimitError)
	if !ok {
		t.Fatalf("expected SynapsePlanLimitError, got %T", err)
	}
	if ple.LimitType != "contacts" {
		t.Errorf("expected contacts, got %s", ple.LimitType)
	}
	if ple.Current != 500 {
		t.Errorf("expected 500, got %d", ple.Current)
	}
	if ple.Maximum != 500 {
		t.Errorf("expected 500, got %d", ple.Maximum)
	}
	if ple.Plan != "free" {
		t.Errorf("expected free, got %s", ple.Plan)
	}
}

func TestMapError_403_WithoutPlanLimit(t *testing.T) {
	err := MapError(403, map[string]interface{}{"message": "forbidden"}, 0)
	_, ok := err.(*SynapseAuthError)
	if !ok {
		t.Fatalf("expected SynapseAuthError, got %T", err)
	}
}

func TestMapError_401(t *testing.T) {
	err := MapError(401, map[string]interface{}{"message": "unauthorized"}, 0)
	ae, ok := err.(*SynapseAuthError)
	if !ok {
		t.Fatalf("expected SynapseAuthError, got %T", err)
	}
	if ae.Message != "unauthorized" {
		t.Errorf("expected unauthorized, got %s", ae.Message)
	}
}

func TestMapError_422(t *testing.T) {
	body := map[string]interface{}{
		"message": "Validation failed",
		"errors": []interface{}{
			map[string]interface{}{"field": "email", "message": "is invalid"},
			map[string]interface{}{"field": "name", "msg": "is required"},
		},
	}
	err := MapError(422, body, 0)
	ve, ok := err.(*SynapseValidationError)
	if !ok {
		t.Fatalf("expected SynapseValidationError, got %T", err)
	}
	if len(ve.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(ve.Errors))
	}
	if ve.Errors[0].Field != "email" {
		t.Errorf("expected email, got %s", ve.Errors[0].Field)
	}
	if ve.Errors[0].Message != "is invalid" {
		t.Errorf("expected is invalid, got %s", ve.Errors[0].Message)
	}
	if ve.Errors[1].Message != "is required" {
		t.Errorf("expected is required, got %s", ve.Errors[1].Message)
	}
}

func TestMapError_OtherStatus(t *testing.T) {
	err := MapError(500, map[string]interface{}{"detail": "server error"}, 0)
	se, ok := err.(*SynapseError)
	if !ok {
		t.Fatalf("expected SynapseError, got %T", err)
	}
	// Should NOT be an auth error
	if _, isAuth := err.(*SynapseAuthError); isAuth {
		t.Error("should not be SynapseAuthError")
	}
	if se.Message != "server error" {
		t.Errorf("expected server error, got %s", se.Message)
	}
}

func TestMapError_DetailOverMessage(t *testing.T) {
	err := MapError(500, map[string]interface{}{"detail": "detail msg", "message": "message msg"}, 0)
	se := err.(*SynapseError)
	if se.Message != "detail msg" {
		t.Errorf("expected detail msg, got %s", se.Message)
	}
}

func TestMapError_FallbackToHTTPStatus(t *testing.T) {
	err := MapError(503, map[string]interface{}{}, 0)
	se := err.(*SynapseError)
	if se.Message != "HTTP 503" {
		t.Errorf("expected HTTP 503, got %s", se.Message)
	}
}

func TestMapError_ExtractsCodeAndRequestID(t *testing.T) {
	body := map[string]interface{}{
		"message":    "err",
		"code":       "some_code",
		"request_id": "req_abc",
	}
	err := MapError(400, body, 0)
	se := err.(*SynapseError)
	if se.Code != "some_code" {
		t.Errorf("expected some_code, got %s", se.Code)
	}
	if se.RequestID != "req_abc" {
		t.Errorf("expected req_abc, got %s", se.RequestID)
	}
}
