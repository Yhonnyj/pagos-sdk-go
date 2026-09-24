// Package payments is the Go client for the payments API, version 2.
//
// ═══════════════════════════════════════════════════════════════════════════
// WHAT IT DOES FOR YOU, AND WHY IT LIVES HERE AND NOT IN YOUR CODE.
//
//  1. Authentication with your API key.
//  2. The contract, typed: you never guess a field name (contract_gen.go is
//     generated from the OpenAPI document).
//  3. Idempotency and retries that are SAFE: every create carries an
//     Idempotency-Key (yours, or one the SDK generates and hands back), and
//     the SDK only ever retries a request that cannot move money twice.
//  4. WEBHOOK SIGNATURE VERIFICATION (webhooks.go): constant time, with the
//     clock window, over the raw body.
//
// ═══════════════════════════════════════════════════════════════════════════
// EXAMPLE
//
//	c := tucapi.New("tuc_live_...")
//
//	created, err := c.Payouts.Create(ctx, tucapi.NewPayout{
//		Country: "VE", Currency: "VES", Method: tucapi.MethodCodeMobilePayment, Amount: "1500.50",
//		Beneficiary: tucapi.Beneficiary{Name: "Ana Pérez", Document: &tucapi.Document{Type: "V", Number: "12345678"},
//			BankCode: "0102", AccountNumber: "04121234567"},
//	}, tucapi.WithIdempotencyKey("order-4821"))
//	// created.IdempotencyKey is the key that was sent: keep it with your order.
//
//	tx, err := c.Transactions.WaitUntilFinal(ctx, created.ID, 2*time.Second)
//	switch tx.Status {
//	case tucapi.StatusConfirmed: // settle
//	case tucapi.StatusFailed:    // look at tx.Failure.Code
//	}
//
// And in your webhook receiver:
//
//	ev, err := tucapi.ParseEvent(r, mySecret)
//	if err != nil { http.Error(w, "invalid signature", 400); return }
package tucapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	mathrand "math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Version is the version of this SDK, sent in User-Agent.
const Version = "0.1.0"

// DefaultTimeout is how long a single request waits. Creating a payment talks
// to a bank, and banks can be slow; cutting earlier would abandon a request
// that is still alive on the other side.
const DefaultTimeout = 30 * time.Second

// DefaultMaxAttempts is how many times a retryable request is tried in total.
const DefaultMaxAttempts = 3

// Error is an error answered by the API, with its stable code.
//
// Compare Code, never Message: the message may improve over time.
type Error struct {
	StatusCode int
	Code       ErrorCode
	Message    string
	Details    []FieldError
}

func (e *Error) Error() string {
	if len(e.Details) > 0 {
		var partes []string
		for _, d := range e.Details {
			partes = append(partes, d.Field+": "+string(d.Code))
		}
		return fmt.Sprintf("payments: %s (%s): %s", e.Message, e.Code, strings.Join(partes, "; "))
	}
	return fmt.Sprintf("payments: %s (%s)", e.Message, e.Code)
}

// IsCode reports whether err is an API error with that code.
func IsCode(err error, code ErrorCode) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == code
}

// TransportError is returned when no HTTP response was obtained at all.
//
// ⚠️ IF THIS COMES BACK FROM A CREATE, THE OPERATION MAY EXIST ANYWAY. Do not
// send it again with a new key: repeat the SAME request with the SAME
// Idempotency-Key (the one in the error) and you will get the operation that
// already exists. The SDK already did that DefaultMaxAttempts times.
type TransportError struct {
	Err            error
	Attempts       int
	IdempotencyKey string
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("payments: no response after %d attempt(s): %v", e.Attempts, e.Err)
}

func (e *TransportError) Unwrap() error { return e.Err }

// Created is a transaction just created, together with the Idempotency-Key
// that was sent: keep it with your order, it is what makes a retry safe.
type Created struct {
	Transaction
	IdempotencyKey string
}

// Client talks to the API.
type Client struct {
	apiKey      string
	baseURL     string
	http        *http.Client
	maxAttempts int
	backoff     func(attempt int) time.Duration
	sleep       func(context.Context, time.Duration) error
	newKey      func() string

	Capabilities     *CapabilitiesService
	Methods          *MethodsService
	Banks            *BanksService
	Balances         *BalancesService
	Payins           *PayinsService
	Payouts          *PayoutsService
	Transactions     *TransactionsService
	Events           *EventsService
	WebhookEndpoints *WebhookEndpointsService
}

