// GENERADO POR scripts/generar_sdk_v2.go DESDE api/openapi-v2.json — NO EDITAR A MANO.
//
// Contract version 2.5.0.

/** The production server. */
export const DEFAULT_BASE_URL = 'https://api.tucapi.app'

/** The version of the API contract this SDK speaks. */
export const CONTRACT_VERSION = '2.5.0'

/** Request and webhook headers. Lower-case: that is how Headers normalises them. */
export const HEADER_IDEMPOTENCY_KEY = 'idempotency-key'
export const HEADER_SIGNATURE = 'x-firma'
export const HEADER_TIMESTAMP = 'x-timestamp'
export const HEADER_EVENT = 'x-evento'
export const HEADER_EVENT_ID = 'x-id-evento'

/** Prefix of the signature header ("v1=<hex>"). */
export const SIGNATURE_VERSION = 'v1'

/** How much clock difference a webhook may carry before it is rejected (ms). */
export const CLOCK_TOLERANCE_MS = 300_000

/** Scopes a key may carry. */
export const SCOPE_OPERACIONES_LEER = 'operaciones:leer'
export const SCOPE_PAYINS_CREAR = 'payins:crear'
export const SCOPE_PAYOUTS_CREAR = 'payouts:crear'
export const SCOPE_SALDOS_LEER = 'saldos:leer'
export const SCOPE_WEBHOOKS_CONFIGURAR = 'webhooks:configurar'

/** Routes of the API, by SDK operation. */
export const ROUTE_BALANCES_LIST = '/v2/balances' // GET
export const ROUTE_BANKS_LIST = '/v2/banks' // GET
export const ROUTE_CAPABILITIES_GET = '/v2/capabilities' // GET
export const ROUTE_EVENTS_LIST = '/v2/events' // GET
export const ROUTE_METHODS_LIST = '/v2/methods' // GET
export const ROUTE_PAYINS_CANCEL = '/v2/payins/{id}/cancel' // POST
export const ROUTE_PAYINS_CONFIRM = '/v2/payins/{id}/confirm' // POST
export const ROUTE_PAYINS_CREATE = '/v2/payins' // POST
export const ROUTE_PAYINS_GET = '/v2/payins/{id}' // GET
export const ROUTE_PAYINS_RESEND_CODE = '/v2/payins/{id}/resend-code' // POST
export const ROUTE_PAYOUTS_CANCEL = '/v2/payouts/{id}/cancel' // POST
export const ROUTE_PAYOUTS_CREATE = '/v2/payouts' // POST
export const ROUTE_PAYOUTS_GET = '/v2/payouts/{id}' // GET
export const ROUTE_TRANSACTIONS_GET = '/v2/transactions/{id}' // GET
export const ROUTE_WEBHOOK_ENDPOINTS_CREATE = '/v2/webhook-endpoints' // POST

/** The table the client drives itself with. */
export interface OperationSpec {
  method: string
  route: string
  scope: string
  /** Carries an Idempotency-Key, and is therefore safe to retry. */
  idempotent: boolean
}

