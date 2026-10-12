package consoleapi

import (
	"encoding/json"
	"github.com/daeuniverse/dae-wing/observability"
	"net/http"
)

func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, msg string) {
	reply(w, status, map[string]string{"error": msg})
}
func Handler(logs *observability.Store, directory string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value("user") == nil || r.Context().Value("role") != "admin" {
			fail(w, 401, "authentication required")
			return
		}
		switch r.URL.Path {
		case "/api/groups/resources", "/api/groups/preview":
			groupResourcesAPI(w, r)
		case "/api/config/bundle", "/api/config/bundle/preview", "/api/config/dae":
			bundleAPI(w, r)
		case "/api/geodata", "/api/geodata/source", "/api/geodata/update", "/api/geodata/update-all":
			geodataAPI(w, r, directory)
		default:
			logs.Handler().ServeHTTP(w, r)
		}
	})
}
