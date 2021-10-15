package handler

import (
	"html/template"
	"log"
	"net/http"

	"github.com/TwiN/gatus/v3/config/ui"
	"github.com/TwiN/gatus/v3/web"
)

func SinglePageApplication(ui *ui.Config) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		t, err := template.ParseFS(web.StaticFolder, "index.html")
		if err != nil {
			log.Println("[handler][SinglePageApplication] Failed to parse template:", err.Error())
			return
		}
		writer.Header().Set("Content-Type", "text/html")
		err = t.Execute(writer, ui)
		if err != nil {
			log.Println("[handler][SinglePageApplication] Failed to parse template:", err.Error())
			return
		}
	}
}
