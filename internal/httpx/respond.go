// Package httpx contains small HTTP helpers shared by the gateway and any
// service that exposes REST endpoints directly.
package httpx

import (
	"encoding/json"
	"net/http"

	apperr "github.com/dimasd-angga/go-microkit/internal/errors"
)

type errorBody struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func Error(w http.ResponseWriter, err error) {
	if e := apperr.As(err); e != nil {
		JSON(w, e.HTTPStatus(), errorBody{
			Error:   http.StatusText(e.HTTPStatus()),
			Code:    e.Code,
			Message: e.Message,
		})
		return
	}
	JSON(w, http.StatusInternalServerError, errorBody{
		Error: "Internal Server Error",
		Code:  "internal_error",
	})
}

// DecodeJSON pulls JSON from r.Body into v with a sane size limit.
func DecodeJSON(r *http.Request, v any, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = 1 << 20 // 1 MiB
	}
	r.Body = http.MaxBytesReader(nil, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
