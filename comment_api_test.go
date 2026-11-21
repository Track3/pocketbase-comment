package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifyFCaptchaRequest(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       map[string]any
		wantReject bool
		wantErr    bool
	}{
		{"valid", http.StatusOK, map[string]any{"valid": true, "score": 0.1}, false, false},
		{"invalid token", http.StatusOK, map[string]any{"valid": false, "reason": "invalid"}, true, false},
		{"bad secret", http.StatusUnauthorized, map[string]any{"valid": false, "reason": "invalid_secret"}, false, true},
		{"server error", http.StatusInternalServerError, map[string]any{}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got fcaptchaVerifyRequest
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("unexpected content type %q", r.Header.Get("Content-Type"))
				}
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Errorf("decode body: %v", err)
				}
				w.WriteHeader(tt.status)
				json.NewEncoder(w).Encode(tt.body)
			}))
			defer srv.Close()

			err := verifyFCaptchaRequest(context.Background(), srv.Client(), srv.URL, "secret", "token")

			if got.Token != "token" || got.Secret != "secret" {
				t.Errorf("unexpected payload: %+v", got)
			}
			switch {
			case tt.wantReject:
				if !errors.Is(err, errFCaptchaRejected) {
					t.Fatalf("expected rejection, got %v", err)
				}
			case tt.wantErr:
				if err == nil {
					t.Fatal("expected an error")
				}
			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
