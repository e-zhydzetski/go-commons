package xhttp

import "net/http"

func AllowAllCORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, PUT, POST, DELETE, HEAD")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Vary", "Origin, Access-Control-Request-Method")
				if r.Method == "OPTIONS" { // preflight request
					w.WriteHeader(http.StatusOK)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
