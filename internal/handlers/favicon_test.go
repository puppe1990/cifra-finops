package handlers

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAppHTML_doesNotHardcodeFinOpsAWS(t *testing.T) {
	root := projectRoot(t)
	for _, rel := range []string{
		"web/templates/layouts/app.html",
		"web/templates/layouts/public.html",
	} {
		html, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(html, []byte("FinOps AWS")) {
			t.Errorf("%s still hardcodes FinOps AWS", rel)
		}
	}
}

func TestAppHTML_localeToggleSkipsDrive(t *testing.T) {
	root := projectRoot(t)
	html, err := os.ReadFile(filepath.Join(root, "web/templates/layouts/app.html"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(html, []byte(`action="/locale"`)) != 2 {
		t.Fatal("expected EN and PT locale forms")
	}
	if bytes.Count(html, []byte(`action="/locale" method="post" data-amarra-skip`)) != 2 {
		t.Fatal("locale forms must skip Drive so the selected language box (outside #amarra-main) re-renders")
	}
}

func TestAppHTML_mobileNavHidesScrollbar(t *testing.T) {
	root := projectRoot(t)
	html, err := os.ReadFile(filepath.Join(root, "web/templates/layouts/app.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(html, []byte(`shell-nav`)) {
		t.Fatal("mobile nav missing shell-nav (hide scrollbar + edge fade)")
	}
}

func TestLayouts_linkCifraPWAIcons(t *testing.T) {
	root := projectRoot(t)
	partial, err := os.ReadFile(filepath.Join(root, "web/templates/partials/pwa_icons.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`rel="manifest" href="/static/manifest.webmanifest"`,
		`rel="icon" href="/static/icons/favicon.svg" type="image/svg+xml"`,
		`rel="apple-touch-icon" href="/static/icons/apple-touch-icon.png"`,
	} {
		if !bytes.Contains(partial, []byte(want)) {
			t.Errorf("pwa_icons.html missing %s", want)
		}
	}
	for _, rel := range []string{
		"web/templates/layouts/app.html",
		"web/templates/layouts/auth.html",
		"web/templates/layouts/public.html",
	} {
		html, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(html, []byte(`{{ template "pwa_icons" . }}`)) {
			t.Errorf("%s missing pwa_icons partial", rel)
		}
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
