package synapse

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testAPIKey      = "psk_test_abc123"
	testWorkspaceID = "ws_test_1"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := NewClient(Config{
		APIKey:      testAPIKey,
		WorkspaceID: testWorkspaceID,
		BaseURL:     server.URL,
		Timeout:     5 * time.Second,
		MaxRetries:  1,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	// Disable real sleeps in tests
	client.sleepFunc = func(d time.Duration) {}
	return client
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

func TestNewClient_EmptyAPIKey(t *testing.T) {
	_, err := NewClient(Config{APIKey: "", WorkspaceID: "ws_1"})
	if err == nil {
		t.Fatal("expected error for empty api_key")
	}
	if !strings.Contains(err.Error(), "api_key") {
		t.Errorf("expected error to mention api_key, got: %s", err.Error())
	}
}

func TestNewClient_EmptyWorkspaceID(t *testing.T) {
	_, err := NewClient(Config{APIKey: "key", WorkspaceID: ""})
	if err == nil {
		t.Fatal("expected error for empty workspace_id")
	}
	if !strings.Contains(err.Error(), "workspace_id") {
		t.Errorf("expected error to mention workspace_id, got: %s", err.Error())
	}
}

func TestNewClient_DetectsTestEnvironment(t *testing.T) {
	c, _ := NewClient(Config{APIKey: "psk_test_abc", WorkspaceID: "ws_1"})
	if c.Environment != "test" {
		t.Errorf("expected test, got %s", c.Environment)
	}
}

func TestNewClient_DetectsLiveEnvironment(t *testing.T) {
	c, _ := NewClient(Config{APIKey: "psk_live_abc", WorkspaceID: "ws_1"})
	if c.Environment != "live" {
		t.Errorf("expected live, got %s", c.Environment)
	}
}

func TestNewClient_DetectsUnknownEnvironment(t *testing.T) {
	c, _ := NewClient(Config{APIKey: "custom_key", WorkspaceID: "ws_1"})
	if c.Environment != "unknown" {
		t.Errorf("expected unknown, got %s", c.Environment)
	}
}

func TestNewClient_StripsTrailingSlash(t *testing.T) {
	c, _ := NewClient(Config{APIKey: "key", WorkspaceID: "ws_1", BaseURL: "https://api.test.com/"})
	if c.BaseURL() != "https://api.test.com" {
		t.Errorf("expected trailing slash stripped, got %s", c.BaseURL())
	}
}

func TestNewClient_ExposesSubClients(t *testing.T) {
	c, _ := NewClient(Config{APIKey: "key", WorkspaceID: "ws_1"})
	if c.Contacts == nil {
		t.Error("expected Contacts sub-client to be set")
	}
	if c.Templates == nil {
		t.Error("expected Templates sub-client to be set")
	}
}

// ---------------------------------------------------------------------------
// Headers
// ---------------------------------------------------------------------------

func TestRequestHeaders(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Workspace-Id") != testWorkspaceID {
			t.Errorf("expected X-Workspace-Id %s, got %s", testWorkspaceID, r.Header.Get("X-Workspace-Id"))
		}
		if r.Header.Get("X-Api-Key") != testAPIKey {
			t.Errorf("expected X-Api-Key %s, got %s", testAPIKey, r.Header.Get("X-Api-Key"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		expected := "pyrx-synapse-go/" + Version
		if r.Header.Get("User-Agent") != expected {
			t.Errorf("expected User-Agent %s, got %s", expected, r.Header.Get("User-Agent"))
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]interface{}{"event_id": "e_1", "status": "ok"})
	})

	client.Track(TrackParams{ExternalID: "u1", EventName: "test"})
}

// ---------------------------------------------------------------------------
// Track
// ---------------------------------------------------------------------------

func TestTrack(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/events" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"event_id": "evt_1", "status": "accepted"})
	})

	result, err := client.Track(TrackParams{ExternalID: "user_1", EventName: "purchase"})
	if err != nil {
		t.Fatalf("Track: %v", err)
	}
	if result.EventID != "evt_1" {
		t.Errorf("expected evt_1, got %s", result.EventID)
	}
	if result.Status != "accepted" {
		t.Errorf("expected accepted, got %s", result.Status)
	}
}

