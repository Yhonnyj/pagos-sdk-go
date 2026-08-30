# SDK Go — API de Pagos en Bolívares

Cliente de la API de pagos en bolívares: crear pagos, consultarlos y **verificar
la firma de los webhooks**.

Sin dependencias fuera de la biblioteca estándar.

```bash
go get github.com/Yhonnyj/pagos-sdk-go
```

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
