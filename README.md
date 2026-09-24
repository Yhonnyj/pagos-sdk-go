# SDK Go — API de Pagos en Bolívares

Cliente de la API de pagos en bolívares: crear pagos, consultarlos y **verificar
la firma de los webhooks**.

Sin dependencias fuera de la biblioteca estándar.

```bash
go get github.com/Yhonnyj/pagos-sdk-go
```

## API v2 — SDK de TuCapi

La API v2 (cobros y pagos por país y moneda, contrato en
`https://api.tucapi.app/v2/openapi.json`) tiene sus propios SDK en este
mismo repositorio:

| Lenguaje | Carpeta | Instalar |
|---|---|---|
| Go | [`v2/`](v2/) | `go get github.com/Yhonnyj/pagos-sdk-go/v2` |
| Node | [`node/`](node/) | `npm install tucapi` |
| Python | [`python/`](python/) | `pip install tucapi` |

Documentación: <https://api.tucapi.app/v2/docs>.

## Crear un pago

```go
c := pagos.Nuevo("ck_live_...")

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