func TestTrack_OnlySendsNonEmptyFields(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]interface{}
		json.Unmarshal(body, &parsed)

		if _, ok := parsed["attributes"]; ok {
			t.Error("attributes should not be sent when empty")
		}
		if _, ok := parsed["contact"]; ok {
			t.Error("contact should not be sent when empty")
		}
		if _, ok := parsed["external_id"]; !ok {
			t.Error("external_id should be sent")
		}
		if _, ok := parsed["event_name"]; !ok {
			t.Error("event_name should be sent")
		}

		json.NewEncoder(w).Encode(map[string]interface{}{"event_id": "e1", "status": "ok"})
	})

	client.Track(TrackParams{ExternalID: "u1", EventName: "click"})
}

func TestTrack_IncludesOptionalFields(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var parsed map[string]interface{}
		json.Unmarshal(body, &parsed)

		attrs, ok := parsed["attributes"].(map[string]interface{})
		if !ok || attrs["amount"] != float64(99) {
			t.Error("expected attributes.amount = 99")
		}
		if parsed["idempotency_key"] != "idk_1" {
			t.Error("expected idempotency_key = idk_1")
		}
		if parsed["occurred_at"] != "2026-01-01T00:00:00Z" {
			t.Error("expected occurred_at = 2026-01-01T00:00:00Z")
		}

		json.NewEncoder(w).Encode(map[string]interface{}{"event_id": "e1", "status": "ok"})
	})

	client.Track(TrackParams{
		ExternalID:     "u1",
		EventName:      "purchase",
		Attributes:     map[string]interface{}{"amount": 99},
		IdempotencyKey: "idk_1",
		OccurredAt:     "2026-01-01T00:00:00Z",
	})
}

// ---------------------------------------------------------------------------
// Track Batch
// ---------------------------------------------------------------------------

func TestTrackBatch(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/events/batch" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"accepted": 2, "rejected": 0})
	})

	result, err := client.TrackBatch(TrackBatchParams{
		Events: []TrackParams{
			{ExternalID: "u1", EventName: "click"},
			{ExternalID: "u2", EventName: "view"},
		},
	})
	if err != nil {
		t.Fatalf("TrackBatch: %v", err)
	}
	if result.Accepted != 2 {
		t.Errorf("expected 2 accepted, got %d", result.Accepted)
	}
	if result.Rejected != 0 {
		t.Errorf("expected 0 rejected, got %d", result.Rejected)
	}
}

// ---------------------------------------------------------------------------
// Identify
// ---------------------------------------------------------------------------

func TestIdentify(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/contacts" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "c_1", "external_id": "u_1", "email": "a@b.com",
			"created_at": "2026-01-01", "updated_at": "2026-01-01",
		})
	})

	result, err := client.Identify(IdentifyParams{ExternalID: "u_1", Email: "a@b.com"})
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if result.ID != "c_1" {
		t.Errorf("expected c_1, got %s", result.ID)
	}
	if result.ExternalID != "u_1" {
		t.Errorf("expected u_1, got %s", result.ExternalID)
	}
	if result.Email != "a@b.com" {
		t.Errorf("expected a@b.com, got %s", result.Email)
	}
}

// ---------------------------------------------------------------------------
// Identify Batch
// ---------------------------------------------------------------------------

func TestIdentifyBatch(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/contacts/bulk" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total": 2, "created": 1, "updated": 1, "skipped": 0, "errors": []interface{}{},
		})
	})

	result, err := client.IdentifyBatch(IdentifyBatchParams{
		Contacts: []IdentifyParams{{ExternalID: "u1"}},
	})
	if err != nil {
		t.Fatalf("IdentifyBatch: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Total)
	}
	if result.Created != 1 {
		t.Errorf("expected created 1, got %d", result.Created)
	}
}

// ---------------------------------------------------------------------------
// Send Email
// ---------------------------------------------------------------------------

