# PYRX Synapse Go SDK

Official Go SDK for the [PYRX Synapse](https://synapse.pyrx.tech) customer engagement platform.

## Installation

```bash
go get github.com/pyrx-tech/pyrx-synapse-go
```

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	synapse "github.com/pyrx-tech/pyrx-synapse-go"
)

func main() {
	client, err := synapse.NewClient(synapse.Config{
		APIKey:      "psk_live_your_api_key",
		WorkspaceID: "your_workspace_id",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Track an event
	resp, err := client.Track(synapse.TrackParams{
		ExternalID: "user_123",
		EventName:  "purchase_completed",
		Attributes: map[string]interface{}{
			"amount":   99.99,
			"currency": "USD",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Event tracked: %s\n", resp.EventID)
}
```

## Usage

### Configuration

```go
client, err := synapse.NewClient(synapse.Config{
	APIKey:      "psk_live_your_api_key",  // Required
	WorkspaceID: "your_workspace_id",       // Required
	BaseURL:     "https://synapse-api.pyrx.tech", // Optional (default)
	Timeout:     30 * time.Second,          // Optional (default: 30s)
	MaxRetries:  3,                         // Optional (default: 3)
})
```

### Track Events

```go
// Single event
resp, err := client.Track(synapse.TrackParams{
	ExternalID: "user_123",
	EventName:  "page_viewed",
	Attributes: map[string]interface{}{
		"page": "/pricing",
	},
})

// Batch events
batchResp, err := client.TrackBatch(synapse.TrackBatchParams{
	Events: []synapse.TrackParams{
		{ExternalID: "user_1", EventName: "click"},
		{ExternalID: "user_2", EventName: "view"},
	},
})
```

### Identify Contacts

```go
// Single contact
contact, err := client.Identify(synapse.IdentifyParams{
	ExternalID: "user_123",
	Email:      "jane@example.com",
	FirstName:  "Jane",
	LastName:   "Doe",
	Properties: map[string]interface{}{
		"plan": "pro",
	},
})

// Batch contacts
bulkResp, err := client.IdentifyBatch(synapse.IdentifyBatchParams{
	Contacts: []synapse.IdentifyParams{
		{ExternalID: "user_1", Email: "a@example.com"},
		{ExternalID: "user_2", Email: "b@example.com"},
	},
	OnConflict: "update",
})
```

### Send Emails

```go
sendResp, err := client.Send(synapse.SendParams{
	TemplateSlug: "welcome",
	To:           map[string]interface{}{"email": "jane@example.com"},
	Attributes:   map[string]interface{}{"name": "Jane"},
})
```

### Manage Contacts

```go
// List contacts
list, err := client.Contacts.List(synapse.ContactListParams{
	Page:    1,
	PerPage: 25,
})

// Get a contact
contact, err := client.Contacts.Get("contact_id")

// Update a contact
updated, err := client.Contacts.Update("external_id", synapse.ContactUpdateParams{
	Email: "new@example.com",
	AddTags: []string{"vip"},
})

// Delete a contact
err = client.Contacts.Delete("external_id")
```

### Manage Templates

```go
// List templates
templates, err := client.Templates.List()

// Get a template
tmpl, err := client.Templates.Get("welcome")

// Create a template
created, err := client.Templates.Create(synapse.TemplateCreateParams{
	Name:       "Welcome Email",
	Slug:       "welcome",
	Subject:    "Welcome, {{contact.first_name}}!",
	BodyHTML:   "<h1>Welcome!</h1>",
	SenderName: "PYRX",
	FromEmail:  "hello@pyrx.tech",
})

// Update a template
updated, err := client.Templates.Update("welcome", synapse.TemplateUpdateParams{
	Subject: "Updated Subject",
})

// Delete a template
err = client.Templates.Delete("welcome")

// Preview a template
preview, err := client.Templates.Preview("welcome", synapse.TemplatePreviewParams{
	Contact: map[string]interface{}{"first_name": "Jane"},
})
```

### Verify Webhooks

```go
result, err := synapse.VerifyWebhook(
	rawBodyString,
	map[string]string{
		"svix-id":        r.Header.Get("svix-id"),
		"svix-timestamp": r.Header.Get("svix-timestamp"),
		"svix-signature": r.Header.Get("svix-signature"),
	},
	"whsec_your_webhook_secret",
	false, // disableTimestampCheck
)
if err != nil {
	// Signature verification failed
	log.Printf("Webhook verification failed: %v", err)
	return
}
// result is a map[string]interface{} with the parsed event
```

### Error Handling

```go
resp, err := client.Track(synapse.TrackParams{
	ExternalID: "user_123",
	EventName:  "test",
})
if err != nil {
	switch e := err.(type) {
	case *synapse.SynapseAuthError:
		log.Printf("Authentication failed: %s", e.Message)
	case *synapse.SynapseRateLimitError:
		log.Printf("Rate limited, retry after: %.0fs", e.RetryAfter)
	case *synapse.SynapsePlanLimitError:
		log.Printf("Plan limit reached: %s (%d/%d on %s plan)",
			e.LimitType, e.Current, e.Maximum, e.Plan)
	case *synapse.SynapseValidationError:
		for _, fieldErr := range e.Errors {
			log.Printf("Validation error on %s: %s", fieldErr.Field, fieldErr.Message)
		}
	case *synapse.SynapseError:
		log.Printf("API error %d: %s", e.Status, e.Message)
	default:
		log.Printf("Unexpected error: %v", err)
	}
}
```

## Retry Logic

The SDK automatically retries on:
- HTTP 429 (Too Many Requests) - uses `Retry-After` header when available
- HTTP 500, 502, 503, 504 (Server Errors)
- Network errors (timeouts, connection refused)

Backoff: `min(1.0 * 2^attempt, 30.0) + random(0, 0.5)` seconds.

Non-retryable errors (401, 403, 422) are returned immediately.

## Environment Detection

The SDK auto-detects your environment from the API key prefix:
- `psk_test_*` -> `"test"`
- `psk_live_*` -> `"live"`
- Other -> `"unknown"`

Access via `client.Environment`.

## Requirements

- Go 1.21+
- No external dependencies (stdlib only)

## License

MIT
