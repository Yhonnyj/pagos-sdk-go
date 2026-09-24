"""Tests of the Python SDK: the same cases as Go and Node, same vectors."""

from __future__ import annotations

import io
import json
import sys
import time
import unittest
import urllib.error
from pathlib import Path
from unittest import mock

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from tucapi import (  # noqa: E402
    CLOCK_TOLERANCE_SECONDS,
    OPERATIONS,
    TuCapi,
    PaymentsError,
    SignatureError,
    TransportError,
    parse_event,
    sign,
    verify_event,
)

SECRET = "whsec_test_0123456789"
EVENT_BODY = (
    b'{"id":"0c9d1b2e-7f3a-4b8c-9d0e-1f2a3b4c5d6e","type":"payout.confirmed","occurred_at":"2026-09-23T15:04:05Z",'
    b'"data":{"id":"6f1c2a9e-3b4d-4c5e-8f70-1a2b3c4d5e6f","type":"payout","country":"VE","currency":"VES",'
    b'"method":"mobile_payment","amount":"1500.50","status":"confirmed","bank_reference":"000123456789","failure":null,'
    b'"created_at":"2026-09-23T14:59:05Z","updated_at":"2026-09-23T15:04:05Z"}}'
)
FIXED_TS = 1790000000
FIXED_SIG = "v1=f62b5544c9b42eb7d94c053ab1d6bda34febf84da6a205dc2df998a7d87d0382"

TX_JSON = (
    '{"id":"6f1c2a9e-3b4d-4c5e-8f70-1a2b3c4d5e6f","type":"payout","country":"VE","currency":"VES","method":"mobile_payment",'
    '"amount":"1500.50","status":"pending","pending_reason":"processing","bank_reference":null,"failure":null,'
    '"created_at":"2026-09-23T14:59:05Z","updated_at":"2026-09-23T15:04:05Z"}'
)

PAYOUT = {
    "country": "VE", "currency": "VES", "method": "mobile_payment", "amount": "1500.50",
    "beneficiary": {"name": "Ana", "document": {"type": "V", "number": "12345678"}, "bank_code": "0102", "account_number": "04121234567"},
}


class _Response(io.BytesIO):
    def __init__(self, status: int, body: str) -> None:
        super().__init__(body.encode("utf-8"))
        self.status = status

    def __enter__(self):
        return self

    def __exit__(self, *a):
        self.close()


class FakeAPI:
    """Scripted answers per 'METHOD /path'; records what arrived."""

    def __init__(self, script: dict[str, list]) -> None:
        self.script = {k: list(v) for k, v in script.items()}
        self.seen: list[dict] = []

    def urlopen(self, req, timeout=None, context=None):
        from urllib.parse import urlsplit

        path = urlsplit(req.full_url).path
        self.seen.append({"method": req.get_method(), "url": req.full_url, "key": req.get_header("Idempotency-key"), "body": req.data})
        answers = self.script.get(f"{req.get_method()} {path}", [])
        a = answers.pop(0) if len(answers) > 1 else (answers[0] if answers else (200, "{}"))
        if a == "drop":
            raise urllib.error.URLError("connection reset")
        status, body = a
        if status >= 400:
            raise urllib.error.HTTPError(req.full_url, status, "err", {}, io.BytesIO(body.encode("utf-8")))
        return _Response(status, body)


def client(fake: FakeAPI, **kw) -> TuCapi:
    return TuCapi("ck_test_x", base_url="https://api.test", sleep=lambda s: None, **kw)


def patched(fake: FakeAPI):
    return mock.patch("tucapi.client.urllib.request.urlopen", side_effect=fake.urlopen)


