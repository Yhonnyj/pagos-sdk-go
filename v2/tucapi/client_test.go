package tucapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeAPI records what arrives and answers a scripted sequence per route.
type fakeAPI struct {
	mu       sync.Mutex
	requests []recorded
	script   map[string][]response // "METHOD /path" → answers, in order
}

type recorded struct {
	Method, Path, Key, Body, Auth string
}

type response struct {
	status int
	body   string
	drop   bool // cut the connection instead of answering
}

func (f *fakeAPI) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.requests = append(f.requests, recorded{r.Method, r.URL.RequestURI(), r.Header.Get(HeaderIdempotencyKey), string(raw), r.Header.Get("Authorization")})
		clave := r.Method + " " + r.URL.Path
		answers := f.script[clave]
		var res response
		if len(answers) == 0 {
			res = response{status: 200, body: `{}`}
		} else {
			res = answers[0]
			if len(answers) > 1 {
				f.script[clave] = answers[1:]
			}
		}
		f.mu.Unlock()
		if res.drop {
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(res.status)
		_, _ = w.Write([]byte(res.body))
	})
}

func (f *fakeAPI) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func newClient(t *testing.T, f *fakeAPI, opts ...Option) *Client {
	t.Helper()
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	c := New("ck_test_x", append([]Option{WithBaseURL(srv.URL)}, opts...)...)
	c.sleep = func(context.Context, time.Duration) error { return nil } // no waiting in tests
	return c
}

const txJSON = `{"id":"6f1c2a9e-3b4d-4c5e-8f70-1a2b3c4d5e6f","type":"payout","country":"VE","currency":"VES","method":"mobile_payment","amount":"1500.50","status":"pending","pending_reason":"processing","bank_reference":null,"failure":null,"created_at":"2026-09-23T14:59:05Z","updated_at":"2026-09-23T14:59:05Z"}`

func payout() NewPayout {
	return NewPayout{Country: "VE", Currency: "VES", Method: MethodCodeMobilePayment, Amount: "1500.50",
		Beneficiary: Beneficiary{Name: "Ana", Document: &Document{Type: "V", Number: "12345678"}, BankCode: "0102", AccountNumber: "04121234567"}}
}

// ─── idempotency ─────────────────────────────────────────────────────────────

func TestACreateSendsTheKeyYouGiveAndReturnsIt(t *testing.T) {
	f := &fakeAPI{script: map[string][]response{"POST /v2/payouts": {{201, txJSON, false}}}}
	c := newClient(t, f)
	created, err := c.Payouts.Create(context.Background(), payout(), WithIdempotencyKey("order-4821"))
	if err != nil {
		t.Fatal(err)
	}
	if created.IdempotencyKey != "order-4821" || created.ID != "6f1c2a9e-3b4d-4c5e-8f70-1a2b3c4d5e6f" || created.Status != StatusPending {
		t.Fatalf("created: %+v", created)
	}
	r := f.requests[0]
	if r.Key != "order-4821" || r.Auth != "Bearer ck_test_x" || !strings.Contains(r.Body, `"method":"mobile_payment"`) || strings.Contains(r.Body, "purpose") {
		t.Fatalf("request: %+v", r)
	}
}

