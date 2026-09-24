package tucapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ═════════════════════════════════════════════════════════════════════════════
// WEBHOOKS: THE DELICATE PART
//
// An unverified webhook is an endpoint where anyone can post a fake
// "payout.confirmed" and make you settle money that never moved. The
// signature is the only thing that tells our event from an invented one.
//
// The signature is HMAC-SHA256 over `timestamp + "." + raw body`, hex, keyed
// with your webhook secret, and travels with a version prefix:
//
//	X-Firma: v1=8a7f3c...
//
// The timestamp is INSIDE what is signed, not only in a header: otherwise a
// captured event could be replayed forever with a fresh timestamp.
//
// The two classic mistakes this file avoids: comparing with ==, which leaks
// how many bytes matched (hmac.Equal takes constant time), and verifying over
// re-serialised JSON, which changes the bytes.
// ═════════════════════════════════════════════════════════════════════════════

// ErrInvalidSignature: the event does not verify. DISCARD IT, do not process it.
var ErrInvalidSignature = errors.New("payments: the webhook signature is not valid")

// ParseEvent reads and VERIFIES a webhook request. It consumes the body: if
// you need the raw bytes afterwards, read them yourself and use VerifyEvent.
func ParseEvent(r *http.Request, secret string) (Event, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return Event{}, fmt.Errorf("payments: reading the webhook: %w", err)
	}
	return VerifyEvent(raw, r.Header.Get(HeaderSignature), r.Header.Get(HeaderTimestamp), secret)
}

// VerifyEvent verifies a body you already read. `now` is time.Now().
func VerifyEvent(body []byte, signature, timestamp, secret string) (Event, error) {
	return verifyEventAt(body, signature, timestamp, secret, time.Now())
}

func verifyEventAt(body []byte, signature, timestamp, secret string, now time.Time) (Event, error) {
	seconds, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return Event{}, fmt.Errorf("%w: invalid timestamp", ErrInvalidSignature)
	}
	when := time.Unix(seconds, 0)
	// The clock window. Without it a legitimate event captured once works forever.
	if d := now.Sub(when); d > ClockTolerance || d < -ClockTolerance {
		return Event{}, fmt.Errorf("%w: outside the %s window", ErrInvalidSignature, ClockTolerance)
	}
	expected := sign(body, seconds, secret)
	// THE HEADER MAY CARRY SEVERAL SIGNATURES, separated by spaces: during a
	// secret rotation the event is signed with the new secret AND the previous
	// one, so you can update your copy whenever you want without losing an
	// event. One match is enough. The current secret's signature goes first.
	for _, candidate := range strings.Fields(signature) {
		if hmac.Equal([]byte(expected), []byte(candidate)) {
			var ev Event
			if err := json.Unmarshal(body, &ev); err != nil {
				return Event{}, fmt.Errorf("payments: the event is not readable: %w", err)
			}
			return ev, nil
		}
	}
	return Event{}, ErrInvalidSignature
}

// sign computes "v1=<hex>" over timestamp + "." + body.
func sign(body []byte, seconds int64, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(seconds, 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return SignatureVersion + "=" + hex.EncodeToString(mac.Sum(nil))
}
