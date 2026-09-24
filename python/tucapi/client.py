"""TuCapi: HTTP client of the payments API, version 2.

Hand-written over ``urllib``: the contract (types, enums, routes) is generated
by ``scripts/generar_sdk_v2.go`` from ``api/openapi-v2.json``; the transport,
the retry policy and the idempotency are not mechanical and live here.

WHAT IT DOES FOR YOU:

1. Authentication with your API key.
2. The contract, typed.
3. Idempotency and retries that are SAFE: every create carries an
   ``Idempotency-Key`` (yours, or one the SDK generates and hands back), and
   the SDK only retries a request that cannot move money twice.

Example::

    c = TuCapi("tuc_live_...")
    created = c.payouts.create({"country": "VE", "currency": "VES", "method": "mobile_payment",
                                "amount": "1500.50", "beneficiary": {...}}, idempotency_key="order-4821")
    created["idempotency_key"]   # the key that was sent: keep it with your order
    tx = c.transactions.wait_until_final(created["id"])
"""

from __future__ import annotations

import json
import random
import secrets
import ssl
import time
import urllib.error
import urllib.parse
import urllib.request
from typing import Any, Callable

from .contract_gen import (
    DEFAULT_BASE_URL,
    HEADER_IDEMPOTENCY_KEY,
    OPERATIONS,
    ROUTE_BALANCES_LIST,
    ROUTE_BANKS_LIST,
    ROUTE_CAPABILITIES_GET,
    ROUTE_EVENTS_LIST,
    ROUTE_METHODS_LIST,
    ROUTE_PAYINS_CONFIRM,
    ROUTE_PAYINS_CREATE,
    ROUTE_PAYINS_GET,
    ROUTE_PAYINS_RESEND_CODE,
    ROUTE_PAYOUTS_CANCEL,
    ROUTE_PAYOUTS_CREATE,
    ROUTE_PAYOUTS_GET,
    ROUTE_TRANSACTIONS_GET,
    ROUTE_WEBHOOK_ENDPOINTS_CREATE,
    BalanceList,
    BankList,
    Capabilities,
    Direction,
    EventList,
    FieldError,
    MethodCode,
    MethodList,
    NewPayin,
    NewPayout,
    Transaction,
    WebhookEndpoint,
)

#: The version of this SDK, sent in User-Agent.
VERSION = "0.1.0"

#: How long a single request waits. Creating a payment talks to a bank.
DEFAULT_TIMEOUT_SECONDS = 30.0

#: How many times a retryable request is tried in total.
DEFAULT_MAX_ATTEMPTS = 3


class PaymentsError(Exception):
    """An error answered by the API, with its stable code.

    Compare ``code``, never ``message``: the code is contract, the message may
    improve over time.
    """

    def __init__(self, status: int, code: str, message: str, details: list[FieldError] | None = None) -> None:
        self.status = status
        self.code = code
        self.message = message
        self.details = details or []
        if self.details:
            super().__init__(f"{message} ({code}): " + "; ".join(f"{d['field']}: {d['code']}" for d in self.details))
        else:
            super().__init__(f"{message} ({code})")


class TransportError(Exception):
    """No HTTP response was obtained at all, after every attempt.

    If this comes from a create, THE OPERATION MAY EXIST ANYWAY. Do not send
    it again with a new key: repeat the SAME request with ``idempotency_key``.
    """

    def __init__(self, cause: BaseException, attempts: int, idempotency_key: str | None = None) -> None:
        self.cause = cause
        self.attempts = attempts
        self.idempotency_key = idempotency_key
        super().__init__(f"payments: no response after {attempts} attempt(s): {cause}")


def new_idempotency_key() -> str:
    """The default key generator: 32 hex characters."""
    return secrets.token_hex(16)


def _default_backoff(attempt: int) -> float:
    """250 ms, 1 s, 4 s... with ±25 % jitter, capped at 10 s."""
    s = min(0.25 * 4 ** (attempt - 1), 10.0)
    return s + (random.random() - 0.5) * s * 0.5


def _with_id(route: str, id_: str) -> str:
    return route.replace("{id}", urllib.parse.quote(id_, safe=""))


