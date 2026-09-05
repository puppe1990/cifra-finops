package i18n

import (
	caisi18n "github.com/puppe1990/amarra-cais/pkg/cais/i18n"
)

var locales = map[string]map[string]string{
	"en": enMessages,
	"pt": ptMessages,
}

// NewCatalog returns a catalog for the given locale (en default, pt for pt-BR).
func NewCatalog(locale string) *caisi18n.Catalog {
	return caisi18n.NewCatalogFrom(locale, locales)
}

// DefaultCatalog returns the English catalog.
func DefaultCatalog() *caisi18n.Catalog {
	return NewCatalog(caisi18n.DefaultLocale)
}

// Labels returns a copy of all UI strings for the locale (en or pt).
func Labels(locale string) map[string]string {
	cat := NewCatalog(locale)
	src := locales[cat.Locale()]
	if src == nil {
		src = locales["en"]
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
