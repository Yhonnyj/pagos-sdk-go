# GENERADO POR scripts/generar_sdk_v2.go DESDE api/openapi-v2.json — NO EDITAR A MANO.
#
# Contract version 2.5.0.

from __future__ import annotations

from typing import Any, Literal, TypedDict

#: The production server.
DEFAULT_BASE_URL = "https://api.tucapi.app"

#: The version of the API contract this SDK speaks.
CONTRACT_VERSION = "2.5.0"

#: Request and webhook headers.
HEADER_IDEMPOTENCY_KEY = "Idempotency-Key"
HEADER_SIGNATURE = "X-Firma"
HEADER_TIMESTAMP = "X-Timestamp"
HEADER_EVENT = "X-Evento"
HEADER_EVENT_ID = "X-Id-Evento"

#: Prefix of the signature header ("v1=<hex>").
SIGNATURE_VERSION = "v1"

#: How much clock difference a webhook may carry before it is rejected (seconds).
CLOCK_TOLERANCE_SECONDS = 300

#: Scopes a key may carry.
SCOPE_OPERACIONES_LEER = "operaciones:leer"
SCOPE_PAYINS_CREAR = "payins:crear"
SCOPE_PAYOUTS_CREAR = "payouts:crear"
SCOPE_SALDOS_LEER = "saldos:leer"
SCOPE_WEBHOOKS_CONFIGURAR = "webhooks:configurar"

#: Routes of the API, by SDK operation.
ROUTE_BALANCES_LIST = "/v2/balances"  # GET
ROUTE_BANKS_LIST = "/v2/banks"  # GET
ROUTE_CAPABILITIES_GET = "/v2/capabilities"  # GET
ROUTE_EVENTS_LIST = "/v2/events"  # GET
ROUTE_METHODS_LIST = "/v2/methods"  # GET
ROUTE_PAYINS_CANCEL = "/v2/payins/{id}/cancel"  # POST
ROUTE_PAYINS_CONFIRM = "/v2/payins/{id}/confirm"  # POST
ROUTE_PAYINS_CREATE = "/v2/payins"  # POST
ROUTE_PAYINS_GET = "/v2/payins/{id}"  # GET
ROUTE_PAYINS_RESEND_CODE = "/v2/payins/{id}/resend-code"  # POST
ROUTE_PAYOUTS_CANCEL = "/v2/payouts/{id}/cancel"  # POST
ROUTE_PAYOUTS_CREATE = "/v2/payouts"  # POST
ROUTE_PAYOUTS_GET = "/v2/payouts/{id}"  # GET
ROUTE_TRANSACTIONS_GET = "/v2/transactions/{id}"  # GET
ROUTE_WEBHOOK_ENDPOINTS_CREATE = "/v2/webhook-endpoints"  # POST

#: The table the client drives itself with: (method, route, scope, idempotent).
OPERATIONS: dict[str, tuple[str, str, str, bool]] = {
    "balances.list": ("GET", "/v2/balances", "saldos:leer", False),
    "banks.list": ("GET", "/v2/banks", "", False),
    "capabilities.get": ("GET", "/v2/capabilities", "", False),
    "events.list": ("GET", "/v2/events", "operaciones:leer", False),
    "methods.list": ("GET", "/v2/methods", "", False),
    "payins.cancel": ("POST", "/v2/payins/{id}/cancel", "payins:crear", False),
    "payins.confirm": ("POST", "/v2/payins/{id}/confirm", "payins:crear", False),
    "payins.create": ("POST", "/v2/payins", "payins:crear", True),
    "payins.get": ("GET", "/v2/payins/{id}", "operaciones:leer", False),
    "payins.resendCode": ("POST", "/v2/payins/{id}/resend-code", "payins:crear", False),
    "payouts.cancel": ("POST", "/v2/payouts/{id}/cancel", "payouts:crear", False),
    "payouts.create": ("POST", "/v2/payouts", "payouts:crear", True),
    "payouts.get": ("GET", "/v2/payouts/{id}", "operaciones:leer", False),
    "transactions.get": ("GET", "/v2/transactions/{id}", "operaciones:leer", False),
    "webhookEndpoints.create": ("POST", "/v2/webhook-endpoints", "webhooks:configurar", False),
}