export const OPERATIONS: Readonly<Record<string, OperationSpec>> = {
  'balances.list': { method: 'GET', route: '/v2/balances', scope: 'saldos:leer', idempotent: false },
  'banks.list': { method: 'GET', route: '/v2/banks', scope: '', idempotent: false },
  'capabilities.get': { method: 'GET', route: '/v2/capabilities', scope: '', idempotent: false },
  'events.list': { method: 'GET', route: '/v2/events', scope: 'operaciones:leer', idempotent: false },
  'methods.list': { method: 'GET', route: '/v2/methods', scope: '', idempotent: false },
  'payins.cancel': { method: 'POST', route: '/v2/payins/{id}/cancel', scope: 'payins:crear', idempotent: false },
  'payins.confirm': { method: 'POST', route: '/v2/payins/{id}/confirm', scope: 'payins:crear', idempotent: false },
  'payins.create': { method: 'POST', route: '/v2/payins', scope: 'payins:crear', idempotent: true },
  'payins.get': { method: 'GET', route: '/v2/payins/{id}', scope: 'operaciones:leer', idempotent: false },
  'payins.resendCode': { method: 'POST', route: '/v2/payins/{id}/resend-code', scope: 'payins:crear', idempotent: false },
  'payouts.cancel': { method: 'POST', route: '/v2/payouts/{id}/cancel', scope: 'payouts:crear', idempotent: false },
  'payouts.create': { method: 'POST', route: '/v2/payouts', scope: 'payouts:crear', idempotent: true },
  'payouts.get': { method: 'GET', route: '/v2/payouts/{id}', scope: 'operaciones:leer', idempotent: false },
  'transactions.get': { method: 'GET', route: '/v2/transactions/{id}', scope: 'operaciones:leer', idempotent: false },
  'webhookEndpoints.create': { method: 'POST', route: '/v2/webhook-endpoints', scope: 'webhooks:configurar', idempotent: false },
}

/** A closed set of values. */
export type AccountType = 'mobile' | 'account'
export const ACCOUNT_TYPE_VALUES: readonly AccountType[] = ['mobile', 'account']

/** A closed set of values. */
export type Availability = 'available' | 'temporarily_unavailable' | 'disabled'
export const AVAILABILITY_VALUES: readonly Availability[] = ['available', 'temporarily_unavailable', 'disabled']

/** Part of the contract. */
export interface Balance {
  available: string
  currency: string
  reserved: string
}

/** Part of the contract. */
export interface BalanceList {
  balances: Balance[]
}

/** Part of the contract. */
export interface Bank {
  code: string
  /** Los que admite hoy. */
  methods: MethodCode[]
  name: string
}

/** Part of the contract. */
export interface BankList {
  banks: Bank[]
}

/** Los campos que el método declara en GET /v2/capabilities (fields). */
export interface Beneficiary {
  account_number?: string
  bank_code?: string
  document?: Document
  name?: string
}

/** Part of the contract. */
export interface Capabilities {
  /** Cambia cuando cambia el catálogo. */
  config_version: number
  countries: Country[]
}

/** Part of the contract. */
export interface Confirmation {
  /** El código que recibió su usuario. */
  code: string
}

/** Part of the contract. */
export interface Country {
  code: string
  currencies: Currency[]
  name: string
  payin_methods: Method[]
  payout_methods: Method[]
}

/** company: la creó usted por la API. */
export type CreadoPor = 'company' | 'operator'
export const CREADO_POR_VALUES: readonly CreadoPor[] = ['company', 'operator']

/** Part of the contract. */
export interface CuentaReceptora {
  /** Sólo incoming_transfer: la cuenta de 20 dígitos. */
  account_number?: string
  /** El banco de la cuenta receptora. */
  bank_code: string
  /** El documento del titular, como lo pide el banco emisor. */
  document: string
  /** El nombre del titular. */
  holder: string
  /** Sólo incoming_mobile_payment: el teléfono al que se manda el Pago Móvil. */
  phone?: string
}

/** Part of the contract. */
export interface Currency {
  code: string
  decimals: number
}

/** A closed set of values. */
export type Direction = 'payin' | 'payout'
export const DIRECTION_VALUES: readonly Direction[] = ['payin', 'payout']

/** Part of the contract. */
export interface Document {
  number?: string
  type?: string
}

