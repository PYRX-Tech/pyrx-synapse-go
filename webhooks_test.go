package synapse

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"
)

func generateSecret() (raw []byte, b64 string, withPrefix string) {
	raw = make([]byte, 32)
	rand.Read(raw)
	b64 = base64.StdEncoding.EncodeToString(raw)
	withPrefix = "whsec_" + b64
	return
}

func computeTestSignature(secretBytes []byte, msgID, timestamp, payload string) string {
	signedContent := fmt.Sprintf("%s.%s.%s", msgID, timestamp, payload)
	mac := hmac.New(sha256.New, secretBytes)
	mac.Write([]byte(signedContent))
	sig := mac.Sum(nil)
	return "v1," + base64.StdEncoding.EncodeToString(sig)
}

func TestVerifyWebhook_ValidWithPrefix(t *testing.T) {
	secretRaw, _, secretWithPrefix := generateSecret()
	payload := `{"type":"contact.created","data":{"id":"c_123"}}`
	svixID := "msg_abc123"
	svixTimestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := computeTestSignature(secretRaw, svixID, svixTimestamp, payload)

	headers := map[string]string{
		"svix-id":        svixID,
		"svix-timestamp": svixTimestamp,
		"svix-signature": sig,
	}

	result, err := VerifyWebhook(payload, headers, secretWithPrefix, false)
	if err != nil {
		t.Fatalf("VerifyWebhook: %v", err)
	}
	if result["type"] != "contact.created" {
		t.Errorf("expected contact.created, got %v", result["type"])
	}
}

func TestVerifyWebhook_ValidWithoutPrefix(t *testing.T) {
	secretRaw, secretB64, _ := generateSecret()
	payload := `{"type":"contact.created","data":{"id":"c_123"}}`
	svixID := "msg_abc123"
	svixTimestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := computeTestSignature(secretRaw, svixID, svixTimestamp, payload)

	headers := map[string]string{
		"svix-id":        svixID,
		"svix-timestamp": svixTimestamp,
		"svix-signature": sig,
	}

	result, err := VerifyWebhook(payload, headers, secretB64, false)
	if err != nil {
		t.Fatalf("VerifyWebhook: %v", err)
	}
	if result["type"] != "contact.created" {
		t.Errorf("expected contact.created, got %v", result["type"])
	}
}

// ---------------------------------------------------------------------------
// Missing headers
// ---------------------------------------------------------------------------

func TestVerifyWebhook_MissingSvixID(t *testing.T) {
	_, _, secretWithPrefix := generateSecret()
	headers := map[string]string{
		"svix-timestamp": "123",
		"svix-signature": "v1,abc",
	}
	_, err := VerifyWebhook("{}", headers, secretWithPrefix, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Missing required webhook headers") {
		t.Errorf("unexpected error: %s", err.Error())
	}
}

func TestVerifyWebhook_MissingSvixTimestamp(t *testing.T) {
	_, _, secretWithPrefix := generateSecret()
	headers := map[string]string{
		"svix-id":        "msg_1",
		"svix-signature": "v1,abc",
	}
	_, err := VerifyWebhook("{}", headers, secretWithPrefix, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Missing required webhook headers") {
		t.Errorf("unexpected error: %s", err.Error())
	}
}

func TestVerifyWebhook_MissingSvixSignature(t *testing.T) {
	_, _, secretWithPrefix := generateSecret()
	headers := map[string]string{
		"svix-id":        "msg_1",
		"svix-timestamp": "123",
	}
	_, err := VerifyWebhook("{}", headers, secretWithPrefix, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Missing required webhook headers") {
		t.Errorf("unexpected error: %s", err.Error())
	}
}