#: A closed set of values.
AccountType = Literal["mobile", "account"]
ACCOUNT_TYPE_MOBILE: AccountType = "mobile"
ACCOUNT_TYPE_ACCOUNT: AccountType = "account"
ACCOUNT_TYPE_VALUES: tuple[AccountType, ...] = ("mobile", "account",)

#: A closed set of values.
Availability = Literal["available", "temporarily_unavailable", "disabled"]
AVAILABILITY_AVAILABLE: Availability = "available"
AVAILABILITY_TEMPORARILY_UNAVAILABLE: Availability = "temporarily_unavailable"
AVAILABILITY_DISABLED: Availability = "disabled"
AVAILABILITY_VALUES: tuple[Availability, ...] = ("available", "temporarily_unavailable", "disabled",)

#: company: la creó usted por la API.
CreadoPor = Literal["company", "operator"]
CREADO_POR_COMPANY: CreadoPor = "company"
CREADO_POR_OPERATOR: CreadoPor = "operator"
CREADO_POR_VALUES: tuple[CreadoPor, ...] = ("company", "operator",)

#: A closed set of values.
Direction = Literal["payin", "payout"]
DIRECTION_PAYIN: Direction = "payin"
DIRECTION_PAYOUT: Direction = "payout"
DIRECTION_VALUES: tuple[Direction, ...] = ("payin", "payout",)

#: - unauthorized: Falta la cabecera Authorization o la llave de API no es válida.
ErrorCode = Literal["unauthorized", "scope_missing", "body_invalid", "field_required", "field_invalid", "method_unavailable", "bank_unsupported", "amount_out_of_range", "currency_unsupported", "country_unsupported", "idempotency_key_reused", "idempotency_key_required", "not_found", "not_cancellable", "code_not_expected", "too_many_codes", "temporarily_unavailable", "route_not_found", "method_not_allowed"]
ERROR_CODE_UNAUTHORIZED: ErrorCode = "unauthorized"
ERROR_CODE_SCOPE_MISSING: ErrorCode = "scope_missing"
ERROR_CODE_BODY_INVALID: ErrorCode = "body_invalid"
ERROR_CODE_FIELD_REQUIRED: ErrorCode = "field_required"
ERROR_CODE_FIELD_INVALID: ErrorCode = "field_invalid"
ERROR_CODE_METHOD_UNAVAILABLE: ErrorCode = "method_unavailable"
ERROR_CODE_BANK_UNSUPPORTED: ErrorCode = "bank_unsupported"
ERROR_CODE_AMOUNT_OUT_OF_RANGE: ErrorCode = "amount_out_of_range"
ERROR_CODE_CURRENCY_UNSUPPORTED: ErrorCode = "currency_unsupported"
ERROR_CODE_COUNTRY_UNSUPPORTED: ErrorCode = "country_unsupported"
ERROR_CODE_IDEMPOTENCY_KEY_REUSED: ErrorCode = "idempotency_key_reused"
ERROR_CODE_IDEMPOTENCY_KEY_REQUIRED: ErrorCode = "idempotency_key_required"
ERROR_CODE_NOT_FOUND: ErrorCode = "not_found"
ERROR_CODE_NOT_CANCELLABLE: ErrorCode = "not_cancellable"
ERROR_CODE_CODE_NOT_EXPECTED: ErrorCode = "code_not_expected"
ERROR_CODE_TOO_MANY_CODES: ErrorCode = "too_many_codes"
ERROR_CODE_TEMPORARILY_UNAVAILABLE: ErrorCode = "temporarily_unavailable"
ERROR_CODE_ROUTE_NOT_FOUND: ErrorCode = "route_not_found"
ERROR_CODE_METHOD_NOT_ALLOWED: ErrorCode = "method_not_allowed"
ERROR_CODE_VALUES: tuple[ErrorCode, ...] = ("unauthorized", "scope_missing", "body_invalid", "field_required", "field_invalid", "method_unavailable", "bank_unsupported", "amount_out_of_range", "currency_unsupported", "country_unsupported", "idempotency_key_reused", "idempotency_key_required", "not_found", "not_cancellable", "code_not_expected", "too_many_codes", "temporarily_unavailable", "route_not_found", "method_not_allowed",)

