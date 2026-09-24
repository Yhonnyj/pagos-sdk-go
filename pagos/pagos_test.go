package pagos

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

const secreto = "secreto-de-prueba"

func firmar(cuerpo []byte, t time.Time) string {
	mac := hmac.New(sha256.New, []byte(secreto))
	mac.Write([]byte(strconv.FormatInt(t.Unix(), 10)))
	mac.Write([]byte("."))
	mac.Write(cuerpo)
	return TipoDeFirma + "=" + hex.EncodeToString(mac.Sum(nil))
}

// ═════════════════════════════════════════════════════════════════════════════
// LA VERIFICACIÓN DEL WEBHOOK — lo que el SDK existe para resolver
// ═════════════════════════════════════════════════════════════════════════════

func TestUnAvisoLegitimoSeLee(t *testing.T) {
	cuerpo := []byte(`{"idEvento":"e1","evento":"pago.completado","idPago":"p1",` +
		`"claveIdempotencia":"orden-1","referencia":"290449559","monto":"50.00","moneda":"VES"}`)
	ahora := time.Now()

	a, err := VerificarAviso(cuerpo, firmar(cuerpo, ahora), strconv.FormatInt(ahora.Unix(), 10), secreto)
	if err != nil {
		t.Fatalf("un aviso legitimo no verifico: %v", err)
	}
	if a.Evento != EventoCompletado {
		t.Fatalf("evento=%q", a.Evento)
	}
	if a.Referencia != "290449559" {
		t.Fatalf("referencia=%q", a.Referencia)
	}
}

// UN AVISO FALSIFICADO NO PASA. Es el ataque que importa: alguien manda un
// "pago.completado" que nosotros nunca emitimos.
func TestUnAvisoFalsificadoNoPasa(t *testing.T) {
	falso := []byte(`{"idEvento":"e9","evento":"pago.completado","monto":"999999.00"}`)
	ahora := time.Now()

	// Firmado con OTRO secreto: es lo que puede hacer un atacante.
	mac := hmac.New(sha256.New, []byte("el-atacante-no-tiene-el-secreto"))
	mac.Write([]byte(strconv.FormatInt(ahora.Unix(), 10)))
	mac.Write([]byte("."))
	mac.Write(falso)
	firmaFalsa := TipoDeFirma + "=" + hex.EncodeToString(mac.Sum(nil))

	if _, err := VerificarAviso(falso, firmaFalsa, strconv.FormatInt(ahora.Unix(), 10), secreto); err == nil {
		t.Fatal("un aviso falsificado paso la verificacion")
	}
}

// Cambiar un byte del cuerpo invalida la firma.
func TestUnCuerpoAlteradoNoPasa(t *testing.T) {
	original := []byte(`{"monto":"1.00"}`)
	ahora := time.Now()
	firma := firmar(original, ahora)

	alterado := []byte(`{"monto":"9.00"}`)
	if _, err := VerificarAviso(alterado, firma, strconv.FormatInt(ahora.Unix(), 10), secreto); err == nil {
		t.Fatal("se pudo cambiar el monto de un aviso sin invalidar la firma")
	}
}

// UN AVISO VIEJO REENVIADO NO PASA, aunque la firma sea perfecta.
func TestUnAvisoViejoNoPasa(t *testing.T) {
	cuerpo := []byte(`{"evento":"pago.completado"}`)
	viejo := time.Now().Add(-2 * ToleranciaDeReloj)

	_, err := VerificarAviso(cuerpo, firmar(cuerpo, viejo), strconv.FormatInt(viejo.Unix(), 10), secreto)
	if err == nil {
		t.Fatal("un aviso viejo paso: se puede capturar uno y reenviarlo para siempre")
	}
	if !errors.Is(err, ErrFirmaInvalida) {
		t.Fatalf("el error no es ErrFirmaInvalida: %v", err)
	}
}

// LeerAviso funciona sobre una petición HTTP real.
func TestLeerAvisoDesdeUnaPeticion(t *testing.T) {
	cuerpo := []byte(`{"idEvento":"e1","evento":"pago.no_enviado","reintentar":true,` +
		`"motivo":{"codigo":"tiempo_agotado","texto":"El banco no completó a tiempo."}}`)
	ahora := time.Now()

	r := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(string(cuerpo)))
	r.Header.Set(CabeceraFirma, firmar(cuerpo, ahora))
	r.Header.Set(CabeceraTimestamp, strconv.FormatInt(ahora.Unix(), 10))

	a, err := LeerAviso(r, secreto)
	if err != nil {
		t.Fatalf("no verifico: %v", err)
	}
	if a.Evento != EventoNoEnviado || !a.Reintentar {
		t.Fatalf("aviso mal leido: %+v", a)
	}
	if a.Motivo == nil || a.Motivo.Codigo != MotivoTiempoAgotado {
		t.Fatalf("motivo mal leido: %+v", a.Motivo)
	}
}