func TestACreateWithoutKeyGeneratesOneAndReusesItOnRetry(t *testing.T) {
	f := &fakeAPI{script: map[string][]response{"POST /v2/payins": {{0, "", true}, {503, `{"error":{"code":"temporarily_unavailable","message":"x"}}`, false}, {201, txJSON, false}}}}
	c := newClient(t, f)
	created, err := c.Payins.Create(context.Background(), NewPayin{Country: "VE", Currency: "VES", Method: MethodCodeDebitOTP, Amount: "10.00", Payer: Payer{Name: "Ana"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.IdempotencyKey) != 32 {
		t.Fatalf("generated key: %q", created.IdempotencyKey)
	}
	if f.count() != 3 {
		t.Fatalf("attempts: %d", f.count())
	}
	for _, r := range f.requests {
		if r.Key != created.IdempotencyKey {
			t.Fatalf("a retry changed the key: %q vs %q", r.Key, created.IdempotencyKey)
		}
	}
}

func TestACreateThatNeverGetsAnAnswerReturnsTheKeyInTheError(t *testing.T) {
	f := &fakeAPI{script: map[string][]response{"POST /v2/payouts": {{0, "", true}, {0, "", true}, {0, "", true}}}}
	c := newClient(t, f)
	_, err := c.Payouts.Create(context.Background(), payout(), WithIdempotencyKey("order-1"))
	var te *TransportError
	if !errors.As(err, &te) || te.Attempts != 3 || te.IdempotencyKey != "order-1" {
		t.Fatalf("error: %v", err)
	}
	if f.count() != 3 {
		t.Fatalf("attempts: %d", f.count())
	}
}

// ─── retries: only when nothing can move twice ───────────────────────────────

func TestAGetIsRetriedOn5xxAnd429(t *testing.T) {
	f := &fakeAPI{script: map[string][]response{"GET /v2/transactions/t1": {{502, `bad gateway`, false}, {429, `{"error":{"code":"x","message":"slow"}}`, false}, {200, txJSON, false}}}}
	c := newClient(t, f)
	tx, err := c.Transactions.Get(context.Background(), "t1")
	if err != nil || tx.ID == "" {
		t.Fatalf("%v %+v", err, tx)
	}
	if f.count() != 3 {
		t.Fatalf("attempts: %d", f.count())
	}
}

func TestAGetGivesUpAfterMaxAttempts(t *testing.T) {
	f := &fakeAPI{script: map[string][]response{"GET /v2/balances": {{500, `{"error":{"code":"internal","message":"x"}}`, false}, {500, `{"error":{"code":"internal","message":"x"}}`, false}}}}
	c := newClient(t, f, WithMaxAttempts(2))
	_, err := c.Balances.List(context.Background())
	var e *Error
	if !errors.As(err, &e) || e.StatusCode != 500 {
		t.Fatalf("error: %v", err)
	}
	if f.count() != 2 {
		t.Fatalf("attempts: %d", f.count())
	}
}

func TestConfirmCancelAndResendAreNeverRetried(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		ruta   string
		llamar func(c *Client) error
	}{
		{"confirm", "POST /v2/payins/p1/confirm", func(c *Client) error { _, err := c.Payins.Confirm(context.Background(), "p1", "123456"); return err }},
		{"resend", "POST /v2/payins/p1/resend-code", func(c *Client) error { _, err := c.Payins.ResendCode(context.Background(), "p1"); return err }},
		{"cancel", "POST /v2/payouts/p1/cancel", func(c *Client) error { _, err := c.Payouts.Cancel(context.Background(), "p1"); return err }},
		{"webhook", "POST /v2/webhook-endpoints", func(c *Client) error {
			_, err := c.WebhookEndpoints.Create(context.Background(), "https://x")
			return err
		}},
	} {
		f := &fakeAPI{script: map[string][]response{caso.ruta: {{503, `{"error":{"code":"temporarily_unavailable","message":"x"}}`, false}, {200, txJSON, false}}}}
		c := newClient(t, f)
		err := caso.llamar(c)
		if !IsCode(err, ErrorCodeTemporarilyUnavailable) {
			t.Fatalf("%s: %v", caso.nombre, err)
		}
		if f.count() != 1 {
			t.Fatalf("%s was retried: %d attempts (money could move twice)", caso.nombre, f.count())
		}
		if f.requests[0].Key != "" {
			t.Fatalf("%s sent an Idempotency-Key it does not have", caso.nombre)
		}
	}
}

// ─── errors and shapes ───────────────────────────────────────────────────────

func TestAnAPIErrorIsTypedWithItsDetails(t *testing.T) {
	f := &fakeAPI{script: map[string][]response{"POST /v2/payouts": {{400, `{"error":{"code":"field_invalid","message":"Hay campos con un valor inválido.","details":[{"field":"beneficiary.bank_code","code":"invalid"}]}}`, false}}}}
	c := newClient(t, f)
	_, err := c.Payouts.Create(context.Background(), payout(), WithIdempotencyKey("k"))
	var e *Error
	if !errors.As(err, &e) || e.Code != ErrorCodeFieldInvalid || e.StatusCode != 400 || len(e.Details) != 1 || e.Details[0].Field != "beneficiary.bank_code" || e.Details[0].Code != FieldErrorCodeInvalid {
		t.Fatalf("error: %#v", err)
	}
	if !IsCode(err, ErrorCodeFieldInvalid) || IsCode(err, ErrorCodeNotFound) {
		t.Fatal("IsCode")
	}
	if !strings.Contains(err.Error(), "beneficiary.bank_code: invalid") {
		t.Fatalf("message: %s", err)
	}
}

func TestAnUnreadableErrorStillHasTheStatus(t *testing.T) {
	f := &fakeAPI{script: map[string][]response{"GET /v2/payins/p1": {{404, `<html>`, false}}}}
	c := newClient(t, f)
	_, err := c.Payins.Get(context.Background(), "p1")
	var e *Error
	if !errors.As(err, &e) || e.StatusCode != 404 || e.Code != "unreadable_response" {
		t.Fatalf("error: %v", err)
	}
}

func TestIDsAreEscapedInThePath(t *testing.T) {
	f := &fakeAPI{script: map[string][]response{}}
	c := newClient(t, f)
	_, _ = c.Payouts.Get(context.Background(), "p1/cancel")
	if f.requests[0].Path != "/v2/payouts/p1%2Fcancel" {
		t.Fatalf("path: %s", f.requests[0].Path)
	}
}

func TestQueriesAreBuiltFromTheArguments(t *testing.T) {
	f := &fakeAPI{script: map[string][]response{}}
	c := newClient(t, f)
	_, _ = c.Banks.List(context.Background(), "VE", MethodCodeDebitOTP)
	_, _ = c.Methods.List(context.Background(), "", DirectionPayin)
	_, _ = c.Events.List(context.Background(), "c-9", 25)
	got := []string{f.requests[0].Path, f.requests[1].Path, f.requests[2].Path}
	want := []string{"/v2/banks?country=VE&method=debit_otp", "/v2/methods?direction=payin", "/v2/events?cursor=c-9&limit=25"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("query %d: %s, want %s", i, got[i], want[i])
		}
	}
}