class Idempotency(unittest.TestCase):
    def test_a_create_sends_the_key_you_give_and_returns_it(self) -> None:
        f = FakeAPI({"POST /v2/payouts": [(201, TX_JSON)]})
        with patched(f):
            created = client(f).payouts.create(PAYOUT, idempotency_key="order-4821")
        self.assertEqual(created["idempotency_key"], "order-4821")
        self.assertEqual(created["status"], "pending")
        self.assertEqual(f.seen[0]["key"], "order-4821")
        self.assertIn(b'"method": "mobile_payment"', f.seen[0]["body"])

    def test_a_create_without_key_generates_one_and_reuses_it_on_retry(self) -> None:
        f = FakeAPI({"POST /v2/payins": ["drop", (503, '{"error":{"code":"temporarily_unavailable","message":"x"}}'), (201, TX_JSON)]})
        with patched(f):
            created = client(f).payins.create({"country": "VE", "currency": "VES", "method": "debit_otp", "amount": "10.00", "payer": {"name": "Ana"}})
        self.assertEqual(len(created["idempotency_key"]), 32)
        self.assertEqual(len(f.seen), 3)
        for s in f.seen:
            self.assertEqual(s["key"], created["idempotency_key"])

    def test_a_create_that_never_gets_an_answer_returns_the_key_in_the_error(self) -> None:
        f = FakeAPI({"POST /v2/payouts": ["drop", "drop", "drop"]})
        with patched(f), self.assertRaises(TransportError) as cm:
            client(f).payouts.create(PAYOUT, idempotency_key="order-1")
        self.assertEqual(cm.exception.attempts, 3)
        self.assertEqual(cm.exception.idempotency_key, "order-1")
        self.assertEqual(len(f.seen), 3)


class Retries(unittest.TestCase):
    def test_a_get_is_retried_on_5xx_and_429(self) -> None:
        f = FakeAPI({"GET /v2/transactions/t1": [(502, "bad gateway"), (429, '{"error":{"code":"x","message":"slow"}}'), (200, TX_JSON)]})
        with patched(f):
            tx = client(f).transactions.get("t1")
        self.assertEqual(tx["id"], "6f1c2a9e-3b4d-4c5e-8f70-1a2b3c4d5e6f")
        self.assertEqual(len(f.seen), 3)

    def test_a_get_gives_up_after_max_attempts(self) -> None:
        f = FakeAPI({"GET /v2/balances": [(500, '{"error":{"code":"internal","message":"x"}}'), (500, '{"error":{"code":"internal","message":"x"}}')]})
        with patched(f), self.assertRaises(PaymentsError) as cm:
            client(f, max_attempts=2).balances.list()
        self.assertEqual(cm.exception.status, 500)
        self.assertEqual(len(f.seen), 2)

    def test_confirm_resend_cancel_and_webhook_endpoints_are_never_retried(self) -> None:
        cases = [
            ("POST /v2/payins/p1/confirm", lambda c: c.payins.confirm("p1", "123456")),
            ("POST /v2/payins/p1/resend-code", lambda c: c.payins.resend_code("p1")),
            ("POST /v2/payouts/p1/cancel", lambda c: c.payouts.cancel("p1")),
            ("POST /v2/webhook-endpoints", lambda c: c.webhook_endpoints.create("https://x")),
        ]
        for route, call in cases:
            f = FakeAPI({route: [(503, '{"error":{"code":"temporarily_unavailable","message":"x"}}'), (200, TX_JSON)]})
            with patched(f), self.assertRaises(PaymentsError) as cm:
                call(client(f))
            self.assertEqual(cm.exception.code, "temporarily_unavailable", route)
            self.assertEqual(len(f.seen), 1, f"{route} was retried")
            self.assertIsNone(f.seen[0]["key"], f"{route} sent a key it does not have")


