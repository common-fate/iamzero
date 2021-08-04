package healthcheck_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/common-fate/iamzero/pkg/healthcheck"
)

func TestStatusString(t *testing.T) {
	tests := map[Status]string{
		Unavailable: "unavailable",
		Ready:       "ready",
		Broken:      "broken",
		Status(-1):  "unknown",
	}
	for k, v := range tests {
		assert.Equal(t, v, k.String())
	}
}

func TestStatusSetGet(t *testing.T) {
	hc := New()
	assert.Equal(t, Unavailable, hc.Get())

	hc = New()
	assert.Equal(t, Unavailable, hc.Get())

	hc.Set(Ready)
	assert.Equal(t, Ready, hc.Get())
}

func TestHealthCheck_Handler_ContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	New().Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	resp := rec.Result()

	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
}