func TestSend(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/send" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"email_log_id": "el_1", "status": "queued"})
	})

	result, err := client.Send(SendParams{
		TemplateSlug: "welcome",
		To:           map[string]interface{}{"email": "a@b.com"},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if result.EmailLogID != "el_1" {
		t.Errorf("expected el_1, got %s", result.EmailLogID)
	}
	if result.Status != "queued" {
		t.Errorf("expected queued, got %s", result.Status)
	}
}

// ---------------------------------------------------------------------------
// Contacts sub-client
// ---------------------------------------------------------------------------

func TestContacts_List(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/contacts" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("expected page=2, got %s", r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("per_page") != "10" {
			t.Errorf("expected per_page=10, got %s", r.URL.Query().Get("per_page"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{map[string]interface{}{"id": "c_1", "external_id": "u_1"}},
			"meta": map[string]interface{}{"total": 1, "page": 2, "per_page": 10, "total_pages": 1},
		})
	})

	result, err := client.Contacts.List(ContactListParams{Page: 2, PerPage: 10})
	if err != nil {
		t.Fatalf("Contacts.List: %v", err)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 contact, got %d", len(result.Data))
	}
	if result.Meta.Page != 2 {
		t.Errorf("expected page 2, got %d", result.Meta.Page)
	}
}

func TestContacts_Get(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/contacts/c_1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "c_1", "external_id": "u_1"})
	})

	result, err := client.Contacts.Get("c_1")
	if err != nil {
		t.Fatalf("Contacts.Get: %v", err)
	}
	if result.ID != "c_1" {
		t.Errorf("expected c_1, got %s", result.ID)
	}
}

func TestContacts_Update(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != "/v1/contacts/u_1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "c_1", "external_id": "u_1", "email": "new@b.com",
		})
	})

	result, err := client.Contacts.Update("u_1", ContactUpdateParams{Email: "new@b.com"})
	if err != nil {
		t.Fatalf("Contacts.Update: %v", err)
	}
	if result.Email != "new@b.com" {
		t.Errorf("expected new@b.com, got %s", result.Email)
	}
}

func TestContacts_Delete(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/v1/contacts/u_1" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(204)
	})

	err := client.Contacts.Delete("u_1")
	if err != nil {
		t.Fatalf("Contacts.Delete: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Templates sub-client
// ---------------------------------------------------------------------------

func TestTemplates_List(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/templates" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"t_1","name":"Welcome","slug":"welcome"}]`))
	})

	result, err := client.Templates.List()
	if err != nil {
		t.Fatalf("Templates.List: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 template, got %d", len(result))
	}
	if result[0].Slug != "welcome" {
		t.Errorf("expected welcome, got %s", result[0].Slug)
	}
}

func TestTemplates_Get(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/templates/welcome" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "t_1", "name": "Welcome", "slug": "welcome", "subject": "Hello",
		})
	})

	result, err := client.Templates.Get("welcome")
	if err != nil {
		t.Fatalf("Templates.Get: %v", err)
	}
	if result.Subject != "Hello" {
		t.Errorf("expected Hello, got %s", result.Subject)
	}
}

func TestTemplates_Create(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/templates" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "t_2", "name": "New", "slug": "new-tmpl", "subject": "Hi",
		})
	})

	result, err := client.Templates.Create(TemplateCreateParams{
		Name: "New", Slug: "new-tmpl", Subject: "Hi",
	})
	if err != nil {
		t.Fatalf("Templates.Create: %v", err)
	}
	if result.Slug != "new-tmpl" {
		t.Errorf("expected new-tmpl, got %s", result.Slug)
	}
}

func TestTemplates_Update(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/v1/templates/welcome" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "t_1", "name": "Welcome v2", "slug": "welcome", "subject": "Updated",
		})
	})

	result, err := client.Templates.Update("welcome", TemplateUpdateParams{Subject: "Updated"})
	if err != nil {
		t.Fatalf("Templates.Update: %v", err)
	}
	if result.Subject != "Updated" {
		t.Errorf("expected Updated, got %s", result.Subject)
	}
}

func TestTemplates_Delete(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/v1/templates/welcome" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(204)
	})

	err := client.Templates.Delete("welcome")
	if err != nil {
		t.Fatalf("Templates.Delete: %v", err)
	}
}