// ═════════════════════════════════════════════════════════════════════════════
// EL CLIENTE
// ═════════════════════════════════════════════════════════════════════════════

func TestCrearPagoMandaLaLlaveYElCuerpo(t *testing.T) {
	var llaveVista, cuerpoVisto string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		llaveVista = r.Header.Get("Authorization")
		b := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(b)
		cuerpoVisto = string(b)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"p1","estado":"pago.completado","monto":"150.00",` +
			`"moneda":"VES","referencia":"290449559"}`))
	}))
	defer srv.Close()

	c := Nuevo("tuc_live_abc", ConURL(srv.URL), ConHTTP(srv.Client()))
	p, err := c.CrearPago(context.Background(), NuevoPago{
		ClaveIdempotencia: "orden-1", Monto: "150.00", Metodo: PagoMovil,
		BeneficiarioDocumento: "V12345678", BeneficiarioTelefono: "04145555555",
		BancoDestino: "0105",
	})
	if err != nil {
		t.Fatal(err)
	}
	if llaveVista != "Bearer tuc_live_abc" {
		t.Fatalf("la llave no viajo bien: %q", llaveVista)
	}
	if !strings.Contains(cuerpoVisto, `"claveIdempotencia":"orden-1"`) {
		t.Fatalf("el cuerpo no lleva la clave: %s", cuerpoVisto)
	}
	if p.Estado != EventoCompletado || p.Referencia != "290449559" {
		t.Fatalf("respuesta mal leida: %+v", p)
	}
}

// Un error de la API llega tipado, con su código estable.
func TestUnErrorDeLaAPILlegaTipado(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"codigo":"datos_invalidos","mensaje":"No se aceptan",` +
			`"detalle":["monto: usa punto decimal"]}`))
	}))
	defer srv.Close()

	c := Nuevo("k", ConURL(srv.URL), ConHTTP(srv.Client()))
	_, err := c.CrearPago(context.Background(), NuevoPago{})
	if err == nil {
		t.Fatal("no devolvio error")
	}
	if !EsCodigo(err, "datos_invalidos") {
		t.Fatalf("el codigo no llego: %v", err)
	}
	var e *Error
	if !errors.As(err, &e) || e.Estado != http.StatusBadRequest {
		t.Fatalf("el estado HTTP no llego: %v", err)
	}
}

// SePuedeReintentar no explota con un pago sin ese campo.
func TestSePuedeReintentarConCampoAusente(t *testing.T) {
	var p Pago
	if err := json.Unmarshal([]byte(`{"id":"p1","estado":"pago.completado"}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.SePuedeReintentar() {
		t.Fatal("un completado dijo que se puede reintentar")
	}
	if err := json.Unmarshal([]byte(`{"reintentar":true}`), &p); err != nil {
		t.Fatal(err)
	}
	if !p.SePuedeReintentar() {
		t.Fatal("un reintentable dijo que no")
	}
}

// DURANTE UNA ROTACIÓN LLEGAN DOS FIRMAS Y CUALQUIERA VALE.
//
// Es lo que hace que rotar el secreto no le haga perder avisos al cliente:
// mientras dura la ventana, el aviso viaja firmado con el nuevo y con el viejo.
func TestConDosFirmasVerificaLaQueElClienteTenga(t *testing.T) {
	cuerpo := []byte(`{"evento":"pago.completado"}`)
	ahora := time.Now()
	ts := strconv.FormatInt(ahora.Unix(), 10)

	conSecreto := func(s string) string {
		mac := hmac.New(sha256.New, []byte(s))
		mac.Write([]byte(ts))
		mac.Write([]byte("."))
		mac.Write(cuerpo)
		return TipoDeFirma + "=" + hex.EncodeToString(mac.Sum(nil))
	}
	dos := conSecreto("el-nuevo") + " " + conSecreto("el-viejo")

	// El que ya actualizo su copia.
	if a, err := VerificarAviso(cuerpo, dos, ts, "el-nuevo"); err != nil || a.Evento != EventoCompletado {
		t.Fatalf("el secreto nuevo no verifico: %v", err)
	}
	// Y el que todavia no.
	if a, err := VerificarAviso(cuerpo, dos, ts, "el-viejo"); err != nil || a.Evento != EventoCompletado {
		t.Fatalf("el secreto viejo no verifico durante la ventana: %v", err)
	}
	// Un tercero sigue sin poder.
	if _, err := VerificarAviso(cuerpo, dos, ts, "otro"); err == nil {
		t.Fatal("un secreto ajeno verifico contra una cabecera de dos firmas")
	}
}
