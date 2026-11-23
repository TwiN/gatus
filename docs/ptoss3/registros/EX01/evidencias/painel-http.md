# EX01 — Observações no painel

Observação em 23/11/2026, commit `eb1d214a`, porta local 18080.

| Estado e ação | Painel | Detalhe de `recuperacao-http` |
| --- | --- | --- |
| `503`, clicar em `Refresh data` e abrir o cartão | `Unhealthy`, em vermelho | `Unhealthy`, `Issues Detected` e evento `Endpoint is unhealthy` |
| Abrir o último check vermelho em `Recent Checks` | | Informação exibida para `11:40:21`: `✓ [CONNECTED] == true` e `✗ [STATUS] (503) == 200` |
| Restaurar `200`, recarregar o detalhe e voltar ao painel | `Healthy` | `Operational` e evento `Endpoint is healthy` |

O cartão e o resumo do detalhe identificaram a falha visualmente. A condição específica apareceu ao selecionar a barra do check em `Recent Checks`.
