package platform

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

// Response is the standard API envelope.
type Response struct {
	Data  any    `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data})
}

// JSONList writes a paginated JSON response.
func JSONList(w http.ResponseWriter, status int, data any, meta Meta) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Data: data, Meta: &meta})
}

// JSONError writes an error response.
func JSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Error: &Error{Code: code, Message: message}})
}

// HandleError writes an AppError as JSON, or falls back to 500 for unknown errors.
func HandleError(w http.ResponseWriter, err error) {
	if appErr, ok := AsAppError(err); ok {
		JSONError(w, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	slog.Error("unhandled error", "error", err)
	JSONError(w, http.StatusInternalServerError, CodeInternal, "an unexpected error occurred")
}

// Pagination holds parsed pagination params.
type Pagination struct {
	Page    int
	PerPage int
	Offset  int
}

// ParsePagination extracts page and per_page from query params.
func ParsePagination(r *http.Request) Pagination {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	return Pagination{
		Page:    page,
		PerPage: perPage,
		Offset:  (page - 1) * perPage,
	}
}

// Decode reads JSON from a request body into the target.
func Decode(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return NewBadRequest("invalid request body: " + err.Error())
	}
	return nil
}
