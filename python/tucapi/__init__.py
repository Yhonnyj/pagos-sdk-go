"""TuCapi SDK of the payments API, version 2.

Everything is imported from here: the client, the errors, the webhook
verification, and the contract (types, enums, routes) generated from the
OpenAPI document.
"""

from .client import (
    DEFAULT_MAX_ATTEMPTS,
    DEFAULT_TIMEOUT_SECONDS,
    VERSION,
    TuCapi,
    PaymentsError,
    TransportError,
    new_idempotency_key,
)
from .contract_gen import *  # noqa: F401,F403
from .webhooks import SignatureError, parse_event, sign, verify_event

__all__ = [
    "DEFAULT_MAX_ATTEMPTS",
    "DEFAULT_TIMEOUT_SECONDS",
    "VERSION",
    "TuCapi",
    "PaymentsError",
    "TransportError",
    "SignatureError",
    "new_idempotency_key",
    "parse_event",
    "sign",
    "verify_event",
]
