// Package pagos es el cliente Go de la API de pagos en bolívares.
//
// ═══════════════════════════════════════════════════════════════════════════
// LO QUE RESUELVE POR VOS, Y POR QUÉ ESTÁ ACÁ Y NO EN TU CÓDIGO.
//
//  1. La autenticación con tu llave.
//  2. El contrato de los pagos, tipado: no adivinás nombres de campos.
//  3. LA VERIFICACIÓN DE LA FIRMA DEL WEBHOOK.
//
// El tercero es el que importa. Verificar mal una firma es la clase de error
// que no se nota: todo anda, los avisos llegan, y resulta que cualquiera puede
// mandarte un "pago.completado" falso y acreditarle plata a quien quiera.
//
// Los dos errores típicos que este SDK evita:
//
//   - comparar la firma con == , que tarda distinto según cuántos caracteres
//     coincidieron y alcanza para adivinarla byte por byte. Acá se usa
//     hmac.Equal, que tarda lo mismo siempre;
//   - no mirar el timestamp, que deja reenviar un aviso viejo para siempre.
//
// ═══════════════════════════════════════════════════════════════════════════
// EJEMPLO
//
//	c := pagos.Nuevo("tuc_live_...", pagos.ConURL("https://api.tucapi.app"))
//
//	p, err := c.CrearPago(ctx, pagos.NuevoPago{
//		ClaveIdempotencia:     "mi-orden-4821",
//		Monto:                 "150.00",
//		Metodo:                pagos.PagoMovil,
//		BeneficiarioDocumento: "V12345678",
//		BeneficiarioTelefono:  "04145555555",
//		BancoDestino:          "0105",
//	})
//
// Y en el receptor del webhook:
//
//	aviso, err := pagos.LeerAviso(r, miSecreto)
//	if err != nil { http.Error(w, "firma invalida", 400); return }
//	switch aviso.Evento {
//	case pagos.EventoCompletado: // acreditar
//	case pagos.EventoNoEnviado:  // reintentar el mismo pago
//	case pagos.EventoRechazado:  // corregir aviso.Motivo.Codigo y crear otro
//	case pagos.EventoEnRevision: // esperar: NO reintentar
//	}
package pagos

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PlazoPorOmision es lo que se espera una respuesta.
//
// 30 s: crear un pago habla con el banco, y el banco puede tardar. Más corto
// haría que el cliente cortara una llamada que del otro lado sigue viva — y
// entonces no sabría si el pago se creó.
const PlazoPorOmision = 30 * time.Second

// SePuedeReintentar dice si mandar el mismo pago de nuevo tiene sentido.
//
// `Reintentar` es un puntero porque el campo NO VIENE cuando el pago salió bien,
// y ahí la ausencia significa "no hubo fracaso". Este método evita tener que
// acordarse de comprobar el nil en cada uso.
func (p Pago) SePuedeReintentar() bool { return p.Reintentar != nil && *p.Reintentar }

// Error es un error devuelto por la API, con su código estable.
type Error struct {
	Estado  int      `json:"-"`
	Codigo  string   `json:"codigo"`
	Mensaje string   `json:"mensaje"`
	Detalle []string `json:"detalle,omitempty"`
}

func (e *Error) Error() string {
	if len(e.Detalle) > 0 {
		return fmt.Sprintf("pagos: %s (%s): %s", e.Mensaje, e.Codigo, strings.Join(e.Detalle, "; "))
	}
	return fmt.Sprintf("pagos: %s (%s)", e.Mensaje, e.Codigo)
}

// EsCodigo dice si el error trae ese código. Para no comparar strings sueltos.
func EsCodigo(err error, codigo string) bool {
	var e *Error
	return errors.As(err, &e) && e.Codigo == codigo
}

// Cliente habla con el orquestador.
type Cliente struct {
	llave string
	base  string
	http  *http.Client
}

// Opcion configura el cliente.
type Opcion func(*Cliente)

// ConURL apunta a otro orquestador (pruebas, o un ambiente propio).
func ConURL(u string) Opcion {
	return func(c *Cliente) { c.base = strings.TrimRight(strings.TrimSpace(u), "/") }
}

// ConHTTP reemplaza el cliente HTTP. Para tests o para tu propio transporte.
func ConHTTP(h *http.Client) Opcion {
	return func(c *Cliente) { c.http = h }
}

// Nuevo arma el cliente con tu llave de API.
func Nuevo(llave string, opciones ...Opcion) *Cliente {
	transporte := http.DefaultTransport.(*http.Transport).Clone()
	// TLS 1.2 mínimo: la llave viaja en cada pedido.
	transporte.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}

	c := &Cliente{
		llave: strings.TrimSpace(llave),
		base:  URLPorOmision,
		http:  &http.Client{Timeout: PlazoPorOmision, Transport: transporte},
	}
	for _, o := range opciones {
		o(c)
	}
	return c
}

