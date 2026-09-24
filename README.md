# SDK Go — API de Pagos en Bolívares

Cliente de la API de pagos en bolívares: crear pagos, **cobrar por débito**,
consultarlos y **verificar la firma de los webhooks**.

Sin dependencias fuera de la biblioteca estándar.

```bash
go get github.com/Yhonnyj/pagos-sdk-go
```

## Crear un pago

```go
c := pagos.Nuevo("tuc_live_...")

p, err := c.CrearPago(ctx, pagos.NuevoPago{
	ClaveIdempotencia:     "orden-4821",
	Monto:                 "1500.50",
	Metodo:                pagos.PagoMovil,
	BeneficiarioDocumento: "V12345678",
	BeneficiarioTelefono:  "04141234567",
	BancoDestino:          "0105",
})
```

Reintentar con la **misma** `ClaveIdempotencia` es seguro: devuelve el mismo pago
en vez de crear otro.

## Cobrar

La otra dirección: sacarle plata a un cliente **que lo autoriza con un código**
que recibe por mensaje. Son dos pasos, con una persona en el medio.

```go
co, err := c.CrearCobro(ctx, pagos.NuevoCobro{
	ClaveIdempotencia: "cuota-2026-09-cliente-882",
	Monto:             "1500.50",
	PagadorNombre:     "Maria Perez",
	PagadorDocumento:  "V12345678",
	PagadorTelefono:   "04141234567",
	BancoPagador:      "0102",
})
// su usuario tiene *co.SegundosParaVencer para teclear el código

co, err = c.ConfirmarCobro(ctx, co.ID, codigo)

switch co.Estado {
case pagos.EventoCobroVerificando:    // lo normal: consultar con VerCobro
case pagos.EventoCobroCompletado:     // acreditado, con co.Referencia
case pagos.EventoCobroCodigoInvalido: // pedir OTRO código y volver a confirmar
case pagos.EventoCobroVencido:        // se acabó el tiempo: crear otro cobro
}
```

Cuatro cosas que conviene saber antes de integrarlo:

- **Todos los datos del pagador son obligatorios**, el nombre incluido. Es una
  diferencia con un pago, y la exige el banco para poder emitir el código.
- **El código tiene un solo intento.** Si está equivocado no se puede
  reintentar: hay que pedir uno nuevo con `PedirOtroCodigo`.
- **Pedir otro código invalida el anterior.** Por eso es una llamada aparte y no
  un efecto de reintentar `CrearCobro`: reintentar la creación con la misma
  clave es seguro y **no** le toca el código que su usuario ya tiene.
- **No guarde el código.** Es una autorización de débito sobre la cuenta de una
  persona, no un identificador. Nosotros tampoco lo guardamos.

`co.SigueEsperando()` y `co.HayQuePedirOtroCodigo()` responden, sin leer
estados a mano, si la pantalla sigue esperando algo y si corresponde ofrecer el
botón de pedir otro.

## Recibir el webhook

```go
aviso, err := pagos.LeerAviso(r, miSecreto)
if err != nil {
	http.Error(w, "firma invalida", http.StatusBadRequest)
	return
}
switch aviso.Evento {
case pagos.EventoCompletado: // acreditar
case pagos.EventoNoEnviado:  // reintentar el mismo pago
case pagos.EventoRechazado:  // corregir y crear otro
case pagos.EventoEnRevision: // esperar: NO reintentar
}
```

Deduplique por `aviso.IDEvento`: un reintento nuestro trae el mismo id.

## Documentación

Contrato: `curl https://api.tucapi.app/v1/openapi.json`

## Desarrollo

```bash
go test ./...
```

`pagos/contrato_gen.go` **se genera** desde el contrato OpenAPI. No lo edite a
mano.
