package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApp_jobsDashboardServesOnLoopback(t *testing.T) {
	h := testApp(t).Handler()

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("GET /jobs status=%d want 200 body=%s", rr.Code, rr.Body.String())
	}
}

func TestApp_jobsDashboardRejectsForwardedRequests(t *testing.T) {
	h := testApp(t).Handler()

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	req.Header.Set("X-Forwarded-For", "203.0.113.7")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("GET /jobs via proxy status=%d want 403", rr.Code)
	}
}
