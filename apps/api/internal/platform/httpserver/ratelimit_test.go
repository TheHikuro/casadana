package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientIP_PrefersRightmostForwardedForEntry(t *testing.T) {
	// The caller injected the first entry hoping to be counted as 9.9.9.9; the
	// edge appended the address it actually accepted.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Forwarded-For", "9.9.9.9, 203.0.113.7")
	req.RemoteAddr = "127.0.0.1:5555"

	if got := clientIP(req); got != "203.0.113.7" {
		t.Errorf("clientIP = %q, want the appended edge entry 203.0.113.7", got)
	}
}

func TestClientIP_IgnoresForwardedForWhenRightmostEntryIsNotAnIP(t *testing.T) {
	// A malformed tail must not send us walking left into caller-written
	// entries; X-Real-IP is the next trustworthy source.
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Forwarded-For", "9.9.9.9, unknown")
	req.Header.Set("X-Real-IP", "203.0.113.7")

	if got := clientIP(req); got != "203.0.113.7" {
		t.Errorf("clientIP = %q, want 203.0.113.7", got)
	}
}

func TestClientIP_FallsBackToRealIPThenPeer(t *testing.T) {
	withRealIP := httptest.NewRequest(http.MethodPost, "/", nil)
	withRealIP.Header.Set("X-Real-IP", "198.51.100.4")
	withRealIP.RemoteAddr = "127.0.0.1:5555"
	if got := clientIP(withRealIP); got != "198.51.100.4" {
		t.Errorf("with X-Real-IP: clientIP = %q, want 198.51.100.4", got)
	}

	bare := httptest.NewRequest(http.MethodPost, "/", nil)
	bare.RemoteAddr = "198.51.100.9:41234"
	if got := clientIP(bare); got != "198.51.100.9" {
		t.Errorf("direct call: clientIP = %q, want the peer 198.51.100.9", got)
	}
}

func TestClientIPKey_BucketsIPv6ByPrefix(t *testing.T) {
	// A subscriber handed a whole /64 must not get a fresh budget by moving to
	// another address inside it.
	first := httptest.NewRequest(http.MethodPost, "/", nil)
	first.Header.Set("X-Real-IP", "2001:db8:1:2::1")
	second := httptest.NewRequest(http.MethodPost, "/", nil)
	second.Header.Set("X-Real-IP", "2001:db8:1:2:ffff::9")

	firstKey, _ := clientIPKey(first)
	secondKey, _ := clientIPKey(second)
	if firstKey != secondKey {
		t.Errorf("keys differ within one /64: %q vs %q", firstKey, secondKey)
	}
}

func TestRateLimit_RejectsOverBudgetWithJSONError(t *testing.T) {
	handler := RateLimit(2, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	post := func(clientAddr string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/villas/casadana/reviews", nil)
		req.Header.Set("X-Forwarded-For", clientAddr)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	for i := 1; i <= 2; i++ {
		if rec := post("203.0.113.7"); rec.Code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, want 201", i, rec.Code)
		}
	}

	rec := post("203.0.113.7")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("over budget: status = %d, want 429", rec.Code)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode 429 body: %v", err)
	}
	if body.Error.Code != "RATE_LIMITED" {
		t.Errorf("error code = %q, want RATE_LIMITED", body.Error.Code)
	}
	if body.Error.Message == "" {
		t.Error("429 body carries no message")
	}
}

func TestRateLimit_CountsPerCallerNotGlobally(t *testing.T) {
	handler := RateLimit(1, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	post := func(clientAddr string) int {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set("X-Forwarded-For", clientAddr)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := post("203.0.113.7"); code != http.StatusCreated {
		t.Fatalf("first caller: status = %d, want 201", code)
	}
	if code := post("203.0.113.7"); code != http.StatusTooManyRequests {
		t.Fatalf("first caller over budget: status = %d, want 429", code)
	}
	// A second caller must be unaffected by the first one's spending.
	if code := post("198.51.100.4"); code != http.StatusCreated {
		t.Errorf("second caller: status = %d, want 201", code)
	}
}