#: Sólo finales.
EventType = Literal["payin.confirmed", "payin.failed", "payout.confirmed", "payout.failed"]
EVENT_TYPE_PAYIN_CONFIRMED: EventType = "payin.confirmed"
EVENT_TYPE_PAYIN_FAILED: EventType = "payin.failed"
EVENT_TYPE_PAYOUT_CONFIRMED: EventType = "payout.confirmed"
EVENT_TYPE_PAYOUT_FAILED: EventType = "payout.failed"
EVENT_TYPE_VALUES: tuple[EventType, ...] = ("payin.confirmed", "payin.failed", "payout.confirmed", "payout.failed",)

#: - counterparty_rejected: El banco del destinatario rechazó la operación.
FailureCode = Literal["counterparty_rejected", "payer_insufficient_funds", "mandate_required", "code_rejected", "limit_exceeded", "outside_hours", "invalid_data", "expired", "cancelled", "temporarily_unavailable", "rejected"]
FAILURE_CODE_COUNTERPARTY_REJECTED: FailureCode = "counterparty_rejected"
FAILURE_CODE_PAYER_INSUFFICIENT_FUNDS: FailureCode = "payer_insufficient_funds"
FAILURE_CODE_MANDATE_REQUIRED: FailureCode = "mandate_required"
FAILURE_CODE_CODE_REJECTED: FailureCode = "code_rejected"
FAILURE_CODE_LIMIT_EXCEEDED: FailureCode = "limit_exceeded"
FAILURE_CODE_OUTSIDE_HOURS: FailureCode = "outside_hours"
FAILURE_CODE_INVALID_DATA: FailureCode = "invalid_data"
FAILURE_CODE_EXPIRED: FailureCode = "expired"
FAILURE_CODE_CANCELLED: FailureCode = "cancelled"
FAILURE_CODE_TEMPORARILY_UNAVAILABLE: FailureCode = "temporarily_unavailable"
FAILURE_CODE_REJECTED: FailureCode = "rejected"
FAILURE_CODE_VALUES: tuple[FailureCode, ...] = ("counterparty_rejected", "payer_insufficient_funds", "mandate_required", "code_rejected", "limit_exceeded", "outside_hours", "invalid_data", "expired", "cancelled", "temporarily_unavailable", "rejected",)

#: A closed set of values.
FieldErrorCode = Literal["required", "invalid", "unknown"]
FIELD_ERROR_CODE_REQUIRED: FieldErrorCode = "required"
FIELD_ERROR_CODE_INVALID: FieldErrorCode = "invalid"
FIELD_ERROR_CODE_UNKNOWN: FieldErrorCode = "unknown"
FIELD_ERROR_CODE_VALUES: tuple[FieldErrorCode, ...] = ("required", "invalid", "unknown",)

#: A closed set of values.
FieldType = Literal["string", "enum", "date"]
FIELD_TYPE_STRING: FieldType = "string"
FIELD_TYPE_ENUM: FieldType = "enum"
FIELD_TYPE_DATE: FieldType = "date"
FIELD_TYPE_VALUES: tuple[FieldType, ...] = ("string", "enum", "date",)

