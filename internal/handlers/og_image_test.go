package handlers

import (
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestOGImage_isCifraBrandCard(t *testing.T) {
	root := projectRoot(t)
	f, err := os.Open(filepath.Join(root, "web/static/og.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Errorf("close og.png: %v", err)
		}
	}()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 1200 || cfg.Height != 630 {
		t.Fatalf("og.png = %dx%d, want 1200x630", cfg.Width, cfg.Height)
	}

	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	r16, g16, b16, _ := img.At(24, 24).RGBA()
	r, g, b := r16>>8, g16>>8, b16>>8
	// Scaffold og.png is a purple bar (#6c63ff-ish). Cifra ink is near #0B100E.
	if r > 40 || g > 50 || b > 80 {
		t.Fatalf("og.png corner is rgb(%d,%d,%d), want dark ink not the purple scaffold", r, g, b)
	}
}
