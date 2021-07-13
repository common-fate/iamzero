package middleware

import (
	"net/http"

	"github.com/common-fate/iamzero/api/io"
	"github.com/common-fate/iamzero/pkg/tokens"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// CollectorTokenAuth is a middleware which returns a HTTP 403 response if the provided
// token header x-iamzero-token does not match a token from the TokerStorer
func CollectorTokenAuth(storer tokens.TokenStorer, log *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			tokenID := r.Header.Get("x-iamzero-token")

			_, err := storer.Get(ctx, tokenID)

			if errors.Cause(err) == tokens.ErrTokenNotFound {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			} else if err != nil {
				io.RespondError(ctx, log, w, err)
			} else {
				next.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(fn)
	}
}
