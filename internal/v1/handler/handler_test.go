package handlerv1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ton-connect/bridge/internal/utils"
	"github.com/ton-connect/bridge/internal/v1/storage"
)

func TestSendMessageHandler_Returns200(t *testing.T) {
	// Setup
	e := echo.New()

	memStorage := storage.NewMemStorage(nil, nil)
	extractor, err := utils.NewRealIPExtractor([]string{})
	if err != nil {
		t.Fatalf("failed to create RealIPExtractor: %v", err)
	}

	h := NewHandler(memStorage, 10*time.Second, extractor, nil, nil)

	// Create request with required query parameters
	// The "to" parameter needs to be a valid hex-encoded public key (64 chars)
	toID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	clientID := "test-client-id"
	body := strings.NewReader("test message payload")

	req := httptest.NewRequest(http.MethodPost, "/bridge/message?client_id="+clientID+"&to="+toID+"&ttl=60&no_request_source=true", body)
	req.Header.Set("Content-Type", "application/octet-stream")

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err = h.SendMessageHandler(c)

	// Verify
	if err != nil {
		t.Fatalf("SendMessageHandler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check response body contains OK message
	responseBody := rec.Body.String()
	if !strings.Contains(responseBody, `"message":"OK"`) {
		t.Errorf("expected response to contain OK message, got: %s", responseBody)
	}
	if !strings.Contains(responseBody, `"statusCode":200`) {
		t.Errorf("expected response to contain statusCode 200, got: %s", responseBody)
	}
}

func TestSendMessageHandler_MissingClientID(t *testing.T) {
	// Setup
	e := echo.New()

	memStorage := storage.NewMemStorage(nil, nil)
	extractor, err := utils.NewRealIPExtractor([]string{})
	if err != nil {
		t.Fatalf("failed to create RealIPExtractor: %v", err)
	}

	h := NewHandler(memStorage, 10*time.Second, extractor, nil, nil)

	// Create request without client_id
	toID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	body := strings.NewReader("test message payload")

	req := httptest.NewRequest(http.MethodPost, "/bridge/message?to="+toID+"&ttl=60", body)
	req.Header.Set("Content-Type", "application/octet-stream")

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err = h.SendMessageHandler(c)

	// Verify - should return error for missing client_id
	if err != nil {
		t.Fatalf("SendMessageHandler returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSendMessageHandler_MissingTo(t *testing.T) {
	// Setup
	e := echo.New()

	memStorage := storage.NewMemStorage(nil, nil)
	extractor, err := utils.NewRealIPExtractor([]string{})
	if err != nil {
		t.Fatalf("failed to create RealIPExtractor: %v", err)
	}

	h := NewHandler(memStorage, 10*time.Second, extractor, nil, nil)

	// Create request without "to" parameter
	clientID := "test-client-id"
	body := strings.NewReader("test message payload")

	req := httptest.NewRequest(http.MethodPost, "/bridge/message?client_id="+clientID+"&ttl=60", body)
	req.Header.Set("Content-Type", "application/octet-stream")

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err = h.SendMessageHandler(c)

	// Verify - should return error for missing "to"
	if err != nil {
		t.Fatalf("SendMessageHandler returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSendMessageHandler_MissingTTL(t *testing.T) {
	// Setup
	e := echo.New()

	memStorage := storage.NewMemStorage(nil, nil)
	extractor, err := utils.NewRealIPExtractor([]string{})
	if err != nil {
		t.Fatalf("failed to create RealIPExtractor: %v", err)
	}

	h := NewHandler(memStorage, 10*time.Second, extractor, nil, nil)

	// Create request without ttl
	toID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	clientID := "test-client-id"
	body := strings.NewReader("test message payload")

	req := httptest.NewRequest(http.MethodPost, "/bridge/message?client_id="+clientID+"&to="+toID, body)
	req.Header.Set("Content-Type", "application/octet-stream")

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err = h.SendMessageHandler(c)

	// Verify - should return error for missing ttl
	if err != nil {
		t.Fatalf("SendMessageHandler returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSendMessageHandler_TTLTooHigh(t *testing.T) {
	// Setup
	e := echo.New()

	memStorage := storage.NewMemStorage(nil, nil)
	extractor, err := utils.NewRealIPExtractor([]string{})
	if err != nil {
		t.Fatalf("failed to create RealIPExtractor: %v", err)
	}

	h := NewHandler(memStorage, 10*time.Second, extractor, nil, nil)

	// Create request with TTL > 300
	toID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	clientID := "test-client-id"
	body := strings.NewReader("test message payload")

	req := httptest.NewRequest(http.MethodPost, "/bridge/message?client_id="+clientID+"&to="+toID+"&ttl=500", body)
	req.Header.Set("Content-Type", "application/octet-stream")

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err = h.SendMessageHandler(c)

	// Verify - should return error for TTL too high
	if err != nil {
		t.Fatalf("SendMessageHandler returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestSendMessageHandler_LargeToID(t *testing.T) {
	// Setup
	e := echo.New()

	memStorage := storage.NewMemStorage(nil, nil)
	extractor, err := utils.NewRealIPExtractor([]string{})
	if err != nil {
		t.Fatalf("failed to create RealIPExtractor: %v", err)
	}

	h := NewHandler(memStorage, 10*time.Second, extractor, nil, nil)

	// Create toID and clientID with 1024*100 = 102400 characters each
	toID := strings.Repeat("a", 1024*100)
	clientID := strings.Repeat("b", 1024*100)
	body := strings.NewReader("test message payload")

	req := httptest.NewRequest(http.MethodPost, "/bridge/message?client_id="+clientID+"&to="+toID+"&ttl=60&no_request_source=true", body)
	req.Header.Set("Content-Type", "application/octet-stream")

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Execute
	err = h.SendMessageHandler(c)

	// Verify - handler should process the request (returns 200 with no_request_source=true)
	if err != nil {
		t.Fatalf("SendMessageHandler returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Check response body contains OK message
	responseBody := rec.Body.String()
	if !strings.Contains(responseBody, `"message":"OK"`) {
		t.Errorf("expected response to contain OK message, got: %s", responseBody)
	}
}