/** - unauthorized: Falta la cabecera Authorization o la llave de API no es válida. */
export type ErrorCode = 'unauthorized' | 'scope_missing' | 'body_invalid' | 'field_required' | 'field_invalid' | 'method_unavailable' | 'bank_unsupported' | 'amount_out_of_range' | 'currency_unsupported' | 'country_unsupported' | 'idempotency_key_reused' | 'idempotency_key_required' | 'not_found' | 'not_cancellable' | 'code_not_expected' | 'too_many_codes' | 'temporarily_unavailable' | 'route_not_found' | 'method_not_allowed'
export const ERROR_CODE_VALUES: readonly ErrorCode[] = ['unauthorized', 'scope_missing', 'body_invalid', 'field_required', 'field_invalid', 'method_unavailable', 'bank_unsupported', 'amount_out_of_range', 'currency_unsupported', 'country_unsupported', 'idempotency_key_reused', 'idempotency_key_required', 'not_found', 'not_cancellable', 'code_not_expected', 'too_many_codes', 'temporarily_unavailable', 'route_not_found', 'method_not_allowed']

/** Part of the contract. */
export interface ErrorDetail {
  code: ErrorCode
  /** Sólo en los errores de validación: un ítem por campo. */
  details?: FieldError[]
  /** Para mostrar, no para comparar. */
  message: string
}

/** La forma de TODOS los errores de esta API. */
export interface ErrorEnvelope {
  error: ErrorDetail
}

/** Lo que viaja por el webhook y lo que devuelve GET /v2/events. */
export interface Event {
  data: Transaction
  id: string
  occurred_at: string
  type: EventType
}

/** Part of the contract. */
export interface EventList {
  events: Event[]
  /** Para pedir lo que sigue. */
  next_cursor: string | null
}

/** Sólo finales. */
export type EventType = 'payin.confirmed' | 'payin.failed' | 'payout.confirmed' | 'payout.failed'
export const EVENT_TYPE_VALUES: readonly EventType[] = ['payin.confirmed', 'payin.failed', 'payout.confirmed', 'payout.failed']

/** Part of the contract. */
export interface Failure {
  code: FailureCode
  /** Para mostrar, no para comparar. */
  message: string
}

/** - counterparty_rejected: El banco del destinatario rechazó la operación. */
export type FailureCode = 'counterparty_rejected' | 'payer_insufficient_funds' | 'mandate_required' | 'code_rejected' | 'limit_exceeded' | 'outside_hours' | 'invalid_data' | 'expired' | 'cancelled' | 'temporarily_unavailable' | 'rejected'
export const FAILURE_CODE_VALUES: readonly FailureCode[] = ['counterparty_rejected', 'payer_insufficient_funds', 'mandate_required', 'code_rejected', 'limit_exceeded', 'outside_hours', 'invalid_data', 'expired', 'cancelled', 'temporarily_unavailable', 'rejected']

/** Part of the contract. */
export interface Field {
  enum?: string[]
  example?: string
  /** La ruta dentro de payer, con punto para los anidados. */
  name: string
  pattern?: string
  required: boolean
  type: FieldType
}

/** Part of the contract. */
export interface FieldError {
  code: FieldErrorCode
  /** La ruta del campo, con punto: payer. */
  field: string
}

/** A closed set of values. */
export type FieldErrorCode = 'required' | 'invalid' | 'unknown'
export const FIELD_ERROR_CODE_VALUES: readonly FieldErrorCode[] = ['required', 'invalid', 'unknown']

/** A closed set of values. */
export type FieldType = 'string' | 'enum' | 'date'
export const FIELD_TYPE_VALUES: readonly FieldType[] = ['string', 'enum', 'date']

/** Sólo para direct_debit: el contrato de domiciliación ya autorizado en el banco del pagador. */
export interface Mandate {
  contract_date?: string
  /** Alfanumérico, hasta 30: es lo que acepta el banco. */
  contract_id?: string
}

/** Part of the contract. */
export interface Method {
  availability: Availability
  code: MethodCode
  country: string
  currency: string
  direction: Direction
  /** Lo que payer (o beneficiary) tiene que traer para este método. */
  fields: Field[]
  /** null = sin tope. */
  max_amount: string | null
  min_amount: string
  name: string
  /** Sólo cobros recibidos: adónde tiene que pagar su pagador. */
  receiving_account?: CuentaReceptora | null
}

