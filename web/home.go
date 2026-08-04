package web

import (
	"html/template"
	"net/http"
)

var homeTemplate = template.Must(template.ParseFS(templateFS, "templates/home.html"))

func handleHome(w http.ResponseWriter, r *http.Request) {
	if err := homeTemplate.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
