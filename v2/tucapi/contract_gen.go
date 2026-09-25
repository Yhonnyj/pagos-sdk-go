// GENERADO POR scripts/generar_sdk_v2.go DESDE api/openapi-v2.json — NO EDITAR A MANO.
//
// Contract version 2.7.0.

package tucapi

import "time"

// DefaultBaseURL is the production server.
const DefaultBaseURL = "https://api.tucapi.app"

// ContractVersion is the version of the API contract this SDK speaks.
const ContractVersion = "2.7.0"

// Request and webhook headers.
const (
	HeaderIdempotencyKey = "Idempotency-Key"
	HeaderSignature      = "X-Firma"
	HeaderTimestamp      = "X-Timestamp"
	HeaderEvent          = "X-Evento"
	HeaderEventID        = "X-Id-Evento"
)

// SignatureVersion is the prefix of the signature header ("v1=<hex>").
const SignatureVersion = "v1"

// ClockTolerance is how much clock difference a webhook may carry before it is rejected.
const ClockTolerance = 300 * time.Second

// Scopes a key may carry.
const (
	ScopeOperacionesLeer    = "operaciones:leer"
	ScopePayinsCrear        = "payins:crear"
	ScopePayoutsCrear       = "payouts:crear"
	ScopeSaldosLeer         = "saldos:leer"
	ScopeWebhooksConfigurar = "webhooks:configurar"
)

// Routes of the API, by SDK operation.
const (
	RouteBalancesList           = "/v2/balances"                // GET
	RouteBanksList              = "/v2/banks"                   // GET
	RouteCapabilitiesGet        = "/v2/capabilities"            // GET
	RouteEventsList             = "/v2/events"                  // GET
	RouteMethodsList            = "/v2/methods"                 // GET
	RoutePayinsCancel           = "/v2/payins/{id}/cancel"      // POST
	RoutePayinsConfirm          = "/v2/payins/{id}/confirm"     // POST
	RoutePayinsCreate           = "/v2/payins"                  // POST
	RoutePayinsGet              = "/v2/payins/{id}"             // GET
	RoutePayinsResendCode       = "/v2/payins/{id}/resend-code" // POST
	RoutePayoutsCancel          = "/v2/payouts/{id}/cancel"     // POST
	RoutePayoutsCreate          = "/v2/payouts"                 // POST
	RoutePayoutsGet             = "/v2/payouts/{id}"            // GET
	RouteTransactionsGet        = "/v2/transactions/{id}"       // GET
	RouteWebhookEndpointsCreate = "/v2/webhook-endpoints"       // POST
)

// operations is the table the client drives itself with: which routes carry an
// Idempotency-Key (and are therefore safe to retry) and what they answer.
type operationSpec struct {
	Method, Route, Scope string
	Idempotent           bool
}

var operations = map[string]operationSpec{
	"balances.list":           {Method: "GET", Route: "/v2/balances", Scope: "saldos:leer", Idempotent: false},
	"banks.list":              {Method: "GET", Route: "/v2/banks", Scope: "", Idempotent: false},
	"capabilities.get":        {Method: "GET", Route: "/v2/capabilities", Scope: "", Idempotent: false},
	"events.list":             {Method: "GET", Route: "/v2/events", Scope: "operaciones:leer", Idempotent: false},
	"methods.list":            {Method: "GET", Route: "/v2/methods", Scope: "", Idempotent: false},
	"payins.cancel":           {Method: "POST", Route: "/v2/payins/{id}/cancel", Scope: "payins:crear", Idempotent: false},
	"payins.confirm":          {Method: "POST", Route: "/v2/payins/{id}/confirm", Scope: "payins:crear", Idempotent: false},
	"payins.create":           {Method: "POST", Route: "/v2/payins", Scope: "payins:crear", Idempotent: true},
	"payins.get":              {Method: "GET", Route: "/v2/payins/{id}", Scope: "operaciones:leer", Idempotent: false},
	"payins.resendCode":       {Method: "POST", Route: "/v2/payins/{id}/resend-code", Scope: "payins:crear", Idempotent: false},
	"payouts.cancel":          {Method: "POST", Route: "/v2/payouts/{id}/cancel", Scope: "payouts:crear", Idempotent: false},
	"payouts.create":          {Method: "POST", Route: "/v2/payouts", Scope: "payouts:crear", Idempotent: true},
	"payouts.get":             {Method: "GET", Route: "/v2/payouts/{id}", Scope: "operaciones:leer", Idempotent: false},
	"transactions.get":        {Method: "GET", Route: "/v2/transactions/{id}", Scope: "operaciones:leer", Idempotent: false},
	"webhookEndpoints.create": {Method: "POST", Route: "/v2/webhook-endpoints", Scope: "webhooks:configurar", Idempotent: false},
}