/** Un código del catálogo. */
export type MethodCode = 'debit_otp' | 'direct_debit' | 'incoming_mobile_payment' | 'incoming_transfer' | 'mobile_payment' | 'bank_transfer'
export const METHOD_CODE_VALUES: readonly MethodCode[] = ['debit_otp', 'direct_debit', 'incoming_mobile_payment', 'incoming_transfer', 'mobile_payment', 'bank_transfer']

/** Part of the contract. */
export interface MethodList {
  methods: Method[]
}

/** Part of the contract. */
export interface NewPayin {
  /** Cadena decimal con punto y los decimales de la moneda. */
  amount: string
  /** ISO 3166-1 alfa-2. */
  country: string
  /** ISO 4217. */
  currency: string
  /** Sólo cobros recibidos: cuántas horas se espera el pago. */
  expires_in_hours?: number
  mandate?: Mandate
  /** Lo que usted quiera guardar, hasta 4 KB. */
  metadata?: Record<string, unknown>
  method: MethodCode
  payer: Payer
  /** Su referencia. */
  reference?: string
}

/** Part of the contract. */
export interface NewPayout {
  /** Cadena decimal con punto y los decimales de la moneda. */
  amount: string
  beneficiary: Beneficiary
  /** ISO 3166-1 alfa-2. */
  country: string
  /** ISO 4217. */
  currency: string
  /** Lo que usted quiera guardar, hasta 4 KB. */
  metadata?: Record<string, unknown>
  method: MethodCode
  purpose?: Purpose
  /** Su referencia. */
  reference?: string
}

/** Part of the contract. */
export interface NewWebhookEndpoint {
  /** Tiene que ser https. */
  url: string
}

/** Los campos que el método declara en GET /v2/capabilities (fields). */
export interface Payer {
  account_number?: string
  account_type?: AccountType
  bank_code?: string
  document?: Document
  email?: string
  name?: string
  phone?: string
}

/** awaiting_code: su usuario tiene que confirmar con el código. */
export type PendingReason = 'awaiting_code' | 'awaiting_payment' | 'processing'
export const PENDING_REASON_VALUES: readonly PendingReason[] = ['awaiting_code', 'awaiting_payment', 'processing']

/** Para qué es el pago. */
export type Purpose = 'remittance' | 'payroll'
export const PURPOSE_VALUES: readonly Purpose[] = ['remittance', 'payroll']

/** pending puede requerir una acción suya (pending_reason). */
export type Status = 'pending' | 'confirmed' | 'failed'
export const STATUS_VALUES: readonly Status[] = ['pending', 'confirmed', 'failed']

/** Lo que usted ve de una operación. */
export interface Transaction {
  amount: string
  /** La referencia que dio el banco, cuando la dio. */
  bank_reference: string | null
  /** Sólo con awaiting_code: hasta cuándo vale el código. */
  code_expires_at?: string
  /** Cuándo se confirmó. */
  confirmed_at?: string
  country: string
  created_at: string
  created_by: CreadoPor
  currency: string
  /** Sólo cobros recibidos: hasta cuándo se espera el pago. */
  expires_at?: string
  /** Sólo con failed. */
  failure: Failure | null
  id: string
  metadata?: Record<string, unknown>
  method: MethodCode
  pending_reason?: PendingReason
  purpose?: Purpose
  reference?: string
  status: Status
  type: TransactionType
  updated_at: string
}

/** A closed set of values. */
export type TransactionType = 'payin' | 'payout'
export const TRANSACTION_TYPE_VALUES: readonly TransactionType[] = ['payin', 'payout']

/** Part of the contract. */
export interface WebhookEndpoint {
  created_at: string
  id: string
  /** Para verificar X-Firma. */
  secret: string
  url: string
}

