package tunneltransport

import (
	"crypto/sha256"
	"crypto/subtle"
	"io"
	"net/http"

	"github.com/oklog/ulid/v2"
)

// basicAuthChallenge is the WWW-Authenticate value sent with every 401. Without
// it browsers never show their credential prompt. It is a constant so it can
// never be built from request data and smuggle a quote into the header.
// charset="UTF-8" is the only value RFC 7617 defines and makes modern browsers
// encode non-ASCII credentials as UTF-8.
const basicAuthChallenge = `Basic realm="Restricted", charset="UTF-8"`

// basicAuthGate enforces a tunnel's basic_auth credential. A nil gate means the
// tunnel is unprotected and lets everything through.
type basicAuthGate struct {
	expected [sha256.Size]byte
}

// newBasicAuthGate takes Tunnel.ResolvedBasicAuth's value directly, where an
// empty credential means the tunnel is unprotected. The credential is hashed
// once here rather than on every request.
func newBasicAuthGate(credential string) *basicAuthGate {
	if credential == "" {
		return nil
	}

	return &basicAuthGate{expected: sha256.Sum256([]byte(credential))}
}

// allow reports whether the request carries the tunnel's credential.
//
// The comparison runs over SHA-256 digests rather than the raw strings so it is
// fixed-width and cannot leak the credential length through timing.
func (g *basicAuthGate) allow(request *http.Request) bool {
	if g == nil {
		return true
	}

	// request.BasicAuth decodes the header for us; comparing the raw header
	// would break on legal base64 and whitespace variations.
	username, password, ok := request.BasicAuth()
	if !ok {
		return false
	}

	// BasicAuth splits on the first colon, so username never contains one and
	// rejoining reproduces the configured credential exactly. It also means a
	// credential with no colon can never be reproduced by any request, so a
	// malformed basic_auth locks the tunnel instead of weakening it.
	received := sha256.Sum256([]byte(username + ":" + password))

	return subtle.ConstantTimeCompare(received[:], g.expected[:]) == 1
}

// rejectUnauthorized challenges the caller and records the attempt.
//
// The body stays plain text: the pages in internal/utils/error-templates exist
// to tell the tunnel owner that something is broken, whereas a 401 is the
// feature working correctly and is shown to a stranger. Browsers only render it
// if the user dismisses the credential prompt.
//
// X-Portr-Error is deliberately not set. A 401 is not a portr failure, and
// replayFailure treats any unrecognised error reason as a 502.
func (s *Client) rejectUnauthorized(writer http.ResponseWriter, request *http.Request) {
	header := writer.Header()
	header.Set("WWW-Authenticate", basicAuthChallenge)
	header.Set("Content-Type", "text/plain; charset=utf-8")
	header.Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(http.StatusUnauthorized)
	_, _ = io.WriteString(writer, "401 Unauthorized\n")

	// Rejected requests never reach the reverse proxy, so without this the
	// inspector stays empty and a working gate looks like a broken tunnel.
	// Checked before the clone so unlogged tunnels pay nothing. The guessed
	// credential is stored redacted: Authorization is in DefaultRedactHeaders.
	if !s.requestLoggingEnabled() {
		return
	}

	s.submitCapture(httpCaptureTask{
		id:      ulid.Make().String(),
		request: cloneRequestForLog(request),
		response: &http.Response{
			StatusCode: http.StatusUnauthorized,
			Proto:      request.Proto,
			Header:     header.Clone(),
		},
	})
}
