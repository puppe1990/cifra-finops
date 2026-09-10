package app

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	appi18n "github.com/puppe1990/cifra-finops/internal/i18n"
	"github.com/puppe1990/cifra-finops/internal/store"
	"github.com/puppe1990/cifra-finops/web"
)

func testApp(t *testing.T) *App {
	t.Helper()
	tmplFS, err := fs.Sub(web.Templates, "templates")
	if err != nil {
		t.Fatal(err)
	}
	catalog := appi18n.NewCatalog("en")
	views, err := view.Load(tmplFS, catalog)
	if err != nil {
		t.Fatal(err)
	}
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("go.mod not found")
		}
		root = parent
	}

	a, err := New(cais.Config{Env: "test", Locale: "en", AppURL: "http://cifra.test"}, Deps{
		Views:     views,
		Store:     s,
		StaticDir: filepath.Join(root, "web", "static"),
		Site:      meta.SiteFrom("Cifra", "http://cifra.test"),
		Catalog:   catalog,
		AppSecret: []byte("test-secret-not-for-prod"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestApp_unknownPathIs404(t *testing.T) {
	h := testApp(t).Handler()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/.env", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("GET /.env status=%d want 404 (home must not catch-all)", rr.Code)
	}

	ok := httptest.NewRecorder()
	h.ServeHTTP(ok, httptest.NewRequest(http.MethodGet, "/", nil))
	if ok.Code != http.StatusOK {
		t.Fatalf("GET / status=%d want 200", ok.Code)
	}
}
