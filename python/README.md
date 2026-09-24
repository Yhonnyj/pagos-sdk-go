# TuCapi — Python SDK (payments API v2)

The official TuCapi client for the payments API, version 2: payins, payouts,
events and **webhook signature verification**. No dependencies; Python 3.10+.

```
pip install tucapi
```

```python
from tucapi import TuCapi, parse_event

c = TuCapi("tuc_live_...")

created = c.payouts.create(
    {"country": "VE", "currency": "VES", "method": "mobile_payment", "amount": "1500.50",
     "beneficiary": {"name": "Ana Pérez", "document": {"type": "V", "number": "12345678"},
                     "bank_code": "0102", "account_number": "04121234567"}},
    idempotency_key="order-4821",
)
# created["idempotency_key"] is the key that was sent. Without it the SDK
# generates one, reuses it on every retry and returns it here.

tx = c.transactions.wait_until_final(created["id"])
```

## What is generated and what is not

`tucapi/contract_gen.py` (TypedDicts, Literals, routes, scopes, headers,
the operations table) is generated from `api/openapi-v2.json` and must not
be edited. `tucapi/client.py` and `tucapi/webhooks.py` are written by
hand: transport, errors, idempotency, retries and signature verification.

## Retries

Only when nothing can move money twice: a `GET` on a network error, a `429`
or a `5xx`; a `POST` that carries an `Idempotency-Key` (creates) with the
**same key**. `confirm`, `resend_code`, `cancel` and
`webhook_endpoints.create` are never retried automatically. Three attempts by
default (`max_attempts`).

## Webhooks

```python
@app.post("/webhook")
def webhook():
    try:
        ev = parse_event(request.get_data(), request.headers, secret)  # RAW body
    except SignatureError:
        return "", 400  # discard it
    # ev["type"]: "payout.confirmed" | "payout.failed" | "payin.confirmed" | "payin.failed"
    return "", 200
```

Constant-time comparison, a 5-minute clock window, and two signatures
accepted during a secret rotation.

## Tests

```
python -m pytest -q
```

## Documentation and license

Documentation: <https://api.tucapi.app/v2/docs>. Proprietary: use is
permitted to TuCapi customers under the terms of their service agreement.
