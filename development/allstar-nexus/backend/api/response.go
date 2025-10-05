package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error *errorBody  `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope{OK: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope{OK: false, Error: &errorBody{Code: code, Message: msg}})
}

// validatePassword enforces minimal password rules.
func validatePassword(pw string) error {
	if len(pw) < 8 {
		return errStr("password must be at least 8 characters")
	}
	lower := strings.ToLower(pw)
	// reject trivial common passwords (small hard-coded list for now)
	common := []string{"password", "12345678", "qwerty", "letmein", "admin", "welcome"}
	for _, c := range common {
		if lower == c {
			return errStr("password too common")
		}
	}
	return nil
}

type errStr string

func (e errStr) Error() string { return string(e) }
