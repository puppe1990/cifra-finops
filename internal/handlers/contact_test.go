package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"

	"github.com/puppe1990/aws-finops/internal/store"
)

func newContactHandler(t *testing.T) (*ContactHandler, store.Store) {
	t.Helper()
	s := setupTestStore(t)
	h := NewContactHandler(setupTestRenderer(t), s, testSite(), i18n.DefaultCatalog(), cais.Config{}, setupTestViews(t))
	return h, s
}

func TestContactHandler_Get_InertiaComponent(t *testing.T) {
	h, _ := newContactHandler(t)

	req := inertiaRequest(http.MethodGet, "/contact", nil)
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	assertInertiaComponent(t, rr, "Contact")
}

func TestContactHandler_Post_MalformedEmail_InertiaErrors(t *testing.T) {
	h, _ := newContactHandler(t)

	req := inertiaRequest(http.MethodPost, "/contact", strings.NewReader("name=Alice&email=not-an-email"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Post(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rr.Code)
	}
	assertInertiaComponent(t, rr, "Contact")
	assertInertiaErrors(t, rr, "email")
}

func TestContactHandler_Post_MissingName_InertiaErrors(t *testing.T) {
	h, _ := newContactHandler(t)

	req := inertiaRequest(http.MethodPost, "/contact", strings.NewReader("name=&email=alice@example.com"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Post(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rr.Code)
	}
	assertInertiaErrors(t, rr, "name")
}

func TestContactHandler_Post_InvalidEmail_InertiaErrors(t *testing.T) {
	h, _ := newContactHandler(t)

	req := inertiaRequest(http.MethodPost, "/contact", strings.NewReader("name=Alice&email="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Post(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rr.Code)
	}
	assertInertiaErrors(t, rr, "email")
}

func TestContactHandler_Get_InertiaFlash(t *testing.T) {
	h, _ := newContactHandler(t)

	req := inertiaRequest(http.MethodGet, "/contact", nil)
	req = flash.WithMessage(req, flash.Message{Kind: "success", Message: "Message sent successfully."})
	rr := httptest.NewRecorder()
	h.Get(rr, req)

	if !strings.Contains(rr.Body.String(), "Message sent successfully.") {
		t.Errorf("flash missing from HTML: %s", rr.Body.String())
	}
}

func TestContactHandler_Post_Valid_Redirects(t *testing.T) {
	h, s := newContactHandler(t)

	req := inertiaRequest(http.MethodPost, "/contact", strings.NewReader("name=Alice&email=alice@example.com"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.Post(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want 303", rr.Code)
	}
	if rr.Header().Get("Location") != "/contact" {
		t.Errorf("Location = %q, want /contact", rr.Header().Get("Location"))
	}
	count, err := s.CountContacts()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("contact count = %d, want 1", count)
	}
}
