package hetznerinv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLive_Collect_estimatesFromAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok-123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/pricing":
			_, _ = w.Write([]byte(`{"pricing":{"volume":{"price_per_gb_month":{"net":"0.044"}},"server_types":[{"name":"cx22","prices":[{"location":"fsn1","price_monthly":{"net":"3.49"}}]}]}}`))
		case "/servers":
			_, _ = w.Write([]byte(`{"servers":[{"id":42,"name":"web-1","status":"running","server_type":{"name":"cx22"},"datacenter":{"location":{"name":"fsn1"}},"public_net":{"ipv4":null}}],"meta":{"pagination":{"next_page":null}}}`))
		case "/volumes", "/primary_ips", "/floating_ips", "/load_balancers", "/images":
			_, _ = w.Write([]byte(`{"volumes":[],"primary_ips":[],"floating_ips":[],"load_balancers":[],"images":[],"meta":{"pagination":{"next_page":null}}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	inv, err := (&Live{BaseURL: srv.URL, HTTP: srv.Client()}).Collect(context.Background(), "tok-123")
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Resources) != 1 || inv.Resources[0].Name != "web-1" || inv.Resources[0].MonthlyCents != 349 {
		t.Fatalf("resources = %#v", inv.Resources)
	}
}

func TestLive_Collect_unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"unable to authenticate"}}`))
	}))
	t.Cleanup(srv.Close)

	_, err := (&Live{BaseURL: srv.URL, HTTP: srv.Client()}).Collect(context.Background(), "bad")
	if err == nil {
		t.Fatal("expected auth error")
	}
}
