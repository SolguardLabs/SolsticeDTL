# Security Policy

## Modelo De Seguridad

SolsticeDTL modela un servicio de liquidacion custodial por ventanas. El motor
asume que las entradas de escenario representan comandos autorizados por la capa
operativa externa y concentra sus controles en accounting, limites de politica,
transiciones de estado y reconciliacion por activo.

## Invariantes Esperadas

- Un pago solo puede liquidarse dentro de la ventana a la que pertenece.
- Un deposito solo incrementa reservas de una ventana abierta.
- Un pago diferido mueve saldo libre del source a saldo reservado.
- Una ventana cerrada conserva un snapshot de reservas, obligaciones y saldos.
- Un settlement parcial no puede superar el outstanding del pago.
- Los fees aplicados se registran como metricas y eventos auditables.
- Las retiradas se registran tanto en la cuenta como en el bucket asociado.

## Validaciones Automatizadas

La suite de CI ejecuta:

- formato Go mediante `gofmt`;
- `go test ./...`;
- `go vet ./...`;
- build del binario `solsticedtl`;
- tests TypeScript de integracion sobre fixtures JSON.

## Gestion De Dependencias

El servicio Go no requiere dependencias externas. Los tests usan Node.js y
Prettier como dependencia de desarrollo. Dependabot revisa GitHub Actions,
modulos Go y dependencias npm.

## Alcance De Revision

La revision recomendada cubre:

- transiciones de ventana;
- calculo de fees;
- accounting de reservas y saldos;
- comportamiento de settlements parciales;
- reportes de reconciliacion;
- consistencia entre comandos JSON y API Go.

## Reportes Internos

Los reportes deben incluir:

- escenario JSON minimo;
- salida de `solsticedtl run`;
- version de Go y Node;
- descripcion del impacto economico observado;
- propuesta de test de regresion.
