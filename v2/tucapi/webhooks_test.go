package tucapi

import (
	"errors"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// The same vectors as the Node and Python SDKs: change one, change all three.
const (
	secret    = "whsec_test_0123456789"
	eventBody = `{"id":"0c9d1b2e-7f3a-4b8c-9d0e-1f2a3b4c5d6e","type":"payout.confirmed","occurred_at":"2026-09-23T15:04:05Z","data":{"id":"6f1c2a9e-3b4d-4c5e-8f70-1a2b3c4d5e6f","type":"payout","country":"VE","currency":"VES","method":"mobile_payment","amount":"1500.50","status":"confirmed","bank_reference":"000123456789","failure":null,"created_at":"2026-09-23T14:59:05Z","updated_at":"2026-09-23T15:04:05Z"}}`
	fixedTS   = int64(1790000000)
	// HMAC-SHA256(secret, "1790000000." + eventBody), precomputed once and
	// shared by the three SDKs.
	fixedSig = "v1=" + fixedSigHex
)

func TestTheSharedVectorVerifies(t *testing.T) {
	now := time.Unix(fixedTS, 0).Add(time.Minute)
	ev, err := verifyEventAt([]byte(eventBody), fixedSig, strconv.FormatInt(fixedTS, 10), secret, now)
	if err != nil {
		t.Fatalf("the shared vector does not verify: %v (sig computed here: %s)", err, sign([]byte(eventBody), fixedTS, secret))
	}
	if ev.Type != EventTypePayoutConfirmed || ev.Data.Status != StatusConfirmed || ev.Data.BankReference == nil {
		t.Fatalf("event: %+v", ev)
	}
}

func TestParseEventReadsTheRequest(t *testing.T) {
	now := time.Now().Unix()
	r := httptest.NewRequest("POST", "/webhook", strings.NewReader(eventBody))
	r.Header.Set(HeaderSignature, sign([]byte(eventBody), now, secret))
	r.Header.Set(HeaderTimestamp, strconv.FormatInt(now, 10))
	ev, err := ParseEvent(r, secret)
	if err != nil || ev.ID != "0c9d1b2e-7f3a-4b8c-9d0e-1f2a3b4c5d6e" {
		t.Fatalf("%v %+v", err, ev)
	}
}

func TestATamperedBodyIsRejected(t *testing.T) {
	now := time.Now().Unix()
	tampered := strings.Replace(eventBody, "1500.50", "9500.50", 1)
	if _, err := VerifyEvent([]byte(tampered), sign([]byte(eventBody), now, secret), strconv.FormatInt(now, 10), secret); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("a tampered body verified: %v", err)
	}
}

func TestAnotherSecretIsRejected(t *testing.T) {
	now := time.Now().Unix()
	if _, err := VerifyEvent([]byte(eventBody), sign([]byte(eventBody), now, "other"), strconv.FormatInt(now, 10), secret); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("another secret verified: %v", err)
	}
}

func TestAnOldEventIsRejectedEvenWithAGoodSignature(t *testing.T) {
	old := time.Now().Add(-ClockTolerance - time.Minute).Unix()
	if _, err := VerifyEvent([]byte(eventBody), sign([]byte(eventBody), old, secret), strconv.FormatInt(old, 10), secret); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("an old event verified: %v", err)
	}
	future := time.Now().Add(ClockTolerance + time.Minute).Unix()
	if _, err := VerifyEvent([]byte(eventBody), sign([]byte(eventBody), future, secret), strconv.FormatInt(future, 10), secret); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("a future event verified: %v", err)
	}
}

func TestDuringARotationEitherSignatureIsEnough(t *testing.T) {
	now := time.Now().Unix()
	ts := strconv.FormatInt(now, 10)
	header := sign([]byte(eventBody), now, "new-secret") + " " + sign([]byte(eventBody), now, secret)
	if _, err := VerifyEvent([]byte(eventBody), header, ts, secret); err != nil {
		t.Fatalf("the previous secret did not verify during rotation: %v", err)
	}
	if _, err := VerifyEvent([]byte(eventBody), header, ts, "new-secret"); err != nil {
		t.Fatalf("the new secret did not verify during rotation: %v", err)
	}
	if _, err := VerifyEvent([]byte(eventBody), header, ts, "neither"); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("an unrelated secret verified: %v", err)
	}
}

func TestMissingOrBrokenHeadersAreRejected(t *testing.T) {
	now := strconv.FormatInt(time.Now().Unix(), 10)
	for _, caso := range []struct{ sig, ts string }{{"", now}, {"v1=zz", now}, {sign([]byte(eventBody), time.Now().Unix(), secret), ""}, {sign([]byte(eventBody), time.Now().Unix(), secret), "ayer"}} {
		if _, err := VerifyEvent([]byte(eventBody), caso.sig, caso.ts, secret); !errors.Is(err, ErrInvalidSignature) {
			t.Fatalf("%+v verified: %v", caso, err)
		}
	}
}
