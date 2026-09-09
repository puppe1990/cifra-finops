package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"
)

func newHomeHandler(t *testing.T) *HomeHandler {
	t.Helper()
	return NewHomeHandler(setupTestRenderer(t), testSite(), i18n.DefaultCatalog(), cais.Config{}, setupTestViews(t))
}

func TestHomeHandler_usesMultiCloudLedgerHeading(t *testing.T) {
	h := newHomeHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	body := rr.Body.String()
	if !strings.Contains(body, "The multi cloud finops ledger.") {
		t.Errorf("missing multi cloud heading: %s", body)
	}
	if strings.Contains(body, "The AWS ledger") {
		t.Error("old AWS ledger copy still on home")
	}
}

func TestHomeHandler_Returns200(t *testing.T) {
	h := newHomeHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHomeHandler_InertiaComponent(t *testing.T) {
	h := newHomeHandler(t)

	req := inertiaRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assertInertiaComponent(t, rr, "Home")
}

func TestHomeHandler_InertiaShell(t *testing.T) {
	h := newHomeHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	body := rr.Body.String()
	if !strings.Contains(body, `id="amarra-main"`) {
		t.Errorf("body missing #amarra-main, got: %s", body)
	}
}

func TestHomeHandler_ContentType(t *testing.T) {
	h := newHomeHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
}

func TestHomeHandler_OpenGraphPreview(t *testing.T) {
	h := newHomeHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	body := rr.Body.String()
	wants := []string{
		`property="og:image" content="https://cais.example.com/static/og.png"`,
		`property="og:image:width" content="1200"`,
		`property="og:image:height" content="630"`,
		`name="twitter:card" content="summary_large_image"`,
		`property="og:site_name" content="Cifra"`,
		`property="og:title" content="Cifra · multi cloud FinOps"`,
		`name="twitter:image" content="https://cais.example.com/static/og.png"`,
	}
	for _, want := range wants {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
}
