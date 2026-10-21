# Contrato JSON

La entrada de `run` es un fixture con `policy`, `assets`, `accounts`, `windows` y `steps`. Cada paso tiene una acción (`deposit`, `queue_payment`, `advance`, `close`, `settle`, `withdraw`, `validate`) y los campos que exige esa acción. El orden de los pasos es significativo y el primer error detiene la ejecución.

La respuesta de `run ... --json` contiene:

| Campo | Uso |
| --- | --- |
| `name`, `epoch` | Identidad del ciclo y época final. |
| `assets`, `accounts`, `windows`, `payments` | Estado de entidades y claims. |
| `reconciliation` | Sumas por activo de reserva, claims, outstanding y fees. |
| `metrics` | Totales de depósitos, retiros y liquidación. |
| `events` | Solo si se solicita `--events`. |

`audit ... --json` emite `windows`, `accounts`, `findings`, `summaries` y `schedule`. Cada elemento de `schedule` identifica pago, ventana, activo, vencimiento, prioridad, outstanding, máximo del siguiente tramo, fee y neto estimados. Solo aparecen pagos abiertos y la consulta no modifica el estado.

```bash
bin/solsticedtl run tests/fixtures/partial_settlement.json --json
bin/solsticedtl audit tests/fixtures/partial_settlement.json --json
```

Los importes son enteros. Para interpretar `300 bps` de fee aplique división entera y las reglas de techo/clamp descritas en [modelo económico](modelo-economico.md). No sume saldos de activos distintos sin una conversión externa explícita.