// AccountType is a closed set of values.
type AccountType string

const (
	AccountTypeMobile  AccountType = "mobile"
	AccountTypeAccount AccountType = "account"
)

// AccountTypeValues lists every AccountType.
func AccountTypeValues() []AccountType {
	return []AccountType{AccountTypeMobile, AccountTypeAccount}
}

// Availability is a closed set of values.
type Availability string

const (
	AvailabilityAvailable              Availability = "available"
	AvailabilityTemporarilyUnavailable Availability = "temporarily_unavailable"
	AvailabilityDisabled               Availability = "disabled"
)

// AvailabilityValues lists every Availability.
func AvailabilityValues() []Availability {
	return []Availability{AvailabilityAvailable, AvailabilityTemporarilyUnavailable, AvailabilityDisabled}
}

// Balance is part of the contract.
type Balance struct {
	Available string `json:"available"`
	Currency  string `json:"currency"`
	Reserved  string `json:"reserved"`
}

// BalanceList is part of the contract.
type BalanceList struct {
	Balances []Balance `json:"balances"`
}

// Bank is part of the contract.
type Bank struct {
	Code string `json:"code"`
	// Los que admite hoy.
	Methods []MethodCode `json:"methods"`
	Name    string       `json:"name"`
}

// BankList is part of the contract.
type BankList struct {
	Banks []Bank `json:"banks"`
}

// Beneficiary: Los campos que el método declara en GET /v2/capabilities (fields).
type Beneficiary struct {
	AccountNumber string    `json:"account_number,omitempty"`
	BankCode      string    `json:"bank_code,omitempty"`
	Document      *Document `json:"document,omitempty"`
	Name          string    `json:"name,omitempty"`
}

// Capabilities is part of the contract.
type Capabilities struct {
	// Cambia cuando cambia el catálogo.
	ConfigVersion int       `json:"config_version"`
	Countries     []Country `json:"countries"`
}

// Confirmation is part of the contract.
type Confirmation struct {
	// El código que recibió su usuario.
	Code string `json:"code"`
}

// Country is part of the contract.
type Country struct {
	Code          string     `json:"code"`
	Currencies    []Currency `json:"currencies"`
	Name          string     `json:"name"`
	PayinMethods  []Method   `json:"payin_methods"`
	PayoutMethods []Method   `json:"payout_methods"`
}

// CreadoPor: company: la creó usted por la API.
type CreadoPor string

const (
	CreadoPorCompany  CreadoPor = "company"
	CreadoPorOperator CreadoPor = "operator"
)

// CreadoPorValues lists every CreadoPor.
func CreadoPorValues() []CreadoPor {
	return []CreadoPor{CreadoPorCompany, CreadoPorOperator}
}

// CuentaReceptora is part of the contract.
type CuentaReceptora struct {
	// Sólo incoming_transfer: la cuenta de 20 dígitos.
	AccountNumber string `json:"account_number,omitempty"`
	// El banco de la cuenta receptora.
	BankCode string `json:"bank_code"`
	// El documento del titular, como lo pide el banco emisor.
	Document string `json:"document"`
	// El nombre del titular.
	Holder string `json:"holder"`
	// Sólo incoming_mobile_payment: el teléfono al que se manda el Pago Móvil.
	Phone string `json:"phone,omitempty"`
}

// Currency is part of the contract.
type Currency struct {
	Code     string `json:"code"`
	Decimals int    `json:"decimals"`
}

// Direction is a closed set of values.
type Direction string

const (
	DirectionPayin  Direction = "payin"
	DirectionPayout Direction = "payout"
)

// DirectionValues lists every Direction.
func DirectionValues() []Direction {
	return []Direction{DirectionPayin, DirectionPayout}
}

// Document is part of the contract.
type Document struct {
	Number string `json:"number,omitempty"`
	Type   string `json:"type,omitempty"`
}

// ErrorCode: - unauthorized: Falta la cabecera Authorization o la llave de API no es válida.
type ErrorCode string

