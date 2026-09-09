package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais/meta"

	"github.com/puppe1990/cifra-finops/internal/store"
)

func testSite() meta.Site {
	return meta.Site{AppName: "Cifra", AppURL: "https://cais.example.com"}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("go.mod not found")
		}
		wd = parent
	}
}

func setupTestRenderer(t *testing.T) *view.Renderer {
	return setupTestViews(t)
}

func setupTestStore(t *testing.T) store.Store {
	t.Helper()
	s, err := store.NewSQLiteStore(":memory:", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func navLinkClass(body, href string) string {
	needle := `href="` + href + `"`
	i := strings.Index(body, needle)
	if i < 0 {
		return ""
	}
	start := strings.LastIndex(body[:i], "<a ")
	end := strings.Index(body[i:], ">")
	if start < 0 || end < 0 {
		return ""
	}
	return body[start : i+end]
}

func assertPasswordEyeToggles(t *testing.T, body string, want int) {
	t.Helper()
	if n := strings.Count(body, `type="password"`); n != want {
		t.Errorf("password inputs = %d, want %d", n, want)
	}
	if n := strings.Count(body, `amarra-hook="password"`); n != want {
		t.Errorf("password toggles = %d, want %d", n, want)
	}
	if !strings.Contains(body, "<svg") {
		t.Error("password toggle missing svg eye")
	}
	if strings.Contains(body, ">show</button>") {
		t.Error("password toggle still uses text show")
	}
}
