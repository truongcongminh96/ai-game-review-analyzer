package http

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		payload    map[string]string
	}{
		{
			name:       "writes json response with status code",
			statusCode: 200,
			payload: map[string]string{
				"status": "ok",
			},
		},
		{
			name:       "writes custom json payload",
			statusCode: 201,
			payload: map[string]string{
				"message": "created",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			writeJSON(rec, tt.statusCode, tt.payload)

			if rec.Code != tt.statusCode {
				t.Fatalf("expected status %d, got %d", tt.statusCode, rec.Code)
			}

			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected content type application/json, got %q", got)
			}

			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}

			for key, wantValue := range tt.payload {
				if body[key] != wantValue {
					t.Fatalf("expected body[%q] = %q, got %q", key, wantValue, body[key])
				}
			}
		})
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
	}{
		{
			name:       "writes bad request error response",
			statusCode: 400,
			message:    "invalid request body",
		},
		{
			name:       "writes method not allowed error response",
			statusCode: 405,
			message:    "method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			writeError(rec, tt.statusCode, tt.message)

			if rec.Code != tt.statusCode {
				t.Fatalf("expected status %d, got %d", tt.statusCode, rec.Code)
			}

			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected content type application/json, got %q", got)
			}

			var body errorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to decode error response body: %v", err)
			}

			if body.Error != tt.message {
				t.Fatalf("expected error %q, got %q", tt.message, body.Error)
			}
		})
	}
}