const (
	ErrorCodeUnauthorized           ErrorCode = "unauthorized"
	ErrorCodeScopeMissing           ErrorCode = "scope_missing"
	ErrorCodeBodyInvalid            ErrorCode = "body_invalid"
	ErrorCodeFieldRequired          ErrorCode = "field_required"
	ErrorCodeFieldInvalid           ErrorCode = "field_invalid"
	ErrorCodeMethodUnavailable      ErrorCode = "method_unavailable"
	ErrorCodeBankUnsupported        ErrorCode = "bank_unsupported"
	ErrorCodeAmountOutOfRange       ErrorCode = "amount_out_of_range"
	ErrorCodeCurrencyUnsupported    ErrorCode = "currency_unsupported"
	ErrorCodeCountryUnsupported     ErrorCode = "country_unsupported"
	ErrorCodeIdempotencyKeyReused   ErrorCode = "idempotency_key_reused"
	ErrorCodeIdempotencyKeyRequired ErrorCode = "idempotency_key_required"
	ErrorCodeNotFound               ErrorCode = "not_found"
	ErrorCodeNotCancellable         ErrorCode = "not_cancellable"
	ErrorCodeCodeNotExpected        ErrorCode = "code_not_expected"
	ErrorCodeTooManyCodes           ErrorCode = "too_many_codes"
	ErrorCodeTemporarilyUnavailable ErrorCode = "temporarily_unavailable"
	ErrorCodeRouteNotFound          ErrorCode = "route_not_found"
	ErrorCodeMethodNotAllowed       ErrorCode = "method_not_allowed"
)

// ErrorCodeValues lists every ErrorCode.
func ErrorCodeValues() []ErrorCode {
	return []ErrorCode{ErrorCodeUnauthorized, ErrorCodeScopeMissing, ErrorCodeBodyInvalid, ErrorCodeFieldRequired, ErrorCodeFieldInvalid, ErrorCodeMethodUnavailable, ErrorCodeBankUnsupported, ErrorCodeAmountOutOfRange, ErrorCodeCurrencyUnsupported, ErrorCodeCountryUnsupported, ErrorCodeIdempotencyKeyReused, ErrorCodeIdempotencyKeyRequired, ErrorCodeNotFound, ErrorCodeNotCancellable, ErrorCodeCodeNotExpected, ErrorCodeTooManyCodes, ErrorCodeTemporarilyUnavailable, ErrorCodeRouteNotFound, ErrorCodeMethodNotAllowed}
}

// ErrorDetail is part of the contract.
type ErrorDetail struct {
	Code ErrorCode `json:"code"`
	// Sólo en los errores de validación: un ítem por campo.
	Details []FieldError `json:"details,omitempty"`
	// Para mostrar, no para comparar.
	Message string `json:"message"`
}

// ErrorEnvelope: La forma de TODOS los errores de esta API.
type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

// Event: Lo que viaja por el webhook y lo que devuelve GET /v2/events.
type Event struct {
	Data       Transaction `json:"data"`
	ID         string      `json:"id"`
	OccurredAt string      `json:"occurred_at"`
	Type       EventType   `json:"type"`
}

// EventList is part of the contract.
type EventList struct {
	Events []Event `json:"events"`
	// Para pedir lo que sigue.
	NextCursor *string `json:"next_cursor"`
}

// EventType: Sólo finales.
type EventType string

const (
	EventTypePayinConfirmed  EventType = "payin.confirmed"
	EventTypePayinFailed     EventType = "payin.failed"
	EventTypePayoutConfirmed EventType = "payout.confirmed"
	EventTypePayoutFailed    EventType = "payout.failed"
)

// EventTypeValues lists every EventType.
func EventTypeValues() []EventType {
	return []EventType{EventTypePayinConfirmed, EventTypePayinFailed, EventTypePayoutConfirmed, EventTypePayoutFailed}
}

// Failure: Por qué no salió: code es el código principal (fijo, para decidir), reason el motivo normalizado debajo (para explicar), message un texto legible.
type Failure struct {
	Code FailureCode `json:"code"`
	// Para mostrar, no para comparar.
	Message string        `json:"message"`
	Reason  MotivoDeFallo `json:"reason"`
}

// FailureCode: - counterparty_rejected: El banco del destinatario rechazó la operación.
type FailureCode string