class TuCapi:
    """The client.

    Args:
        api_key: your API key. It determines your company.
        base_url: the server. Production by default.
        timeout: seconds to wait for a response.
        max_attempts: how many times a retryable request is tried (1 = never retry).
    """

    def __init__(
        self,
        api_key: str,
        *,
        base_url: str = DEFAULT_BASE_URL,
        timeout: float = DEFAULT_TIMEOUT_SECONDS,
        max_attempts: int = DEFAULT_MAX_ATTEMPTS,
        ssl_context: ssl.SSLContext | None = None,
        sleep: Callable[[float], None] = time.sleep,
        new_key: Callable[[], str] = new_idempotency_key,
    ) -> None:
        if not api_key or not api_key.strip():
            raise ValueError("payments: the API key cannot be empty")
        self._api_key = api_key.strip()
        self._base_url = base_url.rstrip("/")
        self._timeout = timeout
        self._max_attempts = max(1, max_attempts)
        self._sleep = sleep
        self._new_key = new_key
        if ssl_context is None:
            ssl_context = ssl.create_default_context()
            ssl_context.minimum_version = ssl.TLSVersion.TLSv1_2
        self._ssl = ssl_context
        self.capabilities = CapabilitiesService(self)
        self.methods = MethodsService(self)
        self.banks = BanksService(self)
        self.balances = BalancesService(self)
        self.payins = PayinsService(self)
        self.payouts = PayoutsService(self)
        self.transactions = TransactionsService(self)
        self.events = EventsService(self)
        self.webhook_endpoints = WebhookEndpointsService(self)

    # ── transport ───────────────────────────────────────────────────────────

    def _create(self, op: str, route: str, body: dict[str, Any], idempotency_key: str | None) -> dict[str, Any]:
        key = (idempotency_key or "").strip() or self._new_key()
        out = self._call(op, route, body=body, idempotency_key=key)
        out["idempotency_key"] = key
        return out

    def _call(
        self,
        op: str,
        route: str,
        *,
        query: dict[str, str | None] | None = None,
        body: dict[str, Any] | None = None,
        idempotency_key: str | None = None,
    ) -> dict[str, Any]:
        """Performs one operation with the retry policy of the contract.

        A GET is retried on a network error, a 429 or a 5xx; a POST with
        Idempotency-Key is retried the same way WITH THE SAME KEY; any other
        POST (confirm, resend-code, cancel, webhook-endpoints) is never
        retried automatically: a lost answer means "check", not "again".
        """
        method, _route, _scope, idempotent = OPERATIONS[op]
        if idempotent and not idempotency_key:
            raise ValueError("payments: this operation needs an Idempotency-Key")
        retryable = method == "GET" or (idempotent and bool(idempotency_key))
        attempts = self._max_attempts if retryable else 1

        url = self._base_url + route
        if query:
            q = urllib.parse.urlencode({k: v for k, v in query.items() if v})
            if q:
                url += "?" + q
        data = None if body is None else json.dumps(body, ensure_ascii=False).encode("utf-8")

        last: BaseException | None = None
        for attempt in range(1, attempts + 1):
            if attempt > 1:
                self._sleep(_default_backoff(attempt - 1))
            req = urllib.request.Request(url, data=data, method=method)
            req.add_header("Authorization", f"Bearer {self._api_key}")
            req.add_header("Accept", "application/json")
            req.add_header("User-Agent", f"payments-sdk-python/{VERSION}")
            if data is not None:
                req.add_header("Content-Type", "application/json")
            if idempotency_key:
                req.add_header(HEADER_IDEMPOTENCY_KEY, idempotency_key)
            try:
                with urllib.request.urlopen(req, timeout=self._timeout, context=self._ssl) as res:
                    raw = res.read()
                    status = res.status
            except urllib.error.HTTPError as err:
                raw = err.read()
                status = err.code
            except (urllib.error.URLError, TimeoutError, OSError) as err:
                last = err
                continue

            if status == 429 or status >= 500:
                api_error = _decode_error(status, raw)
                if retryable and attempt < attempts:
                    last = api_error
                    continue
                raise api_error
            if status >= 400:
                raise _decode_error(status, raw)
            if not raw:
                return {}
            try:
                return json.loads(raw.decode("utf-8"))
            except ValueError as err:
                raise PaymentsError(status, "unreadable_response", "The response is not JSON") from err

        if isinstance(last, PaymentsError):
            raise last
        raise TransportError(last if last is not None else RuntimeError("no attempt"), attempts, idempotency_key)


def _decode_error(status: int, raw: bytes) -> PaymentsError:
    try:
        env = json.loads(raw.decode("utf-8"))
        e = env["error"]
        return PaymentsError(status, e["code"], e.get("message", f"HTTP {status}"), e.get("details", []))
    except (ValueError, KeyError, TypeError, AttributeError):
        return PaymentsError(status, "unreadable_response", f"HTTP {status}")


# ── the services ────────────────────────────────────────────────────────────


class CapabilitiesService:
    """What your key can do today."""

    def __init__(self, c: TuCapi) -> None:
        self._c = c

    def get(self) -> Capabilities:
        return self._c._call("capabilities.get", ROUTE_CAPABILITIES_GET)  # type: ignore[return-value]