// Option configures the client.
type Option func(*Client)

// WithBaseURL points the client at another server (tests, your own environment).
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(strings.TrimSpace(u), "/") }
}

// WithHTTPClient replaces the HTTP client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// WithMaxAttempts sets how many times a retryable request is tried in total
// (1 = never retry).
func WithMaxAttempts(n int) Option {
	return func(c *Client) {
		if n >= 1 {
			c.maxAttempts = n
		}
	}
}

// WithIdempotencyKeys replaces how the SDK generates a key when you do not
// pass one. The default is 32 hex characters from crypto/rand.
func WithIdempotencyKeys(f func() string) Option {
	return func(c *Client) { c.newKey = f }
}

// New builds a client with your API key.
func New(apiKey string, opts ...Option) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// TLS 1.2 minimum: the key travels in every request.
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	c := &Client{
		apiKey:      strings.TrimSpace(apiKey),
		baseURL:     DefaultBaseURL,
		http:        &http.Client{Timeout: DefaultTimeout, Transport: transport},
		maxAttempts: DefaultMaxAttempts,
		backoff:     defaultBackoff,
		sleep:       sleepContext,
		newKey:      NewIdempotencyKey,
	}
	for _, o := range opts {
		o(c)
	}
	c.Capabilities = &CapabilitiesService{c}
	c.Methods = &MethodsService{c}
	c.Banks = &BanksService{c}
	c.Balances = &BalancesService{c}
	c.Payins = &PayinsService{c}
	c.Payouts = &PayoutsService{c}
	c.Transactions = &TransactionsService{c}
	c.Events = &EventsService{c}
	c.WebhookEndpoints = &WebhookEndpointsService{c}
	return c
}

