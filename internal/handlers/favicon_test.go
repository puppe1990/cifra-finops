package handlers

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAppHTML_linksFaviconSVG(t *testing.T) {
	root := projectRoot(t)
	html, err := os.ReadFile(filepath.Join(root, "web/templates/layouts/app.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(html, []byte(`rel="icon" href="/static/icons/favicon.svg" type="image/svg+xml"`)) {
		t.Fatal("app.html missing SVG favicon")
	}
	if !bytes.Contains(html, []byte(`rel="apple-touch-icon" href="/static/icons/apple-touch-icon.png"`)) {
		t.Fatal("app.html missing apple-touch-icon")
	}
}

func TestFaviconSVG_usesCifraBrandColors(t *testing.T) {
	root := projectRoot(t)
	svg, err := os.ReadFile(filepath.Join(root, "web/static/icons/favicon.svg"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(svg, []byte("<svg")) {
		t.Fatal("not an svg")
	}
	for _, color := range []string{"#0B100E", "#E08A45"} {
		if !bytes.Contains(svg, []byte(color)) {
			t.Errorf("missing brand color %s", color)
		}
	}
}