const (
	FailureCodeCounterpartyRejected   FailureCode = "counterparty_rejected"
	FailureCodePayerInsufficientFunds FailureCode = "payer_insufficient_funds"
	FailureCodeMandateRequired        FailureCode = "mandate_required"
	FailureCodeCodeRejected           FailureCode = "code_rejected"
	FailureCodeLimitExceeded          FailureCode = "limit_exceeded"
	FailureCodeOutsideHours           FailureCode = "outside_hours"
	FailureCodeInvalidData            FailureCode = "invalid_data"
	FailureCodeExpired                FailureCode = "expired"
	FailureCodeCancelled              FailureCode = "cancelled"
	FailureCodeTemporarilyUnavailable FailureCode = "temporarily_unavailable"
	FailureCodeRejected               FailureCode = "rejected"
)

// FailureCodeValues lists every FailureCode.
func FailureCodeValues() []FailureCode {
	return []FailureCode{FailureCodeCounterpartyRejected, FailureCodePayerInsufficientFunds, FailureCodeMandateRequired, FailureCodeCodeRejected, FailureCodeLimitExceeded, FailureCodeOutsideHours, FailureCodeInvalidData, FailureCodeExpired, FailureCodeCancelled, FailureCodeTemporarilyUnavailable, FailureCodeRejected}
}

// Field is part of the contract.
type Field struct {
	Enum    []string `json:"enum,omitempty"`
	Example string   `json:"example,omitempty"`
	// La ruta dentro de payer, con punto para los anidados.
	Name     string    `json:"name"`
	Pattern  string    `json:"pattern,omitempty"`
	Required bool      `json:"required"`
	Type     FieldType `json:"type"`
}

// FieldError is part of the contract.
type FieldError struct {
	Code FieldErrorCode `json:"code"`
	// La ruta del campo, con punto: payer.
	Field string `json:"field"`
}

// FieldErrorCode is a closed set of values.
type FieldErrorCode string

const (
	FieldErrorCodeRequired FieldErrorCode = "required"
	FieldErrorCodeInvalid  FieldErrorCode = "invalid"
	FieldErrorCodeUnknown  FieldErrorCode = "unknown"
)

// FieldErrorCodeValues lists every FieldErrorCode.
func FieldErrorCodeValues() []FieldErrorCode {
	return []FieldErrorCode{FieldErrorCodeRequired, FieldErrorCodeInvalid, FieldErrorCodeUnknown}
}

// FieldType is a closed set of values.
type FieldType string

const (
	FieldTypeString FieldType = "string"
	FieldTypeEnum   FieldType = "enum"
	FieldTypeDate   FieldType = "date"
)

// FieldTypeValues lists every FieldType.
func FieldTypeValues() []FieldType {
	return []FieldType{FieldTypeString, FieldTypeEnum, FieldTypeDate}
}

// Mandate: Sólo para direct_debit: el contrato de domiciliación ya autorizado en el banco del pagador.
type Mandate struct {
	ContractDate string `json:"contract_date,omitempty"`
	// Alfanumérico, hasta 30: es lo que acepta el banco.
	ContractID string `json:"contract_id,omitempty"`
}

// Method is part of the contract.
type Method struct {
	Availability Availability `json:"availability"`
	Code         MethodCode   `json:"code"`
	Country      string       `json:"country"`
	Currency     string       `json:"currency"`
	Direction    Direction    `json:"direction"`
	// Lo que payer (o beneficiary) tiene que traer para este método.
	Fields []Field `json:"fields"`
	// null = sin tope.
	MaxAmount *string `json:"max_amount"`
	MinAmount string  `json:"min_amount"`
	Name      string  `json:"name"`
	// Sólo cobros recibidos: adónde tiene que pagar su pagador.
	ReceivingAccount *CuentaReceptora `json:"receiving_account,omitempty"`
}

// MethodCode: Un código del catálogo.
type MethodCode string

const (
	MethodCodeDebitOTP              MethodCode = "debit_otp"
	MethodCodeDirectDebit           MethodCode = "direct_debit"
	MethodCodeIncomingMobilePayment MethodCode = "incoming_mobile_payment"
	MethodCodeIncomingTransfer      MethodCode = "incoming_transfer"
	MethodCodeMobilePayment         MethodCode = "mobile_payment"
	MethodCodeBankTransfer          MethodCode = "bank_transfer"
)

