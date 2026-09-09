package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"

	"github.com/puppe1990/cifra-finops/internal/locale"
)

func TestLocaleHandler_Post_setsCookieAndRedirects(t *testing.T) {
	h := NewLocaleHandler(cais.Config{}, setupTestViews(t))
	form := url.Values{"locale": {"pt-BR"}}
	req := httptest.NewRequest(http.MethodPost, "/locale", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Host = "cifra.test"
	req.Header.Set("Referer", "https://cifra.test/dashboard")
	rr := httptest.NewRecorder()
	h.Post(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Fatalf("status = %d", rr.Code)
	}
	if rr.Header().Get("Location") != "/dashboard" {
		t.Fatalf("Location = %q", rr.Header().Get("Location"))
	}
	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == locale.CookieName && c.Value == "pt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("locale cookie missing: %#v", rr.Result().Cookies())
	}
}
