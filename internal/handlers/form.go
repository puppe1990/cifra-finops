package handlers

import (
	"net/http"

	"github.com/puppe1990/amarra-cais/pkg/cais/httpx"
)

// parseFormBody fills r.Form from urlencoded, multipart, or JSON bodies so
// FormValue works for Amarra Drive posts. Cais v0.2+ ParseFormOrJSON already
// reads multipart (#26).
func parseFormBody(r *http.Request) error {
	return httpx.ParseFormOrJSON(r)
}
