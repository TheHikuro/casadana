package httpserver

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/httprate"
)

// ErrRateLimited answers a caller that has spent its budget for the current
// window. Registered here rather than by each caller so every rate-limited
// route replies with the same code and the same JSON shape as any other error.
var ErrRateLimited = errors.New("too many requests, please try again later")

func init() {
	Register(ErrRateLimited, http.StatusTooManyRequests, "RATE_LIMITED")
}

// RateLimit caps each caller at requestLimit requests per window.
//
// The count is a sliding window, not a calendar one: a fixed window lets a
// caller spend a whole budget in its last second and a second budget in the
// next, which is twice the intended rate at the moment it matters.
func RateLimit(requestLimit int, window time.Duration) func(http.Handler) http.Handler {
	// Both handlers are overridden: httprate's defaults answer in plain text
	// (429 "Too Many Requests", and 412 for an internal failure), which would be
	// the only two responses in the API a client couldn't parse like the rest.
	// The rate-limit headers httprate sets survive — it stages them before
	// handing over, and neither handler touches them.
	return httprate.LimitBy(requestLimit, window, clientIPKey,
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			WriteError(w, r, ErrRateLimited)
		}),
		httprate.WithErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
			WriteError(w, r, err)
		}),
	)
}

// clientIPKey is the bucket a request counts against.
//
// IPv6 callers are bucketed by /64: a single subscriber is routinely handed a
// whole /64, so keying on the full address would hand them a fresh budget per
// request just by picking another address in their own prefix.
func clientIPKey(r *http.Request) (string, error) {
	return httprate.CanonicalizeIP(clientIP(r)), nil
}

// clientIP resolves who is calling, given that this API is never reached
// directly: nginx fronts it in production (docs/adr/0002) and Caddy in
// development, so r.RemoteAddr is the proxy's own address and is worthless as a
// rate-limit key.
//
// The order below is by how forgeable each source is, not by precision:
//
//  1. the rightmost entry of X-Forwarded-For. Proxies *append* the peer they
//     accepted (nginx's $proxy_add_x_forwarded_for, Caddy's default), so the
//     last entry is the only one our own edge wrote — a caller can inject
//     entries ahead of it but cannot append after it.
//  2. X-Real-IP, for an edge that sets it instead of appending to XFF. Trusted
//     only because our edge overwrites whatever the caller sent; that overwrite
//     is a deployment requirement, not a nicety.
//  3. the peer address, for a direct call — in practice local development.
//
// Deliberately not chi's middleware.RealIP (already installed upstream, which
// is why r.RemoteAddr is not consulted here): it reads True-Client-IP first,
// then X-Real-IP, then the *first* XFF entry. Our edge sets no True-Client-IP
// and appends to XFF, so two of its three sources are caller-supplied — a
// spammer could rotate them and draw a fresh bucket on every request.
func clientIP(r *http.Request) string {
	if ip := lastForwardedFor(r.Header.Get("X-Forwarded-For")); ip != "" {
		return ip
	}
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(ip) != nil {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// lastForwardedFor reads the rightmost X-Forwarded-For entry and only that one.
// Walking further left on a malformed tail would start trusting entries the
// caller wrote, which is the whole thing this ordering exists to avoid.
func lastForwardedFor(header string) string {
	entries := strings.Split(header, ",")
	last := strings.TrimSpace(entries[len(entries)-1])
	if net.ParseIP(last) == nil {
		return ""
	}
	return last
}