#: Un código del catálogo.
MethodCode = Literal["debit_otp", "direct_debit", "incoming_mobile_payment", "incoming_transfer", "mobile_payment", "bank_transfer"]
METHOD_CODE_DEBIT_OTP: MethodCode = "debit_otp"
METHOD_CODE_DIRECT_DEBIT: MethodCode = "direct_debit"
METHOD_CODE_INCOMING_MOBILE_PAYMENT: MethodCode = "incoming_mobile_payment"
METHOD_CODE_INCOMING_TRANSFER: MethodCode = "incoming_transfer"
METHOD_CODE_MOBILE_PAYMENT: MethodCode = "mobile_payment"
METHOD_CODE_BANK_TRANSFER: MethodCode = "bank_transfer"
METHOD_CODE_VALUES: tuple[MethodCode, ...] = ("debit_otp", "direct_debit", "incoming_mobile_payment", "incoming_transfer", "mobile_payment", "bank_transfer",)

#: awaiting_code: su usuario tiene que confirmar con el código.
PendingReason = Literal["awaiting_code", "awaiting_payment", "processing"]
PENDING_REASON_AWAITING_CODE: PendingReason = "awaiting_code"
PENDING_REASON_AWAITING_PAYMENT: PendingReason = "awaiting_payment"
PENDING_REASON_PROCESSING: PendingReason = "processing"
PENDING_REASON_VALUES: tuple[PendingReason, ...] = ("awaiting_code", "awaiting_payment", "processing",)

#: Para qué es el pago.
Purpose = Literal["remittance", "payroll"]
PURPOSE_REMITTANCE: Purpose = "remittance"
PURPOSE_PAYROLL: Purpose = "payroll"
PURPOSE_VALUES: tuple[Purpose, ...] = ("remittance", "payroll",)

#: pending puede requerir una acción suya (pending_reason).
Status = Literal["pending", "confirmed", "failed"]
STATUS_PENDING: Status = "pending"
STATUS_CONFIRMED: Status = "confirmed"
STATUS_FAILED: Status = "failed"
STATUS_VALUES: tuple[Status, ...] = ("pending", "confirmed", "failed",)

#: A closed set of values.
TransactionType = Literal["payin", "payout"]
TRANSACTION_TYPE_PAYIN: TransactionType = "payin"
TRANSACTION_TYPE_PAYOUT: TransactionType = "payout"
TRANSACTION_TYPE_VALUES: tuple[TransactionType, ...] = ("payin", "payout",)

class Balance(TypedDict):
    """Part of the contract."""

    available: str
    currency: str
    reserved: str


class BalanceList(TypedDict):
    """Part of the contract."""

    balances: list[Balance]


class Bank(TypedDict):
    """Part of the contract."""

    code: str
    #: Los que admite hoy.
    methods: list[MethodCode]
    name: str


class BankList(TypedDict):
    """Part of the contract."""

    banks: list[Bank]


class Beneficiary(TypedDict, total=False):
    """Los campos que el método declara en GET /v2/capabilities (fields)."""

    account_number: str
    bank_code: str
    document: Document
    name: str


class Capabilities(TypedDict):
    """Part of the contract."""

    #: Cambia cuando cambia el catálogo.
    config_version: int
    countries: list[Country]


class Confirmation(TypedDict):
    """Part of the contract."""

    #: El código que recibió su usuario.
    code: str


class Country(TypedDict):
    """Part of the contract."""

    code: str
    currencies: list[Currency]
    name: str
    payin_methods: list[Method]
    payout_methods: list[Method]


class _CuentaReceptoraRequired(TypedDict):
    bank_code: str
    document: str
    holder: str


class CuentaReceptora(_CuentaReceptoraRequired, total=False):
    """Part of the contract."""

    #: Sólo incoming_transfer: la cuenta de 20 dígitos.
    account_number: str
    #: Sólo incoming_mobile_payment: el teléfono al que se manda el Pago Móvil.
    phone: str


class Currency(TypedDict):
    """Part of the contract."""

    code: str
    decimals: int


class Document(TypedDict, total=False):
    """Part of the contract."""

    number: str
    type: str


class _ErrorDetailRequired(TypedDict):
    code: ErrorCode
    message: str


