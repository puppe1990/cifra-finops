package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/puppe1990/amarra-cais/pkg/cais/httpx"
)

// parseFormBody parses the request body (urlencoded/multipart/JSON) and
// mirrors body fields into r.Form so FormValue works for Amarra Drive
// multipart posts — net/http's ParseForm alone does not read multipart
// bodies, and FormValue only consults r.Form.
func parseFormBody(r *http.Request) error {
	if err := httpx.ParseFormOrJSON(r); err != nil {
		return err
	}
	if r.MultipartForm == nil && strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return err
		}
	}
	if r.PostForm != nil {
		if r.Form == nil {
			r.Form = make(url.Values)
		}
		for k, vs := range r.PostForm {
			r.Form[k] = append(r.Form[k], vs...)
		}
	}
	return nil
}