// MethodCodeValues lists every MethodCode.
func MethodCodeValues() []MethodCode {
	return []MethodCode{MethodCodeDebitOTP, MethodCodeDirectDebit, MethodCodeIncomingMobilePayment, MethodCodeIncomingTransfer, MethodCodeMobilePayment, MethodCodeBankTransfer}
}

// MethodList is part of the contract.
type MethodList struct {
	Methods []Method `json:"methods"`
}

// MotivoDeFallo: El motivo normalizado de un fallo, con el código al que pertenece:.
type MotivoDeFallo string

const (
	MotivoDeFalloInvalidAccount         MotivoDeFallo = "invalid_account"
	MotivoDeFalloAccountClosed          MotivoDeFallo = "account_closed"
	MotivoDeFalloAccountBlocked         MotivoDeFallo = "account_blocked"
	MotivoDeFalloBeneficiaryMismatch    MotivoDeFallo = "beneficiary_mismatch"
	MotivoDeFalloInvalidPhone           MotivoDeFallo = "invalid_phone"
	MotivoDeFalloInvalidBank            MotivoDeFallo = "invalid_bank"
	MotivoDeFalloInvalidDocument        MotivoDeFallo = "invalid_document"
	MotivoDeFalloOther                  MotivoDeFallo = "other"
	MotivoDeFalloPayerInsufficientFunds MotivoDeFallo = "payer_insufficient_funds"
	MotivoDeFalloCodeInvalid            MotivoDeFallo = "code_invalid"
	MotivoDeFalloCodeExpired            MotivoDeFallo = "code_expired"
	MotivoDeFalloNoMandate              MotivoDeFallo = "no_mandate"
	MotivoDeFalloMandateRevoked         MotivoDeFallo = "mandate_revoked"
	MotivoDeFalloAmountOverLimit        MotivoDeFallo = "amount_over_limit"
	MotivoDeFalloDailyLimit             MotivoDeFallo = "daily_limit"
	MotivoDeFalloOutsideHours           MotivoDeFallo = "outside_hours"
	MotivoDeFalloInvalidData            MotivoDeFallo = "invalid_data"
	MotivoDeFalloBankOffline            MotivoDeFallo = "bank_offline"
	MotivoDeFalloTimeout                MotivoDeFallo = "timeout"
	MotivoDeFalloExpired                MotivoDeFallo = "expired"
	MotivoDeFalloCancelled              MotivoDeFallo = "cancelled"
)

// MotivoDeFalloValues lists every MotivoDeFallo.
func MotivoDeFalloValues() []MotivoDeFallo {
	return []MotivoDeFallo{MotivoDeFalloInvalidAccount, MotivoDeFalloAccountClosed, MotivoDeFalloAccountBlocked, MotivoDeFalloBeneficiaryMismatch, MotivoDeFalloInvalidPhone, MotivoDeFalloInvalidBank, MotivoDeFalloInvalidDocument, MotivoDeFalloOther, MotivoDeFalloPayerInsufficientFunds, MotivoDeFalloCodeInvalid, MotivoDeFalloCodeExpired, MotivoDeFalloNoMandate, MotivoDeFalloMandateRevoked, MotivoDeFalloAmountOverLimit, MotivoDeFalloDailyLimit, MotivoDeFalloOutsideHours, MotivoDeFalloInvalidData, MotivoDeFalloBankOffline, MotivoDeFalloTimeout, MotivoDeFalloExpired, MotivoDeFalloCancelled}
}

// NewPayin is part of the contract.
type NewPayin struct {
	// Cadena decimal con punto y los decimales de la moneda.
	Amount string `json:"amount"`
	// ISO 3166-1 alfa-2.
	Country string `json:"country"`
	// ISO 4217.
	Currency string `json:"currency"`
	// Sólo cobros recibidos: cuántas horas se espera el pago.
	ExpiresInHours *int     `json:"expires_in_hours,omitempty"`
	Mandate        *Mandate `json:"mandate,omitempty"`
	// Lo que usted quiera guardar, hasta 4 KB.
	Metadata map[string]any `json:"metadata,omitempty"`
	Method   MethodCode     `json:"method"`
	Payer    Payer          `json:"payer"`
	// Su referencia.
	Reference string `json:"reference,omitempty"`
}

