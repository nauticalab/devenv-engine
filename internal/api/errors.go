package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// respondJSON sends a JSON response with the given status code
func respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if payload != nil {
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
	}
}

// respondError sends an error response in JSON format
func respondError(w http.ResponseWriter, code int, message string) {
	respondJSON(w, code, ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	})
}

// respondBadRequest sends a 400 Bad Request error
func respondBadRequest(w http.ResponseWriter, message string) {
	respondError(w, http.StatusBadRequest, message)
}

// respondUnauthorized sends a 401 Unauthorized error
func respondUnauthorized(w http.ResponseWriter, message string) {
	respondError(w, http.StatusUnauthorized, message)
}

// respondForbidden sends a 403 Forbidden error
func respondForbidden(w http.ResponseWriter, message string) {
	respondError(w, http.StatusForbidden, message)
}

// respondNotFound sends a 404 Not Found error
func respondNotFound(w http.ResponseWriter, message string) {
	respondError(w, http.StatusNotFound, message)
}

// respondInternalError sends a 500 Internal Server Error
func respondInternalError(w http.ResponseWriter, message string) {
	respondError(w, http.StatusInternalServerError, message)
}

// respondSuccess sends a 200 OK with payload
func respondSuccess(w http.ResponseWriter, payload interface{}) {
	respondJSON(w, http.StatusOK, payload)
}

// respondCreated sends a 201 Created with payload
func respondCreated(w http.ResponseWriter, payload interface{}) {
	respondJSON(w, http.StatusCreated, payload)
}

// respondNoContent sends a 204 No Content (no body)
func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
