package handler

import (
	_ "embed"
	"html/template"
	"log"
	"net/http"

	"github.com/TwiN/gatus/v5/config/ui"
	static "github.com/TwiN/gatus/v5/web"
)

func SinglePageApplication(ui *ui.Config) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		t, err := template.ParseFS(static.FileSystem, static.IndexPath)
		if err != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			log.Println("[handler][SinglePageApplication] Failed to parse template. This should never happen, because the template is validated on start. Error:", err.Error())
			http.Error(writer, "Failed to parse template. This should never happen, because the template is validated on start.", http.StatusInternalServerError)
			return
		}
		writer.Header().Set("Content-Type", "text/html")
		err = t.Execute(writer, ui)
		if err != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			log.Println("[handler][SinglePageApplication] Failed to execute template. This should never happen, because the template is validated on start. Error:", err.Error())
			http.Error(writer, "Failed to execute template. This should never happen, because the template is validated on start.", http.StatusInternalServerError)
			return
		}
	}
}
