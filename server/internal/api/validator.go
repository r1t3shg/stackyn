package api

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// ValidateRequest validates a struct using the validator package
func ValidateRequest(logger *zap.Logger, w http.ResponseWriter, r *http.Request, req interface{}) bool {
	if err := validate.Struct(req); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		logger.Warn("Validation failed",
			zap.String("path", r.URL.Path),
			zap.Error(err),
		)

		// Return validation errors
		respondWithError(w, http.StatusBadRequest, "Validation failed", validationErrors)
		return false
	}
	return true
}

// respondWithError is a helper to send error responses
func respondWithError(w http.ResponseWriter, status int, message string, details interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// TODO: Implement proper JSON error response
	w.Write([]byte(`{"error":"` + message + `"}`))
}

