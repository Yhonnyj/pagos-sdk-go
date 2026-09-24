package pagos

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

/*
 * Las pruebas del cliente de COBROS.
 *
 * ═══════════════════════════════════════════════════════════════════════════
 * NO SON LAS DE PAGOS CON OTRO NOMBRE.
 *
 * Un pago sale solo. Un cobro se detiene en el medio a esperar que una persona
 * lea un mensaje y teclee ocho dígitos contra un reloj, y de ahí salen dos
 * propiedades que en pagos no existen y que, si se rompen, no se notan
 * probando a mano:
 *
 *  1. pedir otro código NO puede ser el mismo pedido que crear el cobro, y
 *  2. el código no puede viajar en la URL.
 *
 * Cada una tiene su prueba acá, y las dos fallan si alguien "simplifica" el
 * cliente de la forma que parece obvia.
 */

// espia es lo que le llegó al servidor en un pedido.
type espia struct {
	metodo  string
	destino string // path + query: exactamente lo que anota el log de un proxy
	cuerpo  string
	llave   string
}

// servidorDeCobros contesta siempre lo mismo y anota todo lo que le llegó.
func servidorDeCobros(t *testing.T, estado int, respuesta string) (*Cliente, *[]espia) {
	t.Helper()
	var vistos []espia
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		crudo := make([]byte, r.ContentLength)
		if r.ContentLength > 0 {
			_, _ = r.Body.Read(crudo)
		}
		vistos = append(vistos, espia{
			metodo:  r.Method,
			destino: r.URL.RequestURI(),
			cuerpo:  string(crudo),
			llave:   r.Header.Get("Authorization"),
		})
		w.WriteHeader(estado)
		_, _ = w.Write([]byte(respuesta))
	}))
	t.Cleanup(srv.Close)
	return Nuevo("tuc_live_abc", ConURL(srv.URL), ConHTTP(srv.Client())), &vistos
}

const cobroEsperando = `{"id":"co1","claveIdempotencia":"cobro-1","estado":"cobro.esperando_codigo",` +
	`"monto":"150.00","moneda":"VES","segundosParaVencer":118}`

func TestCrearCobroMandaLaLlaveYElCuerpo(t *testing.T) {
	c, vistos := servidorDeCobros(t, http.StatusCreated, cobroEsperando)

	co, err := c.CrearCobro(context.Background(), NuevoCobro{
		ClaveIdempotencia: "cobro-1", Monto: "150.00",
		PagadorNombre: "Ana Perez", PagadorDocumento: "V12345678",
		PagadorTelefono: "04145555555", BancoPagador: "0105",
	})
	if err != nil {
		t.Fatal(err)
	}

	v := (*vistos)[0]
	if v.llave != "Bearer tuc_live_abc" {
		t.Fatalf("la llave no viajo bien: %q", v.llave)
	}
	if v.metodo != http.MethodPost || v.destino != "/v1/cobros" {
		t.Fatalf("no fue al endpoint de creacion: %s %s", v.metodo, v.destino)
	}
	// EL NOMBRE VA. Es una diferencia real con un pago —el banco lo exige para
	// poder emitir el código— y sin él el cobro no se crea.
	if !strings.Contains(v.cuerpo, `"pagadorNombre":"Ana Perez"`) {
		t.Fatalf("el cuerpo no lleva el nombre del pagador: %s", v.cuerpo)
	}
	if !strings.Contains(v.cuerpo, `"claveIdempotencia":"cobro-1"`) {
		t.Fatalf("el cuerpo no lleva la clave: %s", v.cuerpo)
	}

	if co.Estado != EventoCobroEsperandoCodigo {
		t.Fatalf("estado mal leido: %+v", co)
	}
	// La cuenta regresiva es lo que va a mostrar la pantalla de quien integra.
	if co.SegundosParaVencer == nil || *co.SegundosParaVencer != 118 {
		t.Fatalf("los segundos no llegaron: %+v", co.SegundosParaVencer)
	}
}

