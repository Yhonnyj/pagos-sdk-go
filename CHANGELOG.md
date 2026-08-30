# Cambios del SDK Go

El SDK versiona APARTE del contrato de la API. `VersionDelContrato` dice contra
qué versión del contrato habla; el tag de este repositorio dice qué versión del
SDK es.

## [0.1.1] — 2026-08-30

### ⚠️ Cambio que rompe la compilación

Tres campos exportados se renombraron para respetar las siglas de Go
(staticcheck ST1003):

| v0.1.0 | v0.1.1 |
|---|---|
| `Pago.Id` | `Pago.ID` |
| `Aviso.IdEvento` | `Aviso.IDEvento` |
| `Aviso.IdPago` | `Aviso.IDPago` |

**Para migrar:** renombrar esos tres usos. No cambia nada más — ni el contrato,
ni el JSON, ni el comportamiento. El JSON sigue siendo `id`, `idEvento` e
`idPago`; sólo cambian los nombres del lado de Go.

Se hace ahora, con un único consumidor, porque renombrar un campo exportado
rompe a quien ya integró y el costo sólo crece.

Contrato: sin cambios (v1.0.0).

## [0.1.0] — 2026-08-30

Primera versión. Crear y consultar pagos, y verificar la firma de los webhooks.
Sin dependencias fuera de la biblioteca estándar.