// NewIdempotencyKey is the default key generator: 32 hex characters.
func NewIdempotencyKey() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("payments: crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}

// defaultBackoff: 250 ms, 1 s, 4 s… with ±25 % jitter, capped at 10 s.
func defaultBackoff(attempt int) time.Duration {
	d := 250 * time.Millisecond
	for i := 1; i < attempt; i++ {
		d *= 4
	}
	if d > 10*time.Second {
		d = 10 * time.Second
	}
	jitter := time.Duration(mathrand.Int63n(int64(d)/2+1)) - d/4
	return d + jitter
}

func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// RequestOption configures one call.
type RequestOption func(*requestOptions)

type requestOptions struct {
	idempotencyKey string
}

// WithIdempotencyKey sets the Idempotency-Key of a create. Use something
// derived from YOUR order (its id), so that a retry from a fresh process
// sends the same key. Without it, the SDK generates one and returns it.
func WithIdempotencyKey(k string) RequestOption {
	return func(o *requestOptions) { o.idempotencyKey = strings.TrimSpace(k) }
}

// ─── the services ────────────────────────────────────────────────────────────

// CapabilitiesService: what your key can do today.
type CapabilitiesService struct{ c *Client }

// Get answers countries, currencies, methods with their fields and limits, and config_version.
func (s *CapabilitiesService) Get(ctx context.Context) (Capabilities, error) {
	var out Capabilities
	err := s.c.call(ctx, "capabilities.get", RouteCapabilitiesGet, nil, nil, "", &out)
	return out, err
}

// MethodsService: the catalogue of methods.
type MethodsService struct{ c *Client }

// List filters by country (ISO 3166-1 alpha-2) and direction; empty = all.
func (s *MethodsService) List(ctx context.Context, country string, direction Direction) (MethodList, error) {
	q := url.Values{}
	if country != "" {
		q.Set("country", country)
	}
	if direction != "" {
		q.Set("direction", string(direction))
	}
	var out MethodList
	err := s.c.call(ctx, "methods.list", RouteMethodsList, q, nil, "", &out)
	return out, err
}

// BanksService: the banks of a country.
type BanksService struct{ c *Client }

// List answers the banks of a country, optionally only those admitting a method.
func (s *BanksService) List(ctx context.Context, country string, method MethodCode) (BankList, error) {
	q := url.Values{"country": {country}}
	if method != "" {
		q.Set("method", string(method))
	}
	var out BankList
	err := s.c.call(ctx, "banks.list", RouteBanksList, q, nil, "", &out)
	return out, err
}

// BalancesService: your balances by currency.
type BalancesService struct{ c *Client }

// List answers available and reserved per currency. Needs ScopeSaldosLeer.
func (s *BalancesService) List(ctx context.Context) (BalanceList, error) {
	var out BalanceList
	err := s.c.call(ctx, "balances.list", RouteBalancesList, nil, nil, "", &out)
	return out, err
}

// PayinsService: charging a person.
type PayinsService struct{ c *Client }

// Create creates a payin. With MethodCodeDebitOTP the payer's bank sends them
// a code; confirm with Confirm. Safe to retry: the same Idempotency-Key
// returns the SAME operation and does not ask the payer for another code.
func (s *PayinsService) Create(ctx context.Context, in NewPayin, opts ...RequestOption) (Created, error) {
	return s.c.create(ctx, "payins.create", RoutePayinsCreate, in, opts)
}

// Get answers the current state. It does not ask the bank anything.
func (s *PayinsService) Get(ctx context.Context, id string) (Transaction, error) {
	var out Transaction
	err := s.c.call(ctx, "payins.get", conID(RoutePayinsGet, id), nil, nil, "", &out)
	return out, err
}

// Confirm executes the debit with the code the payer typed.
//
// ⚠️ THE CODE HAS ONE TRY: a wrong one fails the operation (FailureCodeCodeRejected);
// ask for another with ResendCode. The code is never stored.
// This is NOT retried automatically: a lost response means Get, not Confirm again.
func (s *PayinsService) Confirm(ctx context.Context, id, code string) (Transaction, error) {
	var out Transaction
	err := s.c.call(ctx, "payins.confirm", conID(RoutePayinsConfirm, id), nil, Confirmation{Code: code}, "", &out)
	return out, err
}

// ResendCode asks the payer's bank for a NEW code; the previous one stops working.
func (s *PayinsService) ResendCode(ctx context.Context, id string) (Transaction, error) {
	var out Transaction
	err := s.c.call(ctx, "payins.resendCode", conID(RoutePayinsResendCode, id), nil, nil, "", &out)
	return out, err
}

// PayoutsService: paying a person.
type PayoutsService struct{ c *Client }

// Create creates and sends a payout. Safe to retry with the same Idempotency-Key.
func (s *PayoutsService) Create(ctx context.Context, in NewPayout, opts ...RequestOption) (Created, error) {
	return s.c.create(ctx, "payouts.create", RoutePayoutsCreate, in, opts)
}

// Get answers the current state.
func (s *PayoutsService) Get(ctx context.Context, id string) (Transaction, error) {
	var out Transaction
	err := s.c.call(ctx, "payouts.get", conID(RoutePayoutsGet, id), nil, nil, "", &out)
	return out, err
}

// Cancel cancels a payout that is still pending. ErrorCodeNotCancellable if it
// no longer can be; ErrorCodeTemporarilyUnavailable if the provider did not
// answer clearly — the payout is still pending then, and this is NOT retried
// automatically: check with Get.
func (s *PayoutsService) Cancel(ctx context.Context, id string) (Transaction, error) {
	var out Transaction
	err := s.c.call(ctx, "payouts.cancel", conID(RoutePayoutsCancel, id), nil, nil, "", &out)
	return out, err
}

// TransactionsService: any operation by id, payin or payout.
type TransactionsService struct{ c *Client }

// Get answers any operation, payin or payout, with the same shape.
func (s *TransactionsService) Get(ctx context.Context, id string) (Transaction, error) {
	var out Transaction
	err := s.c.call(ctx, "transactions.get", conID(RouteTransactionsGet, id), nil, nil, "", &out)
	return out, err
}

// WaitUntilFinal polls Get every `every` until the status is confirmed or
// failed, or ctx ends. Webhooks are the right way to learn the outcome; this
// is for scripts and for reconciling.
func (s *TransactionsService) WaitUntilFinal(ctx context.Context, id string, every time.Duration) (Transaction, error) {
	if every <= 0 {
		every = 2 * time.Second
	}
	for {
		tx, err := s.Get(ctx, id)
		if err != nil {
			return tx, err
		}
		if tx.Status != StatusPending {
			return tx, nil
		}
		if err := s.c.sleep(ctx, every); err != nil {
			return tx, err
		}
	}
}

// EventsService: what happened to your operations, by query.
type EventsService struct{ c *Client }

// List answers events in order. cursor = NextCursor of the previous page ("" = from the start); limit 1..200 (0 = default).
func (s *EventsService) List(ctx context.Context, cursor string, limit int) (EventList, error) {
	q := url.Values{}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var out EventList
	err := s.c.call(ctx, "events.list", RouteEventsList, q, nil, "", &out)
	return out, err
}

// WebhookEndpointsService: where events are delivered.
type WebhookEndpointsService struct{ c *Client }

// Create registers the https url. KEEP THE SECRET: it is shown once.
func (s *WebhookEndpointsService) Create(ctx context.Context, u string) (WebhookEndpoint, error) {
	var out WebhookEndpoint
	err := s.c.call(ctx, "webhookEndpoints.create", RouteWebhookEndpointsCreate, nil, NewWebhookEndpoint{URL: u}, "", &out)
	return out, err
}

// ─── transport ───────────────────────────────────────────────────────────────

func conID(route, id string) string {
	return strings.Replace(route, "{id}", url.PathEscape(id), 1)
}

func (c *Client) create(ctx context.Context, op, route string, in any, opts []RequestOption) (Created, error) {
	var o requestOptions
	for _, f := range opts {
		f(&o)
	}
	if o.idempotencyKey == "" {
		o.idempotencyKey = c.newKey()
	}
	var out Created
	err := c.call(ctx, op, route, nil, in, o.idempotencyKey, &out.Transaction)
	out.IdempotencyKey = o.idempotencyKey
	return out, err
}

// call performs one operation, with the retry policy of the contract:
//
//   - a GET is retried on a network error, a 429 or a 5xx (nothing moves);
//   - a POST with Idempotency-Key is retried the same way, WITH THE SAME KEY
//     (that is what makes it safe);
//   - any other POST (confirm, resend-code, cancel, webhook-endpoints) is
//     never retried automatically: a lost answer means "check", not "again".
func (c *Client) call(ctx context.Context, op, route string, query url.Values, body any, idempotencyKey string, out any) error {
	spec, ok := operations[op]
	if !ok {
		return fmt.Errorf("payments: unknown operation %q", op)
	}
	if spec.Idempotent && idempotencyKey == "" {
		return errors.New("payments: this operation needs an Idempotency-Key")
	}
	retryable := spec.Method == http.MethodGet || (spec.Idempotent && idempotencyKey != "")
	attempts := c.maxAttempts
	if !retryable {
		attempts = 1
	}

	var raw []byte
	if body != nil {
		var err error
		if raw, err = json.Marshal(body); err != nil {
			return fmt.Errorf("payments: encoding the body: %w", err)
		}
	}
	target := c.baseURL + route
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if attempt > 1 {
			if err := c.sleep(ctx, c.backoff(attempt-1)); err != nil {
				return err
			}
		}
		var reader io.Reader
		if raw != nil {
			reader = bytes.NewReader(raw)
		}
		req, err := http.NewRequestWithContext(ctx, spec.Method, target, reader)
		if err != nil {
			return fmt.Errorf("payments: building the request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "payments-sdk-go/"+Version)
		if raw != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if idempotencyKey != "" {
			req.Header.Set(HeaderIdempotencyKey, idempotencyKey)
		}

		res, err := c.http.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			lastErr = err
			continue
		}
		crudo, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reading the response: %w", err)
			continue
		}
		if res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500 {
			apiErr := decodeError(res.StatusCode, crudo)
			if retryable && attempt < attempts {
				lastErr = apiErr
				continue
			}
			return apiErr
		}
		if res.StatusCode >= 400 {
			return decodeError(res.StatusCode, crudo)
		}
		if out == nil {
			return nil
		}
		if err := json.Unmarshal(crudo, out); err != nil {
			return fmt.Errorf("payments: unreadable response: %w", err)
		}
		return nil
	}
	var apiErr *Error
	if errors.As(lastErr, &apiErr) {
		return apiErr
	}
	return &TransportError{Err: lastErr, Attempts: attempts, IdempotencyKey: idempotencyKey}
}

func decodeError(status int, crudo []byte) *Error {
	var env ErrorEnvelope
	if err := json.Unmarshal(crudo, &env); err != nil || env.Error.Code == "" {
		return &Error{StatusCode: status, Code: "unreadable_response", Message: "HTTP " + strconv.Itoa(status)}
	}
	return &Error{StatusCode: status, Code: env.Error.Code, Message: env.Error.Message, Details: env.Error.Details}
}
