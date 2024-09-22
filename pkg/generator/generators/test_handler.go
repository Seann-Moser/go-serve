package generators

import (
	"fmt"
	"net/http"
)

func HandlerFuncs(w http.ResponseWriter, r *http.Request) {
	// Set the content type to plain text
	w.Header().Set("Content-Type", "text/plain")

	// Respond with "pong"
	fmt.Fprintln(w, "pong")
}
