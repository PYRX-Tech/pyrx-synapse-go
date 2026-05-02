package synapse

import "time"

// Version is the SDK version string.
const Version = "0.1.0"

// Config holds the configuration for a Synapse client.
type Config struct {
	APIKey      string
	WorkspaceID string
	BaseURL     string        // default: "https://synapse-api.pyrx.tech"
	Timeout     time.Duration // default: 30s
	MaxRetries  int           // default: 3
}

// ---------------------------------------------------------------------------
// Track
// ---------------------------------------------------------------------------

// TrackParams holds the parameters for tracking an event.
type TrackParams struct {
	ExternalID     string                 `json:"external_id"`
	EventName      string                 `json:"event_name"`
	Attributes     map[string]interface{} `json:"attributes,omitempty"`
	Contact        map[string]interface{} `json:"contact,omitempty"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
	OccurredAt     string                 `json:"occurred_at,omitempty"`
}

// TrackResponse is the response from tracking a single event.
type TrackResponse struct {
	EventID string `json:"event_id"`
	Status  string `json:"status"`
}

// TrackBatchParams holds the parameters for tracking multiple events.
type TrackBatchParams struct {
	Events []TrackParams `json:"events"`
}

// BatchTrackResponse is the response from a batch track call.
type BatchTrackResponse struct {
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
}

// ---------------------------------------------------------------------------
// Identify
// ---------------------------------------------------------------------------

// IdentifyParams holds the parameters for identifying a contact.
type IdentifyParams struct {
	ExternalID string                 `json:"external_id"`
	Email      string                 `json:"email,omitempty"`
	Phone      string                 `json:"phone,omitempty"`
	FirstName  string                 `json:"first_name,omitempty"`
	LastName   string                 `json:"last_name,omitempty"`
	Timezone   string                 `json:"timezone,omitempty"`
	Locale     string                 `json:"locale,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
}

// ContactResponse is the response containing contact data.
type ContactResponse struct {
	ID         string                 `json:"id"`
	ExternalID string                 `json:"external_id"`
	Email      string                 `json:"email"`
	Phone      string                 `json:"phone"`
	FirstName  string                 `json:"first_name"`
	LastName   string                 `json:"last_name"`
	Timezone   string                 `json:"timezone"`
	Locale     string                 `json:"locale"`
	Properties map[string]interface{} `json:"properties"`
	Tags       []string               `json:"tags"`
	CreatedAt  string                 `json:"created_at"`
	UpdatedAt  string                 `json:"updated_at"`
}

// IdentifyBatchParams holds the parameters for batch identifying contacts.
type IdentifyBatchParams struct {
	Contacts   []IdentifyParams `json:"contacts"`
	OnConflict string           `json:"on_conflict,omitempty"`
}

// BulkContactResponse is the response from a bulk contact operation.
type BulkContactResponse struct {
	Total   int                      `json:"total"`
	Created int                      `json:"created"`
	Updated int                      `json:"updated"`
	Skipped int                      `json:"skipped"`
	Errors  []map[string]interface{} `json:"errors"`
}

// ---------------------------------------------------------------------------
// Send
// ---------------------------------------------------------------------------

// SendParams holds the parameters for sending an email.
type SendParams struct {
	TemplateSlug   string                 `json:"template_slug"`
	To             map[string]interface{} `json:"to"`
	Attributes     map[string]interface{} `json:"attributes,omitempty"`
	IdempotencyKey string                 `json:"idempotency_key,omitempty"`
}

// SendResponse is the response from a send email call.
type SendResponse struct {
	EmailLogID string `json:"email_log_id"`
	Status     string `json:"status"`
}

// ---------------------------------------------------------------------------
// Contacts list
// ---------------------------------------------------------------------------

// ContactListParams holds the query parameters for listing contacts.
type ContactListParams struct {
	Search    string
	Page      int
	PerPage   int
	SortBy    string
	SortOrder string
}

// ContactListMeta holds pagination metadata.
type ContactListMeta struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

// ContactListResponse is the response from listing contacts.
type ContactListResponse struct {
	Data []ContactResponse `json:"data"`
	Meta ContactListMeta   `json:"meta"`
}

// ---------------------------------------------------------------------------
// Contact update
// ---------------------------------------------------------------------------

// ContactUpdateParams holds the parameters for updating a contact.
type ContactUpdateParams struct {
	Email      string                 `json:"email,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	AddTags    []string               `json:"add_tags,omitempty"`
	RemoveTags []string               `json:"remove_tags,omitempty"`
}

// ---------------------------------------------------------------------------
// Templates
// ---------------------------------------------------------------------------

// TemplateCreateParams holds the parameters for creating a template.
type TemplateCreateParams struct {
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	Subject    string `json:"subject"`
	BodyHTML   string `json:"body_html"`
	SenderName string `json:"sender_name"`
	FromEmail  string `json:"from_email"`
	ReplyTo    string `json:"reply_to,omitempty"`
}

// TemplateUpdateParams holds the parameters for updating a template.
type TemplateUpdateParams struct {
	Name       string `json:"name,omitempty"`
	Subject    string `json:"subject,omitempty"`
	BodyHTML   string `json:"body_html,omitempty"`
	SenderName string `json:"sender_name,omitempty"`
	FromEmail  string `json:"from_email,omitempty"`
}

// TemplateResponse is the response containing template data.
type TemplateResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Subject     string `json:"subject"`
	BodyHTML    string `json:"body_html"`
	SenderName  string `json:"sender_name"`
	FromEmail   string `json:"from_email"`
	ReplyTo     string `json:"reply_to"`
	ContentType string `json:"content_type"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// TemplatePreviewParams holds the parameters for previewing a template.
type TemplatePreviewParams struct {
	Contact          map[string]interface{}   `json:"contact,omitempty"`
	TriggerEvent     map[string]interface{}   `json:"trigger_event,omitempty"`
	AdditionalEvents []map[string]interface{} `json:"additional_events,omitempty"`
}

// TemplatePreviewResponse is the response from previewing a template.
type TemplatePreviewResponse struct {
	Subject          string `json:"subject"`
	HTML             string `json:"html"`
	Suppressed       bool   `json:"suppressed"`
	SuppressedReason string `json:"suppressed_reason"`
}
