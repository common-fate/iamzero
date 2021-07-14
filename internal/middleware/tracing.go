package middleware

import (
	"net/http"

	"github.com/go-chi/chi"
	"go.opentelemetry.io/otel/trace"
)

// Tracing is a middleware which adds the Chi route to the OpenTelemetry
// span generated for the http.Handler
func Tracing(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		rctx := chi.RouteContext(r.Context())
		routePattern := rctx.RoutePattern()
		span := trace.SpanFromContext(r.Context())

		// set the name of the span to be the Chi router path
		span.SetName(routePattern)

		// Alternatively we can set the http.route attribute on the span as per below.
		// span.SetAttributes(semconv.HTTPRouteKey.String(routePattern))

	}
	return http.HandlerFunc(fn)
}
