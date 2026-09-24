"""Webhook signature verification.

THIS IS THE MOST IMPORTANT PART OF THE SDK. An unverified webhook is an
endpoint where anyone can post a fake ``payout.confirmed`` and make you
settle money that never moved.

The signature is ``HMAC-SHA256`` over ``timestamp + "." + raw body``, hex,
keyed with your webhook secret, with a version prefix::

    X-Firma: v1=8a7f3c...

The timestamp is INSIDE what is signed. Verify over the RAW bytes: ``json.loads``
followed by ``json.dumps`` does not give back the same bytes.
"""

from __future__ import annotations

import hashlib
import hmac
import json
import time
from typing import Any, Mapping

from .contract_gen import CLOCK_TOLERANCE_SECONDS, HEADER_SIGNATURE, HEADER_TIMESTAMP, SIGNATURE_VERSION, Event


class SignatureError(Exception):
    """The webhook signature did not verify. **Discard the event: do not process it.**"""

    def __init__(self, reason: str) -> None:
        super().__init__(f"payments: the webhook signature is not valid: {reason}")
        self.reason = reason


def sign(body: bytes, seconds: int, secret: str) -> str:
    """Computes ``v1=<hex>`` over ``timestamp + "." + body``."""
    mac = hmac.new(secret.encode("utf-8"), digestmod=hashlib.sha256)
    mac.update(str(seconds).encode("ascii"))
    mac.update(b".")
    mac.update(body)
    return f"{SIGNATURE_VERSION}={mac.hexdigest()}"


def verify_event(body: bytes, signature: str | None, timestamp: str | None, secret: str, now: float | None = None) -> Event:
    """Verifies the signature and returns the event.

    Args:
        body: the RAW body, in bytes, as it arrived.
        signature: the signature header. May carry SEVERAL signatures separated
            by spaces during a secret rotation: one match is enough.
        timestamp: the timestamp header (Unix seconds).
        secret: your webhook secret.
    """
    if not signature:
        raise SignatureError("missing signature header")
    if not timestamp:
        raise SignatureError("missing timestamp header")
    try:
        seconds = int(timestamp.strip())
    except ValueError as err:
        raise SignatureError("invalid timestamp") from err
    if abs((time.time() if now is None else now) - seconds) > CLOCK_TOLERANCE_SECONDS:
        raise SignatureError("outside the clock window")
    expected = sign(body, seconds, secret)
    # hmac.compare_digest and NOT ==: constant time.
    if not any(hmac.compare_digest(expected, candidate) for candidate in signature.split()):
        raise SignatureError("no match")
    return json.loads(body.decode("utf-8"))


def parse_event(body: bytes, headers: Mapping[str, Any], secret: str) -> Event:
    """Takes the headers of a request (any case) and verifies the event."""
    lowered = {str(k).lower(): v for k, v in headers.items()}
    return verify_event(body, lowered.get(HEADER_SIGNATURE.lower()), lowered.get(HEADER_TIMESTAMP.lower()), secret)
