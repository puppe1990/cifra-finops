package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
)

// Amarra Drive forces a full reload when the layout marker changes; without it
// the next page morphs into the previous shell and the view collapses.
func TestLayoutTemplates_exposeAmarraLayoutMarker(t *testing.T) {
	t.Run("dashboard uses app marker", func(t *testing.T) {
		h := NewDashboardHandler(setupTestRenderer(t), setupTestStore(t), testSite(), cais.Config{}, setupTestViews(t))

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, inertiaRequest(http.MethodGet, "/dashboard", nil))

		assertAmarraLayoutMarker(t, rr.Body.String(), "app")
	})

	t.Run("login uses auth marker", func(t *testing.T) {
		h, _ := newAuthHandler(t)

		rr := httptest.NewRecorder()
		h.Login(rr, httptest.NewRequest(http.MethodGet, "/login", nil))

		assertAmarraLayoutMarker(t, rr.Body.String(), "auth")
	})

	t.Run("home uses public marker", func(t *testing.T) {
		h := newHomeHandler(t)

		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

		assertAmarraLayoutMarker(t, rr.Body.String(), "public")
	})
}

func assertAmarraLayoutMarker(t *testing.T, body, want string) {
	t.Helper()
	if wantAttr := `data-amarra-layout="` + want + `"`; !strings.Contains(body, wantAttr) {
		t.Errorf(`missing %s on <html>`, wantAttr)
	}
	if strings.Contains(body, "data-layout=") {
		t.Error("stale data-layout attribute; Amarra Drive reads data-amarra-layout")
	}
}