class MethodsService:
    """The catalogue of methods."""

    def __init__(self, c: TuCapi) -> None:
        self._c = c

    def list(self, *, country: str | None = None, direction: Direction | None = None) -> MethodList:
        return self._c._call("methods.list", ROUTE_METHODS_LIST, query={"country": country, "direction": direction})  # type: ignore[return-value]


class BanksService:
    """The banks of a country."""

    def __init__(self, c: TuCapi) -> None:
        self._c = c

    def list(self, country: str, method: MethodCode | None = None) -> BankList:
        return self._c._call("banks.list", ROUTE_BANKS_LIST, query={"country": country, "method": method})  # type: ignore[return-value]


class BalancesService:
    """Your balances by currency."""

    def __init__(self, c: TuCapi) -> None:
        self._c = c

    def list(self) -> BalanceList:
        return self._c._call("balances.list", ROUTE_BALANCES_LIST)  # type: ignore[return-value]


class PayinsService:
    """Charging a person."""

    def __init__(self, c: TuCapi) -> None:
        self._c = c

    def create(self, payin: NewPayin, *, idempotency_key: str | None = None) -> dict[str, Any]:
        """Creates a payin. Safe to retry: the same key returns the SAME
        operation and asks the payer for no new code. The result carries
        ``idempotency_key``: keep it with your order."""
        return self._c._create("payins.create", ROUTE_PAYINS_CREATE, dict(payin), idempotency_key)

    def get(self, id_: str) -> Transaction:
        return self._c._call("payins.get", _with_id(ROUTE_PAYINS_GET, id_))  # type: ignore[return-value]

    def confirm(self, id_: str, code: str) -> Transaction:
        """Executes the debit with the code the payer typed.

        THE CODE HAS ONE TRY: a wrong one fails the operation. Never retried
        automatically: a lost answer means ``get``, not ``confirm`` again.
        """
        return self._c._call("payins.confirm", _with_id(ROUTE_PAYINS_CONFIRM, id_), body={"code": code})  # type: ignore[return-value]

    def resend_code(self, id_: str) -> Transaction:
        """Asks the payer's bank for a NEW code; the previous one stops working."""
        return self._c._call("payins.resendCode", _with_id(ROUTE_PAYINS_RESEND_CODE, id_), body={})  # type: ignore[return-value]


class PayoutsService:
    """Paying a person."""

    def __init__(self, c: TuCapi) -> None:
        self._c = c

    def create(self, payout: NewPayout, *, idempotency_key: str | None = None) -> dict[str, Any]:
        """Creates and sends a payout. Safe to retry with the same key."""
        return self._c._create("payouts.create", ROUTE_PAYOUTS_CREATE, dict(payout), idempotency_key)

    def get(self, id_: str) -> Transaction:
        return self._c._call("payouts.get", _with_id(ROUTE_PAYOUTS_GET, id_))  # type: ignore[return-value]

    def cancel(self, id_: str) -> Transaction:
        """Cancels a payout still pending. ``not_cancellable`` if it no longer
        can be; ``temporarily_unavailable`` if the provider did not answer
        clearly — the payout is still pending then, and this is NOT retried:
        check with ``get``."""
        return self._c._call("payouts.cancel", _with_id(ROUTE_PAYOUTS_CANCEL, id_), body={})  # type: ignore[return-value]


class TransactionsService:
    """Any operation by id, payin or payout."""

    def __init__(self, c: TuCapi) -> None:
        self._c = c

    def get(self, id_: str) -> Transaction:
        return self._c._call("transactions.get", _with_id(ROUTE_TRANSACTIONS_GET, id_))  # type: ignore[return-value]

    def wait_until_final(self, id_: str, every: float = 2.0, max_polls: int = 150) -> Transaction:
        """Polls ``get`` until confirmed or failed. Webhooks are the right way; this is for scripts."""
        tx = self.get(id_)
        polls = 1
        while tx["status"] == "pending" and polls < max_polls:
            self._c._sleep(every)
            tx = self.get(id_)
            polls += 1
        return tx


class EventsService:
    """What happened to your operations, by query."""

    def __init__(self, c: TuCapi) -> None:
        self._c = c

    def list(self, cursor: str | None = None, limit: int | None = None) -> EventList:
        """Events in order. ``cursor`` = ``next_cursor`` of the previous page."""
        return self._c._call("events.list", ROUTE_EVENTS_LIST, query={"cursor": cursor, "limit": str(limit) if limit else None})  # type: ignore[return-value]


class WebhookEndpointsService:
    """Where events are delivered."""

    def __init__(self, c: TuCapi) -> None:
        self._c = c

    def create(self, url: str) -> WebhookEndpoint:
        """Registers the https url. KEEP THE SECRET: it is shown once."""
        return self._c._call("webhookEndpoints.create", ROUTE_WEBHOOK_ENDPOINTS_CREATE, body={"url": url})  # type: ignore[return-value]
