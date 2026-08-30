// GENERADO POR scripts/generar_sdk.go DESDE api/openapi.json — NO EDITAR A MANO.
//
// Contrato version 1.0.0.

package pagos

import "time"

// URLPorOmision es el servidor de produccion.
const URLPorOmision = "https://api.tucapi.app"

// VersionDelContrato es la version del contrato que este SDK habla.
const VersionDelContrato = "1.0.0"

// Cabeceras del aviso saliente.
const (
	CabeceraFirma     = "X-Firma"
	CabeceraTimestamp = "X-Timestamp"
	CabeceraEvento    = "X-Evento"
)

// TipoDeFirma es el prefijo de version de la cabecera de firma.
const TipoDeFirma = "v1"

// ToleranciaDeReloj es cuanta diferencia de reloj se acepta al verificar
// un aviso. Fuera de esta ventana se rechaza, por si es uno viejo reenviado.
const ToleranciaDeReloj = 300 * time.Second

// MaximoDeIntentos es cuantas veces se reintenta un aviso no entregado.
const MaximoDeIntentos = 6

// Rutas de la API.
const (
	RutaCrearPago   = "/v1/pagos"        // POST
	RutaVerContrato = "/v1/openapi.json" // GET
	RutaVerPago     = "/v1/pagos/{id}"   // GET
)

// Metodo es como se identifica al beneficiario.
type Metodo string

const (
	PagoMovil     Metodo = "PAGO_MOVIL"
	Transferencia Metodo = "TRANSFERENCIA"
)

// Evento es el desenlace de un pago. Son estos y no hay mas.
type Evento string

const (
	EventoEnviando   Evento = "pago.enviando"
	EventoCompletado Evento = "pago.completado"
	EventoNoEnviado  Evento = "pago.no_enviado"
	EventoRechazado  Evento = "pago.rechazado"
	EventoEnRevision Evento = "pago.en_revision"
)

// Motivo es el codigo estable de por que no salio.
type Motivo string

const (
	MotivoCedulaInvalida       Motivo = "cedula_invalida"
	MotivoTelefonoInvalido     Motivo = "telefono_invalido"
	MotivoBancoInvalido        Motivo = "banco_invalido"
	MotivoMontoInvalido        Motivo = "monto_invalido"
	MotivoDatosInvalidos       Motivo = "datos_invalidos"
	MotivoTiempoAgotado        Motivo = "tiempo_agotado"
	MotivoBancoReceptorRechazo Motivo = "banco_receptor_rechazo"
	MotivoSaldoInsuficiente    Motivo = "saldo_insuficiente"
	MotivoFueraDeHorario       Motivo = "fuera_de_horario"
	MotivoSinConfirmacion      Motivo = "sin_confirmacion"
)

// Codigos de error de la API. Compare por estos, nunca por el mensaje.
const (
	CodigoCuerpoInvalido      = "cuerpo_invalido"
	CodigoDatosInvalidos      = "datos_invalidos"
	CodigoBancoInvalido       = "banco_invalido"
	CodigoNoAutenticado       = "no_autenticado"
	CodigoLlaveInvalida       = "llave_invalida"
	CodigoAlcanceInsuficiente = "alcance_insuficiente"
	CodigoNoExiste            = "no_existe"
	CodigoRutaDesconocida     = "ruta_desconocida"
	CodigoMetodoNoPermitido   = "metodo_no_permitido"
	CodigoNoConsultable       = "no_consultable"
	CodigoSinRiel             = "sin_riel"
	CodigoErrorInterno        = "error_interno"
)

// NuevoPago es el pago que se quiere crear.
type NuevoPago struct {
	// Código de 4 dígitos del banco del beneficiario.
	BancoDestino string `json:"bancoDestino,omitempty"`
	// Cuenta bancaria de 20 dígitos.
	BeneficiarioCuenta string `json:"beneficiarioCuenta,omitempty"`
	// Cédula o RIF del beneficiario.
	BeneficiarioDocumento string `json:"beneficiarioDocumento"`
	// Nombre del beneficiario.
	BeneficiarioNombre string `json:"beneficiarioNombre,omitempty"`
	// Teléfono móvil del beneficiario.
	BeneficiarioTelefono string `json:"beneficiarioTelefono,omitempty"`
	// Su identificador del pago.
	ClaveIdempotencia string `json:"claveIdempotencia"`
	// Texto libre suyo, hasta 200 caracteres.
	Concepto string `json:"concepto,omitempty"`
	Metodo   Metodo `json:"metodo"`
	// Importe en bolívares, como string con punto decimal y hasta dos.
	Monto string `json:"monto"`
	// Por dónde sale el pago.
	Riel string `json:"riel,omitempty"`
}

// Pago es un pago en el vocabulario publico.
type Pago struct {
	// La clave que usted mandó.
	ClaveIdempotencia string `json:"claveIdempotencia"`
	Estado            Evento `json:"estado"`
	// Nuestro identificador del pago.
	Id string `json:"id"`
	// Siempre VES.
	Moneda string `json:"moneda"`
	// Importe en bolívares, con dos decimales.
	Monto  string           `json:"monto"`
	Motivo *MotivoDetallado `json:"motivo,omitempty"`
	// Número de referencia del banco.
	Referencia string `json:"referencia,omitempty"`
	// Si mandar el mismo pago de nuevo tiene sentido.
	Reintentar *bool `json:"reintentar,omitempty"`
}

// MotivoDetallado es el codigo estable mas su texto legible.
type MotivoDetallado struct {
	Codigo Motivo `json:"codigo"`
	// Explicación legible, en español neutro.
	Texto string `json:"texto"`
}

// Aviso es el cuerpo del webhook.
type Aviso struct {
	// La clave que usted mandó al crear el pago.
	ClaveIdempotencia string `json:"claveIdempotencia"`
	Evento            Evento `json:"evento"`
	// Cuándo se generó el aviso, en UTC.
	Fecha string `json:"fecha"`
	// Identifica ESTE aviso.
	IdEvento string `json:"idEvento"`
	// Nuestro id del pago.
	IdPago string `json:"idPago"`
	Moneda string `json:"moneda"`
	// Importe en bolívares, con dos decimales.
	Monto  string           `json:"monto"`
	Motivo *MotivoDetallado `json:"motivo,omitempty"`
	// Referencia del banco.
	Referencia string `json:"referencia,omitempty"`
	// Si reintentar el mismo pago tiene sentido.
	Reintentar bool `json:"reintentar"`
}
