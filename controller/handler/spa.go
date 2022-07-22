package handler

import (
	"html/template"
	"log"
	"net/http"

	"github.com/TwiN/gatus/v4/config/ui"
)

func SinglePageApplication(staticFolder string, ui *ui.Config) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		t, err := template.ParseFiles(staticFolder + "/index.html")
		if err != nil {
			log.Println("[handler][SinglePageApplication] Failed to parse template:", err.Error())
			http.ServeFile(writer, request, staticFolder+"/index.html")
			return
		}
		writer.Header().Set("Content-Type", "text/html")
		err = t.Execute(writer, ui)
		if err != nil {
			log.Println("[handler][SinglePageApplication] Failed to parse template:", err.Error())
			http.ServeFile(writer, request, staticFolder+"/index.html")
			return
		}
	}
}
