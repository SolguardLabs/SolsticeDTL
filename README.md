# SolsticeDTL

![banner](./assets/banner.png)

SolsticeDTL es un servicio Go para liquidacion de pagos diferidos mediante
ventanas solares de liquidez. Cada epoch abre buckets de reserva por activo,
registra depositos, agrupa obligaciones y cierra la ventana con un snapshot
operativo que permite ejecutar settlements parciales y retiradas posteriores.

El repositorio esta pensado para auditoria de logica economica: la API publica
trabaja con escenarios JSON, genera reportes deterministas y mantiene un
journal de movimientos por cuenta, bucket, pago y activo.

## Componentes

- `src/domain`: entidades de cuentas, activos, ventanas, pagos y reportes.
- `src/ledger`: journal contable y movimientos atomicos del servicio.
- `src/policy`: parametros de comisiones, limites y admision.
- `src/solvency`: snapshots de cierre y reconciliacion por activo.
- `src/engine`: servicio principal y comandos de negocio.
- `src/scenario`: runner de fixtures JSON.
- `tests/node`: tests TypeScript de integracion contra el binario Go.

## Requisitos

- Go 1.22 o superior.
- Node.js 24 o superior para ejecutar tests TypeScript con
  `--experimental-strip-types`.
- Bash para los scripts de CI local.

## Uso

Compilar el servicio:

```bash
node scripts/build.mjs
```

Ejecutar un escenario:

```bash
bin/solsticedtl run tests/fixtures/partial_settlement.json --json
```

Validar un escenario:

```bash
bin/solsticedtl validate tests/fixtures/deposit_close.json
```

Ejecutar tests:

```bash
npm test
```

Ejecutar la validacion completa:

```bash
npm run ci
```

## Flujo De Liquidacion

1. Se registra el activo y las cuentas participantes.
2. Se abre una ventana solar con epoch de apertura y cierre.
3. Los usuarios depositan reserva en el bucket de la ventana.
4. Los pagos diferidos bloquean saldo de la cuenta source.
5. La ventana se cierra y conserva un snapshot de obligaciones y saldos.
6. El operador procesa settlements parciales con fees de servicio.
7. Las cuentas pueden retirar saldo libre admitido por la politica de ventana.

## Estado

El proyecto incluye fixtures de depositos, cierre de ventana, settlements
parciales, retiros y separacion de multiples buckets. La salida JSON esta
diseñada para ser estable en CI y para facilitar revision manual.