// NewPayout is part of the contract.
type NewPayout struct {
	// Cadena decimal con punto y los decimales de la moneda.
	Amount      string      `json:"amount"`
	Beneficiary Beneficiary `json:"beneficiary"`
	// ISO 3166-1 alfa-2.
	Country string `json:"country"`
	// ISO 4217.
	Currency string `json:"currency"`
	// Lo que usted quiera guardar, hasta 4 KB.
	Metadata map[string]any `json:"metadata,omitempty"`
	Method   MethodCode     `json:"method"`
	Purpose  Purpose        `json:"purpose,omitempty"`
	// Su referencia.
	Reference string `json:"reference,omitempty"`
}

// NewWebhookEndpoint is part of the contract.
type NewWebhookEndpoint struct {
	// Tiene que ser https.
	URL string `json:"url"`
}

// Payer: Los campos que el método declara en GET /v2/capabilities (fields).
type Payer struct {
	AccountNumber string      `json:"account_number,omitempty"`
	AccountType   AccountType `json:"account_type,omitempty"`
	BankCode      string      `json:"bank_code,omitempty"`
	Document      *Document   `json:"document,omitempty"`
	Email         string      `json:"email,omitempty"`
	Name          string      `json:"name,omitempty"`
	Phone         string      `json:"phone,omitempty"`
}

// PendingReason: awaiting_code: su usuario tiene que confirmar con el código.
type PendingReason string

const (
	PendingReasonAwaitingCode    PendingReason = "awaiting_code"
	PendingReasonAwaitingPayment PendingReason = "awaiting_payment"
	PendingReasonProcessing      PendingReason = "processing"
)

// PendingReasonValues lists every PendingReason.
func PendingReasonValues() []PendingReason {
	return []PendingReason{PendingReasonAwaitingCode, PendingReasonAwaitingPayment, PendingReasonProcessing}
}

// Purpose: Para qué es el pago.
type Purpose string

const (
	PurposeRemittance Purpose = "remittance"
	PurposePayroll    Purpose = "payroll"
)

// PurposeValues lists every Purpose.
func PurposeValues() []Purpose {
	return []Purpose{PurposeRemittance, PurposePayroll}
}

// Status: pending puede requerir una acción suya (pending_reason).
type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusFailed    Status = "failed"
)

// StatusValues lists every Status.
func StatusValues() []Status {
	return []Status{StatusPending, StatusConfirmed, StatusFailed}
}

// Transaction: Lo que usted ve de una operación.
type Transaction struct {
	Amount string `json:"amount"`
	// La referencia del banco: la que la persona ve en su movimiento bancario (en Venezuela, 8 dígitos).
	BankReference *string `json:"bank_reference"`
	// Sólo con awaiting_code: hasta cuándo vale el código.
	CodeExpiresAt string `json:"code_expires_at,omitempty"`
	// El concepto que la contraparte ve en su movimiento bancario, por ejemplo «Pago TCP7K2M9Q»: «Pago» o «Cobro», el prefijo de tres letras de su empresa y seis caracteres tomados del id de la operación.
	Concept string `json:"concept"`
	// Cuándo se confirmó.
	ConfirmedAt string    `json:"confirmed_at,omitempty"`
	Country     string    `json:"country"`
	CreatedAt   string    `json:"created_at"`
	CreatedBy   CreadoPor `json:"created_by"`
	Currency    string    `json:"currency"`
	// Sólo cobros recibidos: hasta cuándo se espera el pago.
	ExpiresAt string `json:"expires_at,omitempty"`
	// Sólo con failed.
	Failure       *Failure       `json:"failure"`
	ID            string         `json:"id"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	Method        MethodCode     `json:"method"`
	PendingReason PendingReason  `json:"pending_reason,omitempty"`
	Purpose       Purpose        `json:"purpose,omitempty"`
	// Su referencia.
	Reference string          `json:"reference,omitempty"`
	Status    Status          `json:"status"`
	Type      TransactionType `json:"type"`
	UpdatedAt string          `json:"updated_at"`
}

// TransactionType is a closed set of values.
type TransactionType string

const (
	TransactionTypePayin  TransactionType = "payin"
	TransactionTypePayout TransactionType = "payout"
)

// TransactionTypeValues lists every TransactionType.
func TransactionTypeValues() []TransactionType {
	return []TransactionType{TransactionTypePayin, TransactionTypePayout}
}

// WebhookEndpoint is part of the contract.
type WebhookEndpoint struct {
	CreatedAt string `json:"created_at"`
	ID        string `json:"id"`
	// Para verificar X-Firma.
	Secret string `json:"secret"`
	URL    string `json:"url"`
}
