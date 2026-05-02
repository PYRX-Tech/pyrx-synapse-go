package synapse

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const toleranceSeconds = 300 // 5 minutes

// VerifyWebhook verifies a webhook signature from Synapse (Svix format) and
// returns the parsed payload. Set disableTimestampCheck to true to skip
// timestamp tolerance validation (for testing only).
func VerifyWebhook(payload string, headers map[string]string, secret string, disableTimestampCheck bool) (map[string]interface{}, error) {
	svixID := headers["svix-id"]
	svixTimestamp := headers["svix-timestamp"]
	svixSignature := headers["svix-signature"]

	if svixID == "" || svixTimestamp == "" || svixSignature == "" {
		return nil, errors.New("Missing required webhook headers: svix-id, svix-timestamp, svix-signature")
	}

	// Validate timestamp
	if !disableTimestampCheck {
		ts, err := parseInt64(svixTimestamp)
		if err != nil {
			return nil, errors.New("Invalid svix-timestamp header")
		}
		nowSec := time.Now().Unix()
		diff := math.Abs(float64(nowSec - ts))
		if diff > toleranceSeconds {
			return nil, fmt.Errorf("Webhook timestamp too old (%ds > %ds tolerance)", int(diff), toleranceSeconds)
		}
	}

	// Decode secret: strip "whsec_" prefix and base64-decode
	secretRaw := secret
	if strings.HasPrefix(secret, "whsec_") {
		secretRaw = secret[6:]
	}
	secretBytes, err := base64.StdEncoding.DecodeString(secretRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode webhook secret: %w", err)
	}

	// Compute expected signature
	signedContent := fmt.Sprintf("%s.%s.%s", svixID, svixTimestamp, payload)
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signedContent))
	expectedBytes := mac.Sum(nil)

	// Compare against all provided signatures (svix sends multiple for key rotation)
	signatures := strings.Split(svixSignature, " ")
	verified := false

	for _, sig := range signatures {
		parts := strings.SplitN(sig, ",", 2)
		if len(parts) < 2 {
			continue
		}
		sigBytes, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			continue
		}
		if hmac.Equal(sigBytes, expectedBytes) {
			verified = true
			break
		}
	}

	if !verified {
		return nil, errors.New("Webhook signature verification failed")
	}

	// Parse and return the event
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		return nil, errors.New("Failed to parse webhook payload as JSON")
	}

	return result, nil
}

func parseInt64(s string) (int64, error) {
	var n int64
	for i, c := range s {
		if c == '-' && i == 0 {
			continue
		}
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
	}
	// Use fmt.Sscanf for simplicity
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