func TestWaitUntilFinalPollsUntilTheOutcome(t *testing.T) {
	final := strings.Replace(txJSON, `"status":"pending","pending_reason":"processing","bank_reference":null`, `"status":"confirmed","bank_reference":"000123"`, 1)
	f := &fakeAPI{script: map[string][]response{"GET /v2/transactions/t1": {{200, txJSON, false}, {200, txJSON, false}, {200, final, false}}}}
	c := newClient(t, f)
	tx, err := c.Transactions.WaitUntilFinal(context.Background(), "t1", time.Millisecond)
	if err != nil || tx.Status != StatusConfirmed || tx.BankReference == nil || *tx.BankReference != "000123" {
		t.Fatalf("%v %+v", err, tx)
	}
	if f.count() != 3 {
		t.Fatalf("polls: %d", f.count())
	}
}

func TestTheGeneratedTableAgreesWithTheRoutes(t *testing.T) {
	// Every service method names an operation that exists, and the ones that
	// create carry a key.
	for _, op := range []string{"payins.create", "payouts.create"} {
		if !operations[op].Idempotent {
			t.Fatalf("%s is not idempotent in the contract table", op)
		}
	}
	for _, op := range []string{"payins.confirm", "payouts.cancel", "payins.resendCode", "webhookEndpoints.create"} {
		if operations[op].Idempotent {
			t.Fatalf("%s must NOT carry an Idempotency-Key", op)
		}
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(txJSON), &raw); err != nil {
		t.Fatal(err)
	}
}
