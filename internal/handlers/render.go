package handlers

import (
	"net/http"

	"github.com/puppe1990/amarra-cais/pkg/amarra/view"
	"github.com/puppe1990/amarra-cais/pkg/cais"
	"github.com/puppe1990/amarra-cais/pkg/cais/csrf"
	"github.com/puppe1990/amarra-cais/pkg/cais/flash"
)

func writePage(w http.ResponseWriter, r *http.Request, views *view.Renderer, cfg cais.Config, layout, name string, data map[string]any, status ...int) {
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["CSRFToken"]; !ok {
		data["CSRFToken"] = csrf.TokenFromRequest(r)
	}
	if _, ok := data["Flash"]; !ok {
		if msg, ok := flash.MessageFromRequest(r); ok {
			data["Flash"] = msg.Message
			data["FlashKind"] = msg.Kind
		}
	}
	if layout == "" {
		layout = "app"
	}
	if _, ok := data["ActiveNav"]; !ok {
		data["ActiveNav"] = name
	}
	if _, ok := data["labels"]; !ok {
		data["labels"] = map[string]string{}
	}
	st := 0
	if len(status) > 0 {
		st = status[0]
	}
	view.Write(w, r, views, view.Page{Layout: layout, Name: name, Data: data, Status: st}, cfg)
}
