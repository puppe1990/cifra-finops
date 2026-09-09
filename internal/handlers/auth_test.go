package handlers

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/i18n"
	"github.com/puppe1990/amarra-cais/pkg/cais/middleware"
	"github.com/puppe1990/amarra-cais/pkg/cais/session"

	"github.com/puppe1990/aws-finops/internal/store"
)

func newAuthHandler(t *testing.T) (*AuthHandler, store.Store) {
	t.Helper()
	s := setupTestStore(t)
	h := NewAuthHandler(setupTestRenderer(t), s, testSite(), s.Sessions(), cais.Config{}, i18n.DefaultCatalog(), setupTestViews(t))
	return h, s
}

func TestAuth_Login_redirectsWhenAuthenticated(t *testing.T) {
	h, s := newAuthHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req = session.WithUserID(req, 1)
	rr := httptest.NewRecorder()
	h.Login(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want 303", rr.Code)
	}
	_ = s
}

func TestAuth_LoginPost_multipartFormAmarraDrive_redirects(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "development")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	h := NewAuthHandler(setupTestRenderer(t), s, testSite(), s.Sessions(), cais.Config{}, i18n.DefaultCatalog(), setupTestViews(t))

	// Run through the real middleware stack: CSRF parses the body first and
	// net/http only fills PostForm for multipart, so handlers must mirror it
	// into r.Form for FormValue to see Amarra Drive multipart submissions.
	mux := http.NewServeMux()
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.Login(w, r)
			return
		}
		h.LoginPost(w, r)
	})
	var stack http.Handler = mux
	stack = middleware.CSRF(cais.Config{})(stack)
	srv := httptest.NewServer(stack)
	t.Cleanup(srv.Close)

	jar, _ := cookiejar.New(nil)
	client := srv.Client()
	client.Jar = jar
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Get(srv.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	var token string
	for _, c := range resp.Cookies() {
		if c.Name == "cais_csrf" {
			token = c.Value
		}
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("email", "demo@example.com")
	_ = w.WriteField("password", "password")
	_ = w.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/login", &body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-CSRF-Token", token)
	resp2, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusSeeOther {
		t.Errorf("status = %d, want 303", resp2.StatusCode)
	}
	if resp2.Header.Get("Location") != "/dashboard" {
		t.Errorf("Location = %q, want /dashboard", resp2.Header.Get("Location"))
	}
}

func TestAuth_LoginPost_invalidCredentials(t *testing.T) {
	h, _ := newAuthHandler(t)

	form := url.Values{"email": {"nobody@example.com"}, "password": {"wrong"}}
	req := inertiaRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.LoginPost(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rr.Code)
	}
	assertInertiaComponent(t, rr, "Login")
	assertInertiaErrors(t, rr, "email")
}

func TestAuth_LoginPost_validCredentials_redirects(t *testing.T) {
	s, err := store.NewSQLiteStore(":memory:", "development")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	h := NewAuthHandler(setupTestRenderer(t), s, testSite(), s.Sessions(), cais.Config{}, i18n.DefaultCatalog(), setupTestViews(t))

	form := url.Values{"email": {"demo@example.com"}, "password": {"password"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.LoginPost(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("status = %d, want 303, body: %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Location") != "/dashboard" {
		t.Errorf("Location = %q, want /dashboard", rr.Header().Get("Location"))
	}
	foundFlash := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == "cais_flash" && c.Value != "" {
			foundFlash = true
			break
		}
	}
	if !foundFlash {
		t.Error("login must set cais_flash cookie via flash.Set (inertia.SetFlash is a no-op without FlashDataProvider)")
	}
}