class Shapes(unittest.TestCase):
    def test_an_api_error_is_typed_with_its_details(self) -> None:
        f = FakeAPI({"POST /v2/payouts": [(400, '{"error":{"code":"field_invalid","message":"Hay campos con un valor inválido.","details":[{"field":"beneficiary.bank_code","code":"invalid"}]}}')]})
        with patched(f), self.assertRaises(PaymentsError) as cm:
            client(f).payouts.create(PAYOUT, idempotency_key="k")
        e = cm.exception
        self.assertEqual((e.code, e.status), ("field_invalid", 400))
        self.assertEqual(e.details, [{"field": "beneficiary.bank_code", "code": "invalid"}])
        self.assertIn("beneficiary.bank_code: invalid", str(e))

    def test_an_unreadable_error_still_has_the_status(self) -> None:
        f = FakeAPI({"GET /v2/payins/p1": [(404, "<html>")]})
        with patched(f), self.assertRaises(PaymentsError) as cm:
            client(f).payins.get("p1")
        self.assertEqual((cm.exception.status, cm.exception.code), (404, "unreadable_response"))

    def test_ids_are_escaped_and_queries_are_built(self) -> None:
        f = FakeAPI({})
        with patched(f):
            c = client(f)
            c.payouts.get("p1/cancel")
            c.banks.list("VE", "debit_otp")
            c.methods.list(direction="payin")
            c.events.list("c-9", 25)
        self.assertEqual(
            [s["url"] for s in f.seen],
            ["https://api.test/v2/payouts/p1%2Fcancel", "https://api.test/v2/banks?country=VE&method=debit_otp", "https://api.test/v2/methods?direction=payin", "https://api.test/v2/events?cursor=c-9&limit=25"],
        )

    def test_wait_until_final_polls_until_the_outcome(self) -> None:
        final = TX_JSON.replace('"status":"pending","pending_reason":"processing","bank_reference":null', '"status":"confirmed","bank_reference":"000123"')
        f = FakeAPI({"GET /v2/transactions/t1": [(200, TX_JSON), (200, TX_JSON), (200, final)]})
        with patched(f):
            tx = client(f).transactions.wait_until_final("t1", every=0)
        self.assertEqual((tx["status"], tx["bank_reference"]), ("confirmed", "000123"))
        self.assertEqual(len(f.seen), 3)

    def test_the_generated_table_says_which_operations_carry_a_key(self) -> None:
        self.assertTrue(OPERATIONS["payins.create"][3] and OPERATIONS["payouts.create"][3])
        for op in ("payins.confirm", "payouts.cancel", "payins.resendCode", "webhookEndpoints.create"):
            self.assertFalse(OPERATIONS[op][3], op)


class Webhooks(unittest.TestCase):
    def test_the_shared_vector_verifies(self) -> None:
        ev = verify_event(EVENT_BODY, FIXED_SIG, str(FIXED_TS), SECRET, now=FIXED_TS + 60)
        self.assertEqual((ev["type"], ev["data"]["status"]), ("payout.confirmed", "confirmed"))
        self.assertEqual(sign(EVENT_BODY, FIXED_TS, SECRET), FIXED_SIG)

    def test_parse_event_takes_the_headers_of_a_request(self) -> None:
        now = int(time.time())
        ev = parse_event(EVENT_BODY, {"X-Firma": sign(EVENT_BODY, now, SECRET), "x-timestamp": str(now)}, SECRET)
        self.assertEqual(ev["id"], "0c9d1b2e-7f3a-4b8c-9d0e-1f2a3b4c5d6e")

    def test_bad_events_are_rejected(self) -> None:
        now = int(time.time())
        good = sign(EVENT_BODY, now, SECRET)
        with self.assertRaises(SignatureError):
            verify_event(EVENT_BODY.replace(b"1500.50", b"9500.50"), good, str(now), SECRET)
        with self.assertRaises(SignatureError):
            verify_event(EVENT_BODY, sign(EVENT_BODY, now, "other"), str(now), SECRET)
        old = now - CLOCK_TOLERANCE_SECONDS - 60
        with self.assertRaises(SignatureError):
            verify_event(EVENT_BODY, sign(EVENT_BODY, old, SECRET), str(old), SECRET)
        for sig, ts in ((None, str(now)), (good, None), (good, "ayer"), ("v1=zz", str(now))):
            with self.assertRaises(SignatureError):
                verify_event(EVENT_BODY, sig, ts, SECRET)

    def test_during_a_rotation_either_signature_is_enough(self) -> None:
        now = int(time.time())
        header = f"{sign(EVENT_BODY, now, 'new-secret')} {sign(EVENT_BODY, now, SECRET)}"
        self.assertEqual(verify_event(EVENT_BODY, header, str(now), SECRET)["type"], "payout.confirmed")
        self.assertEqual(verify_event(EVENT_BODY, header, str(now), "new-secret")["type"], "payout.confirmed")
        with self.assertRaises(SignatureError):
            verify_event(EVENT_BODY, header, str(now), "neither")

    def test_json_bodies_are_parsed_once(self) -> None:
        self.assertEqual(json.loads(EVENT_BODY)["data"]["amount"], "1500.50")


if __name__ == "__main__":
    unittest.main()