/*
REINTENTAR LA CREACIÓN Y PEDIR OTRO CÓDIGO SON DOS PEDIDOS DISTINTOS.

Es la propiedad que sostiene el diseño de esta parte, y la única forma de
romperla es la que parece una simplificación: hacer que PedirOtroCodigo vuelva a
postear /v1/cobros con la misma clave.

Si eso pasara, el reintento que NUESTRO PROPIO CONTRATO recomienda ante una
respuesta perdida le expiraría a la persona el código que tiene en la mano.
*/
func TestPedirOtroCodigoNoEsReintentarLaCreacion(t *testing.T) {
	c, vistos := servidorDeCobros(t, http.StatusOK, cobroEsperando)
	ctx := context.Background()

	orden := NuevoCobro{
		ClaveIdempotencia: "cobro-1", Monto: "150.00",
		PagadorNombre: "Ana Perez", PagadorDocumento: "V12345678",
		PagadorTelefono: "04145555555", BancoPagador: "0105",
	}
	if _, err := c.CrearCobro(ctx, orden); err != nil {
		t.Fatal(err)
	}
	// El reintento de una respuesta perdida: MISMO pedido, MISMA clave.
	if _, err := c.CrearCobro(ctx, orden); err != nil {
		t.Fatal(err)
	}
	// Y recién esto pide otro código.
	if _, err := c.PedirOtroCodigo(ctx, "co1"); err != nil {
		t.Fatal(err)
	}

	quiere := []string{"/v1/cobros", "/v1/cobros", "/v1/cobros/co1/codigo"}
	if len(*vistos) != len(quiere) {
		t.Fatalf("se esperaban %d pedidos y hubo %d", len(quiere), len(*vistos))
	}
	for i, q := range quiere {
		if (*vistos)[i].destino != q {
			t.Errorf("pedido %d fue a %q y tenia que ir a %q", i+1, (*vistos)[i].destino, q)
		}
	}
	// Dicho más fuerte, porque es el error que importa: si reintentar la
	// creación fuera al mismo lado que pedir otro código, un reintento de red
	// le mataría al usuario el código que está tecleando.
	if (*vistos)[1].destino == (*vistos)[2].destino {
		t.Fatal("reintentar la creacion y pedir otro codigo fueron al MISMO endpoint")
	}
}

/*
EL CÓDIGO NO VIAJA EN LA URL. NUNCA.

Una URL se escribe en el log de acceso de cualquier proxy, en el historial del
navegador y en el Referer de la petición siguiente. Un código de autorización
ahí adentro es un permiso de débito sobre la cuenta de una persona, guardado en
texto plano en media docena de lugares que nadie considera secretos.

Por eso confirmar es un POST con el código en el cuerpo, y por eso esta prueba
mira el destino Y las cabeceras, no sólo que la llamada funcione.
*/
func TestElCodigoNoViajaEnLaURLNiEnUnaCabecera(t *testing.T) {
	const codigo = "87654321"

	var destino, cabeceras, cuerpo string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		destino = r.URL.RequestURI()
		var b strings.Builder
		for k, vs := range r.Header {
			b.WriteString(k + ": " + strings.Join(vs, ",") + "\n")
		}
		cabeceras = b.String()
		crudo := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(crudo)
		cuerpo = string(crudo)
		_, _ = w.Write([]byte(`{"id":"co1","estado":"cobro.verificando","monto":"150.00","moneda":"VES"}`))
	}))
	defer srv.Close()

	c := Nuevo("k", ConURL(srv.URL), ConHTTP(srv.Client()))
	co, err := c.ConfirmarCobro(context.Background(), "co1", codigo)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(destino, codigo) {
		t.Fatalf("EL CODIGO ESTA EN LA URL (%q): eso lo escribe cualquier proxy en su log", destino)
	}
	if strings.Contains(cabeceras, codigo) {
		t.Fatalf("EL CODIGO ESTA EN UNA CABECERA:\n%s", cabeceras)
	}
	// El control negativo: la prueba tiene que estar mirando donde el código SÍ
	// está, o pasaría igual con un cliente que no lo manda a ningún lado.
	if !strings.Contains(cuerpo, `"codigo":"`+codigo+`"`) {
		t.Fatalf("el codigo no viajo en el cuerpo: %s", cuerpo)
	}

	// Lo habitual: el débito se confirma contra la red unos segundos después.
	if co.Estado != EventoCobroVerificando {
		t.Fatalf("estado mal leido: %+v", co)
	}
	// Y no vuelve: en el tipo público no hay dónde ponerlo.
	crudo, _ := json.Marshal(co)
	if strings.Contains(string(crudo), codigo) {
		t.Fatalf("el codigo volvio en la respuesta: %s", crudo)
	}
}