func TestTemplates_Preview(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/templates/welcome/preview" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"subject": "Hello Jane", "html": "<p>Hi</p>",
			"suppressed": false, "suppressed_reason": nil,
		})
	})

	result, err := client.Templates.Preview("welcome", TemplatePreviewParams{})
	if err != nil {
		t.Fatalf("Templates.Preview: %v", err)
	}
	if result.Subject != "Hello Jane" {
		t.Errorf("expected Hello Jane, got %s", result.Subject)
	}
	if result.Suppressed != false {
		t.Error("expected suppressed to be false")
	}
}

// ---------------------------------------------------------------------------
// Error mapping (via client)
// ---------------------------------------------------------------------------

func TestClient_AuthError401(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "unauthorized"})
	})

	_, err := client.Track(TrackParams{ExternalID: "u1", EventName: "test"})
	if err == nil {
		t.Fatal("expected error")
	}

	authErr, ok := err.(*SynapseAuthError)
	if !ok {
		t.Fatalf("expected SynapseAuthError, got %T", err)
	}
	if authErr.Status != 401 {
		t.Errorf("expected status 401, got %d", authErr.Status)
	}
	if authErr.Message != "unauthorized" {
		t.Errorf("expected unauthorized, got %s", authErr.Message)
	}
}

func TestClient_RateLimitError429(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(429)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "rate limited"})
	})

	_, err := client.Track(TrackParams{ExternalID: "u1", EventName: "test"})
	if err == nil {
		t.Fatal("expected error")
	}

	_, ok := err.(*SynapseRateLimitError)
	if !ok {
		t.Fatalf("expected SynapseRateLimitError, got %T", err)
	}
}

func TestClient_ValidationError422(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "validation failed",
			"errors":  []interface{}{map[string]interface{}{"field": "email", "message": "invalid"}},
		})
	})

	_, err := client.Track(TrackParams{ExternalID: "u1", EventName: "test"})
	if err == nil {
		t.Fatal("expected error")
	}

	valErr, ok := err.(*SynapseValidationError)
	if !ok {
		t.Fatalf("expected SynapseValidationError, got %T", err)
	}
	if len(valErr.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(valErr.Errors))
	}
}

// ---------------------------------------------------------------------------
// Retry logic
// ---------------------------------------------------------------------------

func TestRetry_500ThenSuccess(t *testing.T) {
	var callCount int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		if count == 1 {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]interface{}{"message": "internal error"})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"event_id": "e1", "status": "ok"})
	})

	result, err := client.Track(TrackParams{ExternalID: "u1", EventName: "test"})
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("expected 2 calls, got %d", atomic.LoadInt32(&callCount))
	}
	if result.EventID != "e1" {
		t.Errorf("expected e1, got %s", result.EventID)
	}
}

func TestRetry_NoRetryOn401(t *testing.T) {
	var callCount int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "unauthorized"})
	})

	_, err := client.Track(TrackParams{ExternalID: "u1", EventName: "test"})
	if err == nil {
		t.Fatal("expected error")
	}

	if _, ok := err.(*SynapseAuthError); !ok {
		t.Fatalf("expected SynapseAuthError, got %T", err)
	}

	if atomic.LoadInt32(&callCount) != 1 {
		t.Errorf("expected 1 call (no retry), got %d", atomic.LoadInt32(&callCount))
	}
}

func TestRetry_502GivesUpAfterMaxRetries(t *testing.T) {
	var callCount int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(502)
		json.NewEncoder(w).Encode(map[string]interface{}{"message": "bad gateway"})
	})

	_, err := client.Track(TrackParams{ExternalID: "u1", EventName: "test"})
	if err == nil {
		t.Fatal("expected error")
	}

	// max_retries=1 means 2 total attempts
	if atomic.LoadInt32(&callCount) != 2 {
		t.Errorf("expected 2 calls, got %d", atomic.LoadInt32(&callCount))
	}
}

// ---------------------------------------------------------------------------
// 204 handling
// ---------------------------------------------------------------------------

func TestResponse204_ReturnsNil(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	err := client.request("DELETE", "/v1/contacts/u_1", nil, nil)
	if err != nil {
		t.Fatalf("expected nil error for 204, got: %v", err)
	}
}