class ErrorDetail(_ErrorDetailRequired, total=False):
    """Part of the contract."""

    #: Sólo en los errores de validación: un ítem por campo.
    details: list[FieldError]


class ErrorEnvelope(TypedDict):
    """La forma de TODOS los errores de esta API."""

    error: ErrorDetail


class Event(TypedDict):
    """Lo que viaja por el webhook y lo que devuelve GET /v2/events."""

    data: Transaction
    id: str
    occurred_at: str
    type: EventType


class EventList(TypedDict):
    """Part of the contract."""

    events: list[Event]
    #: Para pedir lo que sigue.
    next_cursor: str | None


class Failure(TypedDict):
    """Part of the contract."""

    code: FailureCode
    #: Para mostrar, no para comparar.
    message: str


class _FieldRequired(TypedDict):
    name: str
    required: bool
    type: FieldType


class Field(_FieldRequired, total=False):
    """Part of the contract."""

    enum: list[str]
    example: str
    pattern: str


class FieldError(TypedDict):
    """Part of the contract."""

    code: FieldErrorCode
    #: La ruta del campo, con punto: payer.
    field: str


class Mandate(TypedDict, total=False):
    """Sólo para direct_debit: el contrato de domiciliación ya autorizado en el banco del pagador."""

    contract_date: str
    #: Alfanumérico, hasta 30: es lo que acepta el banco.
    contract_id: str


class _MethodRequired(TypedDict):
    availability: Availability
    code: MethodCode
    country: str
    currency: str
    direction: Direction
    fields: list[Field]
    max_amount: str | None
    min_amount: str
    name: str


class Method(_MethodRequired, total=False):
    """Part of the contract."""

    #: Sólo cobros recibidos: adónde tiene que pagar su pagador.
    receiving_account: CuentaReceptora | None


class MethodList(TypedDict):
    """Part of the contract."""

    methods: list[Method]


class _NewPayinRequired(TypedDict):
    amount: str
    country: str
    currency: str
    method: MethodCode
    payer: Payer


class NewPayin(_NewPayinRequired, total=False):
    """Part of the contract."""

    #: Sólo cobros recibidos: cuántas horas se espera el pago.
    expires_in_hours: int
    mandate: Mandate
    #: Lo que usted quiera guardar, hasta 4 KB.
    metadata: dict[str, Any]
    #: Su referencia.
    reference: str


class _NewPayoutRequired(TypedDict):
    amount: str
    beneficiary: Beneficiary
    country: str
    currency: str
    method: MethodCode


class NewPayout(_NewPayoutRequired, total=False):
    """Part of the contract."""

    #: Lo que usted quiera guardar, hasta 4 KB.
    metadata: dict[str, Any]
    purpose: Purpose
    #: Su referencia.
    reference: str


class NewWebhookEndpoint(TypedDict):
    """Part of the contract."""

    #: Tiene que ser https.
    url: str


class Payer(TypedDict, total=False):
    """Los campos que el método declara en GET /v2/capabilities (fields)."""

    account_number: str
    account_type: AccountType
    bank_code: str
    document: Document
    email: str
    name: str
    phone: str


class _TransactionRequired(TypedDict):
    amount: str
    bank_reference: str | None
    country: str
    created_at: str
    created_by: CreadoPor
    currency: str
    failure: Failure | None
    id: str
    method: MethodCode
    status: Status
    type: TransactionType
    updated_at: str


class Transaction(_TransactionRequired, total=False):
    """Lo que usted ve de una operación."""

    #: Sólo con awaiting_code: hasta cuándo vale el código.
    code_expires_at: str
    #: Cuándo se confirmó.
    confirmed_at: str
    #: Sólo cobros recibidos: hasta cuándo se espera el pago.
    expires_at: str
    metadata: dict[str, Any]
    pending_reason: PendingReason
    purpose: Purpose
    reference: str


class WebhookEndpoint(TypedDict):
    """Part of the contract."""

    created_at: str
    id: str
    #: Para verificar X-Firma.
    secret: str
    url: str