// UN ID CON UNA BARRA NO PUEDE FABRICAR OTRA RUTA.
//
// Sin escapar, un id como "co1/confirmar" convertiría una consulta en un pedido
// a otro endpoint. El id lo damos nosotros, pero el cliente no tiene por qué
// confiar en que siempre vaya a ser así.
func TestElIdSeEscapaEnLaRuta(t *testing.T) {
	c, vistos := servidorDeCobros(t, http.StatusOK, cobroEsperando)

	if _, err := c.VerCobro(context.Background(), "co1/confirmar"); err != nil {
		t.Fatal(err)
	}
	if d := (*vistos)[0].destino; d != "/v1/cobros/co1%2Fconfirmar" {
		t.Fatalf("el id no se escapo: %q", d)
	}
}

// El rechazo del código llega tipado, con su código estable.
func TestUnCodigoRechazadoLlegaTipado(t *testing.T) {
	c, _ := servidorDeCobros(t, http.StatusConflict,
		`{"codigo":"codigo_invalido","mensaje":"El codigo no es valido o ya expiro."}`)

	_, err := c.ConfirmarCobro(context.Background(), "co1", "00000000")
	if err == nil {
		t.Fatal("no devolvio error")
	}
	if !EsCodigo(err, CodigoCodigoInvalido) {
		t.Fatalf("el codigo de error no llego: %v", err)
	}
	// Y el texto del error no puede llevar el código que se tecleó: los errores
	// se loguean, y ahí es donde termina lo que se mete en su mensaje.
	if strings.Contains(err.Error(), "00000000") {
		t.Fatalf("el codigo tecleado salio en el texto del error: %v", err)
	}
}

/*
LOS DOS AYUDANTES, CONTRA LOS SIETE ESTADOS.

La tabla está entera a propósito. Son los siete estados que existen, y un octavo
que se agregue mañana entra acá o no entra en ninguna parte: sin la tabla
completa, un estado nuevo caería en el "no" de las dos preguntas y la pantalla
de quien integra se quedaría sin saber qué dibujar.
*/
func TestLosAyudantesCubrenLosSieteEstados(t *testing.T) {
	casos := []struct {
		estado    EventoDeCobro
		esperando bool
		pedirOtro bool
	}{
		{EventoCobroEsperandoCodigo, true, false},
		{EventoCobroVerificando, true, false},
		{EventoCobroCompletado, false, false},
		{EventoCobroCodigoInvalido, false, true}, // el único que ofrece el botón
		{EventoCobroRechazado, false, false},
		{EventoCobroVencido, false, false},
		{EventoCobroEnRevision, false, false},
	}
	if len(casos) != 7 {
		t.Fatalf("la tabla tiene %d estados y el contrato declara 7", len(casos))
	}
	for _, c := range casos {
		co := Cobro{Estado: c.estado}
		if got := co.SigueEsperando(); got != c.esperando {
			t.Errorf("%s: SigueEsperando dio %v y tenia que dar %v", c.estado, got, c.esperando)
		}
		if got := co.HayQuePedirOtroCodigo(); got != c.pedirOtro {
			t.Errorf("%s: HayQuePedirOtroCodigo dio %v y tenia que dar %v", c.estado, got, c.pedirOtro)
		}
	}
}