func TestVerifyWebhook_EmptyHeaders(t *testing.T) {
	_, _, secretWithPrefix := generateSecret()
	headers := map[string]string{
		"svix-id":        "",
		"svix-timestamp": "123",
		"svix-signature": "v1,abc",
	}
	_, err := VerifyWebhook("{}", headers, secretWithPrefix, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Missing required webhook headers") {
		t.Errorf("unexpected error: %s", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Expired timestamp
// ---------------------------------------------------------------------------

func TestVerifyWebhook_ExpiredTimestamp(t *testing.T) {
	secretRaw, _, secretWithPrefix := generateSecret()
	payload := `{"type":"test"}`
	svixID := "msg_1"
	oldTimestamp := fmt.Sprintf("%d", time.Now().Unix()-600)
	sig := computeTestSignature(secretRaw, svixID, oldTimestamp, payload)

	headers := map[string]string{
		"svix-id":        svixID,
		"svix-timestamp": oldTimestamp,
		"svix-signature": sig,
	}

	_, err := VerifyWebhook(payload, headers, secretWithPrefix, false)
	if err == nil {
		t.Fatal("expected error for expired timestamp")
	}
	if !strings.Contains(err.Error(), "Webhook timestamp too old") {
		t.Errorf("unexpected error: %s", err.Error())
	}
}

func TestVerifyWebhook_DisabledTimestampCheck(t *testing.T) {
	secretRaw, _, secretWithPrefix := generateSecret()
	payload := `{"type":"test"}`
	svixID := "msg_1"
	oldTimestamp := fmt.Sprintf("%d", time.Now().Unix()-600)
	sig := computeTestSignature(secretRaw, svixID, oldTimestamp, payload)

	headers := map[string]string{
		"svix-id":        svixID,
		"svix-timestamp": oldTimestamp,
		"svix-signature": sig,
	}

	result, err := VerifyWebhook(payload, headers, secretWithPrefix, true)
	if err != nil {
		t.Fatalf("expected success with disabled timestamp check: %v", err)
	}
	if result["type"] != "test" {
		t.Errorf("expected test, got %v", result["type"])
	}
}

// ---------------------------------------------------------------------------
// Invalid signature
// ---------------------------------------------------------------------------

func TestVerifyWebhook_InvalidSignature(t *testing.T) {
	_, _, secretWithPrefix := generateSecret()
	payload := `{"type":"test"}`
	svixID := "msg_1"
	svixTimestamp := fmt.Sprintf("%d", time.Now().Unix())

	headers := map[string]string{
		"svix-id":        svixID,
		"svix-timestamp": svixTimestamp,
		"svix-signature": "v1,aW52YWxpZHNpZ25hdHVyZWhlcmU=",
	}

	_, err := VerifyWebhook(payload, headers, secretWithPrefix, false)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
	if !strings.Contains(err.Error(), "Webhook signature verification failed") {
		t.Errorf("unexpected error: %s", err.Error())
	}
}

// ---------------------------------------------------------------------------
// Multiple signatures (key rotation)
// ---------------------------------------------------------------------------

func TestVerifyWebhook_MultipleSignatures_OneValid(t *testing.T) {
	secretRaw, _, secretWithPrefix := generateSecret()
	payload := `{"type":"test"}`
	svixID := "msg_1"
	svixTimestamp := fmt.Sprintf("%d", time.Now().Unix())
	validSig := computeTestSignature(secretRaw, svixID, svixTimestamp, payload)
	badSig := "v1,aW52YWxpZHNpZ25hdHVyZWhlcmU="
	multiSig := badSig + " " + validSig

	headers := map[string]string{
		"svix-id":        svixID,
		"svix-timestamp": svixTimestamp,
		"svix-signature": multiSig,
	}

	result, err := VerifyWebhook(payload, headers, secretWithPrefix, false)
	if err != nil {
		t.Fatalf("expected success with valid sig in multi: %v", err)
	}
	if result["type"] != "test" {
		t.Errorf("expected test, got %v", result["type"])
	}
}

func TestVerifyWebhook_MultipleSignatures_NoneValid(t *testing.T) {
	_, _, secretWithPrefix := generateSecret()
	payload := `{"type":"test"}`
	svixID := "msg_1"
	svixTimestamp := fmt.Sprintf("%d", time.Now().Unix())
	multiSig := "v1,aW52YWxpZA== v1,YW5vdGhlcmludmFsaWQ="

	headers := map[string]string{
		"svix-id":        svixID,
		"svix-timestamp": svixTimestamp,
		"svix-signature": multiSig,
	}

	_, err := VerifyWebhook(payload, headers, secretWithPrefix, false)
	if err == nil {
		t.Fatal("expected error when no signatures match")
	}
	if !strings.Contains(err.Error(), "Webhook signature verification failed") {
		t.Errorf("unexpected error: %s", err.Error())
	}
}

// ---------------------------------------------------------------------------
// whsec_ prefix stripping
// ---------------------------------------------------------------------------

func TestVerifyWebhook_SecretWithPrefix(t *testing.T) {
	secretRaw, _, secretWithPrefix := generateSecret()
	payload := `{"type":"test"}`
	svixID := "msg_1"
	svixTimestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := computeTestSignature(secretRaw, svixID, svixTimestamp, payload)

	headers := map[string]string{
		"svix-id":        svixID,
		"svix-timestamp": svixTimestamp,
		"svix-signature": sig,
	}

	result, err := VerifyWebhook(payload, headers, secretWithPrefix, false)
	if err != nil {
		t.Fatalf("VerifyWebhook: %v", err)
	}
	if _, ok := result["type"]; !ok {
		t.Error("expected parsed result to contain type")
	}
}

func TestVerifyWebhook_SecretWithoutPrefix(t *testing.T) {
	secretRaw, secretB64, _ := generateSecret()
	payload := `{"type":"test"}`
	svixID := "msg_1"
	svixTimestamp := fmt.Sprintf("%d", time.Now().Unix())
	sig := computeTestSignature(secretRaw, svixID, svixTimestamp, payload)

	headers := map[string]string{
		"svix-id":        svixID,
		"svix-timestamp": svixTimestamp,
		"svix-signature": sig,
	}

	result, err := VerifyWebhook(payload, headers, secretB64, false)
	if err != nil {
		t.Fatalf("VerifyWebhook: %v", err)
	}
	if _, ok := result["type"]; !ok {
		t.Error("expected parsed result to contain type")
	}
}