// CrearPago dispersa.
//
// ⚠️ SI ESTO DEVUELVE ERROR DE RED, EL PAGO PUEDE HABERSE CREADO IGUAL. No lo
// mandes de nuevo con otra clave de idempotencia: repetí el MISMO pedido con la
// MISMA clave, y vas a recibir el pago que ya existía.
func (c *Cliente) CrearPago(ctx context.Context, p NuevoPago) (Pago, error) {
	var out Pago
	err := c.pedir(ctx, http.MethodPost, "/v1/pagos", p, &out)
	return out, err
}

// VerPago consulta el estado. No le pide nada al banco.
func (c *Cliente) VerPago(ctx context.Context, id string) (Pago, error) {
	var out Pago
	err := c.pedir(ctx, http.MethodGet, "/v1/pagos/"+url.PathEscape(id), nil, &out)
	return out, err
}

// ═════════════════════════════════════════════════════════════════════════════
// COBROS — la otra dirección: sacarle plata a alguien que lo autoriza
// ═════════════════════════════════════════════════════════════════════════════
//
// Cobrar son DOS pasos con una persona en el medio:
//
//	co, _ := c.CrearCobro(ctx, NuevoCobro{...})   // tu usuario recibe un código
//	// ...tu usuario lo teclea en tu pantalla, dentro de *co.SegundosParaVencer...
//	co, _ = c.ConfirmarCobro(ctx, co.ID, codigo)  // se ejecuta el débito
//
// Y el caso que más se ve, que hay que manejar:
//
//	if co.Estado == EventoCobroCodigoInvalido {
//		co, _ = c.PedirOtroCodigo(ctx, co.ID)  // el anterior ya no sirve
//	}

// CrearCobro crea el cobro y le manda el código a tu usuario.
//
// ⚠️ SI ESTO DEVUELVE ERROR DE RED, EL COBRO PUEDE HABERSE CREADO IGUAL.
// Repetí el MISMO pedido con la MISMA clave: recibís el cobro que ya existía y
// —esto es lo importante— NO se le pide otro código a tu usuario, así que el
// que ya tiene en el teléfono sigue sirviendo.
func (c *Cliente) CrearCobro(ctx context.Context, co NuevoCobro) (Cobro, error) {
	var out Cobro
	err := c.pedir(ctx, http.MethodPost, RutaCrearCobro, co, &out)
	return out, err
}

// PedirOtroCodigo le manda a tu usuario un código NUEVO.
//
// ⚠️ EL ANTERIOR DEJA DE SERVIR en cuanto llamás acá. Usá esto sólo cuando tu
// usuario dice que no le llegó el mensaje, o cuando el estado quedó en
// `EventoCobroCodigoInvalido`. Para reintentar un pedido que falló por red, usá
// CrearCobro con la misma clave: eso NO le toca el código.
func (c *Cliente) PedirOtroCodigo(ctx context.Context, id string) (Cobro, error) {
	var out Cobro
	err := c.pedir(ctx, http.MethodPost, "/v1/cobros/"+url.PathEscape(id)+"/codigo", nil, &out)
	return out, err
}

// ConfirmarCobro ejecuta el débito con el código que tecleó tu usuario.
//
// ⚠️ EL CÓDIGO TIENE UN SOLO INTENTO. Si está equivocado no se puede reintentar:
// el estado queda en `EventoCobroCodigoInvalido` y hay que pedir uno nuevo.
//
// ⚠️ NO GUARDES EL CÓDIGO. Es una autorización de débito sobre la cuenta de una
// persona, no un identificador: usalo y descartalo. Nosotros tampoco lo
// guardamos.
//
// Lo normal es recibir `EventoCobroVerificando`: el débito se confirma contra
// la red interbancaria y eso tarda unos segundos. Consultá con VerCobro.
func (c *Cliente) ConfirmarCobro(ctx context.Context, id, codigo string) (Cobro, error) {
	var out Cobro
	err := c.pedir(ctx, http.MethodPost, "/v1/cobros/"+url.PathEscape(id)+"/confirmar",
		ConfirmacionDeCobro{Codigo: codigo}, &out)
	return out, err
}

// VerCobro consulta el estado. No le pide nada al banco.
//
// A diferencia de los pagos, acá SÍ hace falta: lo habitual es que
// ConfirmarCobro devuelva `EventoCobroVerificando` y el desenlace se lea de acá
// unos segundos después.
func (c *Cliente) VerCobro(ctx context.Context, id string) (Cobro, error) {
	var out Cobro
	err := c.pedir(ctx, http.MethodGet, "/v1/cobros/"+url.PathEscape(id), nil, &out)
	return out, err
}

// SigueEsperando dice si el cobro todavía está esperando algo.
//
// Es la pregunta que decide si tu pantalla sigue mostrando el campo del código
// o una cuenta regresiva, y evita que cada integración la escriba con su propio
// `switch` sobre dos estados.
func (co Cobro) SigueEsperando() bool {
	return co.Estado == EventoCobroEsperandoCodigo || co.Estado == EventoCobroVerificando
}

