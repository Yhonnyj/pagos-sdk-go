# TuCapi — Go SDK (payments API v2)

The official TuCapi client for the payments API, version 2: payins, payouts,
events and **webhook signature verification**. No dependencies outside the
standard library.

```
go get github.com/tucapi/sdk-go/v2
```

```go
import "github.com/tucapi/sdk-go/v2/tucapi"

c := tucapi.New("ck_live_...")

created, err := c.Payouts.Create(ctx, tucapi.NewPayout{
	Country: "VE", Currency: "VES", Method: tucapi.MethodCodeMobilePayment, Amount: "1500.50",
	Beneficiary: tucapi.Beneficiary{Name: "Ana Pérez",
		Document: &tucapi.Document{Type: "V", Number: "12345678"},
		BankCode: "0102", AccountNumber: "04121234567"},
}, tucapi.WithIdempotencyKey("order-4821"))
// created.IdempotencyKey is the key that was sent. Without WithIdempotencyKey the
// SDK generates one, reuses it on every retry and returns it here.

tx, err := c.Transactions.WaitUntilFinal(ctx, created.ID, 2*time.Second)
```

## What is generated and what is not

`contract_gen.go` (types, enums, routes, scopes, headers, the operations
table) is generated from `api/openapi-v2.json` and must not be edited.
`client.go` and `webhooks.go` are written by hand: transport, errors,
idempotency, retries and signature verification.

## Retries

Only when nothing can move money twice: a `GET` on a network error, a
`429` or a `5xx`; a `POST` that carries an `Idempotency-Key` (creates) with
the **same key**. `Confirm`, `ResendCode`, `Cancel` and
`WebhookEndpoints.Create` are never retried automatically: a lost answer
means `Get`, not "again". Three attempts by default, exponential wait with
jitter (`WithMaxAttempts`).

## Webhooks

```go
ev, err := tucapi.ParseEvent(r, secret) // verifies X-Firma over the RAW body
if err != nil { http.Error(w, "invalid signature", 400); return }
switch ev.Type {
case tucapi.EventTypePayoutConfirmed: // settle ev.Data
case tucapi.EventTypePayoutFailed:    // look at ev.Data.Failure.Code
}
```

Constant-time comparison, a 5-minute clock window, and two signatures
accepted during a secret rotation.

## Errors

Every API error is a `*tucapi.Error` with `Code` (compare it with
`tucapi.IsCode(err, tucapi.ErrorCodeIdempotencyKeyReused)`), `Message`,
`StatusCode` and `Details` per field. A request that never got an answer is
a `*tucapi.TransportError` carrying the `IdempotencyKey` that was sent.

## Documentation and license

Documentation: <https://api.tucapi.app/v2/docs>. Proprietary: use is
permitted to TuCapi customers under the terms of their service agreement.
