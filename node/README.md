# TuCapi — Node SDK (payments API v2)

The official TuCapi client for the payments API, version 2: payins, payouts,
events and **webhook signature verification**. No runtime dependencies;
Node 20+.

```
npm install tucapi
```

```ts
import { TuCapi, parseEvent } from 'tucapi'

const c = new TuCapi('ck_live_...')

const created = await c.payouts.create(
  { country: 'VE', currency: 'VES', method: 'mobile_payment', amount: '1500.50',
    beneficiary: { name: 'Ana Pérez', document: { type: 'V', number: '12345678' }, bank_code: '0102', account_number: '04121234567' } },
  { idempotencyKey: 'order-4821' },
)
// created.idempotencyKey is the key that was sent. Without it the SDK generates
// one, reuses it on every retry and returns it here.

const tx = await c.transactions.waitUntilFinal(created.id)
```

## What is generated and what is not

`src/contract.gen.ts` (types, enums, routes, scopes, headers, the operations
table) is generated from `api/openapi-v2.json` and must not be edited.
`src/index.ts` is written by hand: transport, errors, idempotency, retries and
signature verification.

## Retries

Only when nothing can move money twice: a `GET` on a network error, a `429`
or a `5xx`; a `POST` that carries an `Idempotency-Key` (creates) with the
**same key**. `confirm`, `resendCode`, `cancel` and `webhookEndpoints.create`
are never retried automatically. Three attempts by default (`maxAttempts`).

## Webhooks

```ts
app.post('/webhook', express.raw({ type: 'application/json' }), (req, res) => {
  try {
    const ev = parseEvent(req.body, req.headers, secret) // over the RAW body
    // ev.type: 'payout.confirmed' | 'payout.failed' | 'payin.confirmed' | 'payin.failed'
    res.sendStatus(200)
  } catch (err) {
    res.sendStatus(400) // SignatureError: discard it
  }
}
```

Use `express.raw`, not `express.json`: `JSON.parse` + `JSON.stringify` does
not give back the same bytes and the signature stops matching.

## Errors

`PaymentsError` (`code`, `status`, `details`), `TransportError` (no answer
at all; carries `idempotencyKey`), `SignatureError` (a webhook that does not
verify).

## Documentation and license

Documentation: <https://api.tucapi.app/v2/docs>. Proprietary: use is
permitted to TuCapi customers under the terms of their service agreement.