// HayQuePedirOtroCodigo dice si corresponde ofrecer el botón de pedir otro.
func (co Cobro) HayQuePedirOtroCodigo() bool {
	return co.Estado == EventoCobroCodigoInvalido
}

func (c *Cliente) pedir(ctx context.Context, metodo, ruta string, cuerpo, destino any) error {
	var lector io.Reader
	if cuerpo != nil {
		crudo, err := json.Marshal(cuerpo)
		if err != nil {
			return fmt.Errorf("pagos: armando el cuerpo: %w", err)
		}
		lector = bytes.NewReader(crudo)
	}

	pedido, err := http.NewRequestWithContext(ctx, metodo, c.base+ruta, lector)
	if err != nil {
		return fmt.Errorf("pagos: armando el pedido: %w", err)
	}
	pedido.Header.Set("Authorization", "Bearer "+c.llave)
	pedido.Header.Set("Accept", "application/json")
	if cuerpo != nil {
		pedido.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(pedido)
	if err != nil {
		return fmt.Errorf("pagos: %w", err)
	}
	defer res.Body.Close()
	crudo, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("pagos: leyendo la respuesta: %w", err)
	}

	if res.StatusCode >= 400 {
		e := &Error{Estado: res.StatusCode}
		if err := json.Unmarshal(crudo, e); err != nil || e.Codigo == "" {
			e.Codigo = "respuesta_ilegible"
			e.Mensaje = fmt.Sprintf("HTTP %d", res.StatusCode)
		}
		return e
	}
	if destino == nil {
		return nil
	}
	if err := json.Unmarshal(crudo, destino); err != nil {
		return fmt.Errorf("pagos: no se pudo leer la respuesta: %w", err)
	}
	return nil
}

// ═════════════════════════════════════════════════════════════════════════════
// WEBHOOK: LA PARTE DELICADA
// ═════════════════════════════════════════════════════════════════════════════

// Cabeceras del aviso.

// Aviso es el cuerpo del webhook.
// ErrFirmaInvalida se devuelve cuando el aviso no verifica.
var ErrFirmaInvalida = errors.New("pagos: la firma del aviso no es valida")

// LeerAviso lee y VERIFICA un webhook.
//
// Es la función que hay que usar. Devuelve error si la firma no coincide o si
// el aviso está fuera de la ventana de tiempo — y en los dos casos hay que
// descartarlo, no procesarlo.
//
// Consume el cuerpo de la petición: si necesitás los bytes crudos después,
// usá VerificarAviso con los bytes que ya tengas.
func LeerAviso(r *http.Request, secreto string) (Aviso, error) {
	crudo, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return Aviso{}, fmt.Errorf("pagos: leyendo el aviso: %w", err)
	}
	return VerificarAviso(crudo, r.Header.Get(CabeceraFirma), r.Header.Get(CabeceraTimestamp), secreto)
}

// VerificarAviso verifica un cuerpo ya leído.
func VerificarAviso(cuerpo []byte, firma, timestamp, secreto string) (Aviso, error) {
	seg, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return Aviso{}, fmt.Errorf("%w: timestamp invalido", ErrFirmaInvalida)
	}
	cuando := time.Unix(seg, 0)

	// La ventana de tiempo. Sin esto, un aviso legítimo capturado una vez sirve
	// para siempre.
	if d := time.Since(cuando); d > ToleranciaDeReloj || d < -ToleranciaDeReloj {
		return Aviso{}, fmt.Errorf("%w: el aviso esta fuera de la ventana de %s",
			ErrFirmaInvalida, ToleranciaDeReloj)
	}

	mac := hmac.New(sha256.New, []byte(secreto))
	mac.Write([]byte(strconv.FormatInt(cuando.Unix(), 10)))
	mac.Write([]byte("."))
	mac.Write(cuerpo)
	esperada := TipoDeFirma + "=" + hex.EncodeToString(mac.Sum(nil))

	// LA CABECERA PUEDE TRAER VARIAS FIRMAS, separadas por espacio.
	//
	// Pasa durante una rotacion de secreto: mientras dura la ventana mandamos el
	// aviso firmado con el secreto nuevo Y con el anterior, para que puedas
	// actualizar tu copia cuando quieras sin perder ni un aviso. Alcanza con que
	// una coincida.
	//
	// Si solo mirabas la primera, seguis andando: la del secreto vigente va
	// adelante.
	for _, candidata := range strings.Fields(firma) {
		// hmac.Equal y NO ==. Ver el encabezado del paquete.
		if hmac.Equal([]byte(esperada), []byte(candidata)) {
			var a Aviso
			if err := json.Unmarshal(cuerpo, &a); err != nil {
				return Aviso{}, fmt.Errorf("pagos: el aviso no se pudo leer: %w", err)
			}
			return a, nil
		}
	}
	return Aviso{}, ErrFirmaInvalida
}
