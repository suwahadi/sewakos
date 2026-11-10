// Package response implements the single JSON envelope every handler must use
// (SPEC-GO.md §5.2-§5.4): no handler writes JSON directly.
package response

import (
	"encoding/json"
	"net/http"
)

// Meta is the envelope's meta block.
type Meta struct {
	ResponseCode int    `json:"response_code"`
	Code         int    `json:"code"`
	Severity     string `json:"severity"`
	Message      string `json:"message"`
}

// Severity maps an HTTP status to the severity SPEC-GO.md §5.2 requires:
// 2xx -> OK, 4xx -> WARNING, 5xx -> ERROR.
func Severity(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "OK"
	case status >= 400 && status < 500:
		return "WARNING"
	default:
		return "ERROR"
	}
}

func meta(status int, message string) Meta {
	if message == "" {
		message = "OK"
	}
	return Meta{ResponseCode: status, Code: status, Severity: Severity(status), Message: message}
}

func write(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Envelope fields are always well-formed Go values; json.Marshal cannot fail here,
	// matching SPEC-GO.md §5.2's requirement that every response goes through this
	// one package rather than ad-hoc json.NewEncoder calls in handlers.
	_ = json.NewEncoder(w).Encode(body)
}

// Success writes a 2xx envelope. data is merged into the top-level body (e.g.
// {"property": ...}), matching every documented contract example.
func Success(w http.ResponseWriter, status int, message string, data map[string]any) {
	body := map[string]any{"status": true, "meta": meta(status, message)}
	for k, v := range data {
		body[k] = v
	}
	write(w, status, body)
}

// Fail writes a 4xx/5xx envelope. messages is the optional per-field validation
// map (SPEC-GO.md §5.4); pass nil to omit the "messages" key entirely, or an
// explicit nil value where the contract requires `"messages": null`.
func Fail(w http.ResponseWriter, status int, message string, messages map[string][]string) {
	body := map[string]any{"status": false, "meta": meta(status, message)}
	if messages != nil {
		body["messages"] = messages
	}
	write(w, status, body)
}

// FailWithNullMessages writes a 4xx/5xx envelope with an explicit `"messages": null`,
// e.g. login failures (SPEC-GO.md §5.3).
func FailWithNullMessages(w http.ResponseWriter, status int, message string) {
	write(w, status, map[string]any{
		"status":   false,
		"meta":     meta(status, message),
		"messages": nil,
	})
}

// Known error messages (SPEC-GO.md §5.3), reused verbatim by middleware and handlers.
const (
	MsgValidationFailed    = "Validasi gagal."
	MsgLoginFailed         = "Nomor handphone atau password salah."
	MsgUnknownClient       = "Klien tidak dikenal."
	MsgInvalidSession      = "Sesi tidak valid. Silakan login kembali."
	MsgAccessDenied        = "Akses ditolak."
	MsgNotFound            = "Data tidak ditemukan."
	MsgMethodNotAllowed    = "Metode tidak diizinkan."
	MsgRateLimited         = "Terlalu banyak permintaan. Silakan coba lagi nanti."
	MsgInternalError       = "Terjadi kesalahan pada server."
	MsgPaymentGatewayError = "Layanan pembayaran sedang bermasalah. Silakan coba lagi."
	MsgAccountSuspended    = "Akun dinonaktifkan."
)

// NotFoundHandler is the router-wide 404 handler (SPEC-GO.md §5.3), used both as
// chi's NotFound handler and for in-handler "resource doesn't exist" responses.
func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	Fail(w, http.StatusNotFound, MsgNotFound, nil)
}

// MethodNotAllowedHandler is the router-wide 405 handler (SPEC-GO.md §5.3).
func MethodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	Fail(w, http.StatusMethodNotAllowed, MsgMethodNotAllowed, nil)
}

// InternalErrorHandler is used by the recovery middleware (B1.6): error detail
// never reaches the response body, only the log (SPEC-GO.md §5.3).
func InternalErrorHandler(w http.ResponseWriter, r *http.Request) {
	Fail(w, http.StatusInternalServerError, MsgInternalError, nil)
}
