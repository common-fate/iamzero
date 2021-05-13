package middleware

import (
	"net/http"
)

// SimpleTokenAuth is a middleware which returns a HTTP 403 response if the provided
// token header x-iamzero-token does not match the configured server token
//
// TODO:AUTH Intended to be modified and removed for a more comprehensive authn/authz system
// in future after alpha.
func SimpleTokenAuth(token string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {

			provided := r.Header.Get("x-iamzero-token")
			if provided != token {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			} else {
				next.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(fn)
	}
}
