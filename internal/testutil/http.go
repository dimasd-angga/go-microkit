//go:build integration

package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Do is a thin convenience around httptest.NewRecorder for handler tests.
func Do(t *testing.T, h http.Handler, method, path string, body any) *http.Response {
	t.Helper()

	var rdr io.Reader
	if body != nil {
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			t.Fatal(err)
		}
		rdr = buf
	}

	req := httptest.NewRequest(method, path, rdr).WithContext(context.Background())
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Result()
}

func DecodeJSON(t *testing.T, r *http.Response, v any) {
	t.Helper()
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}
