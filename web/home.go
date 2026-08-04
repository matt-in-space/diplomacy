package web

import "net/http"

func handleHome(w http.ResponseWriter, r *http.Request) {
	if err := homeTemplate.ExecuteTemplate(w, "layout", newPageData(w, r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
