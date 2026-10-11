package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/vavallee/bindery/internal/httpsec"
)

// upstreamDetailMax caps the upstream text a client may see, in bytes.
const upstreamDetailMax = 200

// writeNotFoundOrServerError answers a lookup that returned (value, err):
// a non-nil err is a server fault (generic 500, the error is logged), and
// found == false with no error is a plain 404. Conflating the two, as the
// older "err != nil || x == nil" shape does, reports a database outage as a
// missing record.
func writeNotFoundOrServerError(w http.ResponseWriter, r *http.Request, err error, found bool) {
	if err != nil {
		writeServerError(w, r, err)
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

// decodeJSON decodes the request body into v and reports whether the handler
// may continue. A body past the server's size cap answers 413; any other
// decode failure answers 400 with a fixed message, so the decoder's own text
// (which can quote the payload) never reaches the client. On false the
// response is already written.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	err := json.NewDecoder(r.Body).Decode(v)
	if err == nil {
		return true
	}
	writeBodyReadError(w, err)
	return false
}

// writeBodyReadError answers a failure to read or decode a request body:
// 413 when the body ran past the cap installed by MaxRequestBody or
// WithMaxBody (http.MaxBytesReader reports *http.MaxBytesError), otherwise a
// 400 with a fixed message. Handlers that buffer the body themselves
// (io.ReadAll) use this directly; everything else goes through decodeJSON.
func writeBodyReadError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "request body too large"})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
}

// upstreamDetail reduces text that came from another service (a download
// client, Prowlarr) to something safe to show: credentials redacted, the first
// line only, at most upstreamDetailMax bytes. Upstream bodies can be HTML
// error pages or stack traces, and a first line is enough for an operator to
// see what the service objected to.
func upstreamDetail(msg string) string {
	msg = httpsec.RedactSecrets(msg)
	if i := strings.IndexAny(msg, "\r\n"); i >= 0 {
		msg = msg[:i]
	}
	msg = strings.TrimSpace(msg)
	if len(msg) > upstreamDetailMax {
		// Cut on a rune boundary so a multi-byte character is never split.
		cut := upstreamDetailMax
		for cut > 0 && msg[cut]&0xC0 == 0x80 {
			cut--
		}
		msg = msg[:cut]
	}
	return msg
}

// writeUpstreamDetail is for connection-test style endpoints, where the
// operator needs some of what the other service said. err is logged in full;
// the client gets the first line of msg (see upstreamDetail). Endpoints that
// do not need the detail use writeUpstreamError instead.
func writeUpstreamDetail(w http.ResponseWriter, r *http.Request, status int, msg string, err error) {
	slog.Warn("upstream request failed", "method", r.Method, "path", r.URL.Path, "error", err)
	writeJSON(w, status, map[string]string{"error": upstreamDetail(msg)})
}
