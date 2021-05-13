package io

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// RespondJSON converts a Go value to JSON and sends it to the client.
func RespondJSON(ctx context.Context, log *zap.SugaredLogger, w http.ResponseWriter, data interface{}, statusCode int) {
	// If there is nothing to marshal then set status code and return.
	if statusCode == http.StatusNoContent {
		w.WriteHeader(statusCode)
		return
	}

	// Convert the response value to JSON.
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.With("err", err).Error("marshalling JSON")
	}

	// Set the content type and headers once we know marshaling has succeeded.
	w.Header().Set("Content-Type", "application/json")

	// Write the status code to the response.
	w.WriteHeader(statusCode)

	// Send the result back to the client.
	if _, err := w.Write(jsonData); err != nil {
		log.With("err", err).Error("writing response")
	}
}

// RespondText returns a text response back to the client.
func RespondText(ctx context.Context, log *zap.SugaredLogger, w http.ResponseWriter, text string, statusCode int) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)
	if _, err := w.Write([]byte(text)); err != nil {
		log.With("err", err).Error("writing response")
	}
}

// RespondError sends an error reponse back to the client and logs the error internally
// if the error is of type io.Error we send it's message back to the client.
// Otherwise, we return a HTTP 500 code with an opaque response to avoid leaking any
// information from the server.
func RespondError(ctx context.Context, log *zap.SugaredLogger, w http.ResponseWriter, err error) {
	log.With("err", err).Error("web handler error")

	// If the error was of the type *Error, the handler has
	// a specific status code and error to return.
	if webErr, ok := errors.Cause(err).(*Error); ok {
		er := ErrorResponse{
			Error:  webErr.Err.Error(),
			Fields: webErr.Fields,
		}
		RespondJSON(ctx, log, w, er, webErr.Status)
		return
	}

	// If not, the handler sent any arbitrary error value so use 500.
	er := ErrorResponse{
		Error: http.StatusText(http.StatusInternalServerError),
	}
	RespondJSON(ctx, log, w, er, http.StatusInternalServerError)
}
