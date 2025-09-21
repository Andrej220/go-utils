package srvx

import(
	"encoding/json"
	"net/http"
)

const (
    ErrCodeEmptyBody            = "empty_body"
    ErrCodeInvalidJSON          = "invalid_json"
    ErrCodeUnsupportedMediaType = "unsupported_media_type"
	ErrValidationFailed			= "validation_failed"
)

type APIError struct {
    Code    string      `json:"error"`
    Message string      `json:"message"`
    Status  int         `json:"-"`
    Details interface{} `json:"details,omitempty"`
}

func WriteJSONError(w http.ResponseWriter, e APIError) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(e.Status)
    _ = json.NewEncoder(w).Encode(e)
}

//func writeJSONError(w http.ResponseWriter, code int, err, msg string) {
//    w.Header().Set("Content-Type", "application/json")
//    w.WriteHeader(code)
//    _ = json.NewEncoder(w).Encode(map[string]any{"error": err, "message": msg})
//}
