package utils

import (
	"encoding/json"
	"net/http"
)

func RespondJSON(w http.ResponseWriter, status int, data interface{}) error {
	response, err := json.Marshal(data)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, _ = w.Write(response)

	return nil
}

func Unauthorized(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(message))
}

func NotFound(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(message))
}

func BadRequest(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(message))
}
