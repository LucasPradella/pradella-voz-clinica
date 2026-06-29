# Phase 1 API Contracts: Evolução Clínica por Voz

API REST (Go, roteador `chi`), JSON, autenticação via Bearer JWT (exceto cadastro/login/healthz/webhook).
Erros no formato `{ "error": { "code": "...", "message": "..." } }`. Datas em ISO-8601.

## Auth

### POST /api/auth/register
Cria conta Free (FR-010).
- Body: `{ "email": string, "password": string }`
- 201: `{ "user": { "id", "email", "plan": "free" }, "token": string }`
- 409 `email_taken` se e-mail já existe.

### POST /api/auth/login
- Body: `{ "email": string, "password": string }`
- 200: `{ "token": string, "user": { "id", "email", "plan" } }`
- 401 `invalid_credentials`.

## Evoluções

### POST /api/evolutions
Pipeline áudio→SOAP (FR-002). `multipart/form-data`.
- Form: `audio` (arquivo, ≤120s), `label` (string opcional, sem PII).
- Pré-condição: cota disponível (Free ≤10/mês). Débito **só** em sucesso (FR-018).
- 200: 
  ```json
  {
    "id": "uuid|null",            // null no Free (efêmero, não persistido)
    "soap": { "s": "...", "o": "...", "a": "...", "p": "..." },
    "cid_suggestions": [{ "code": "M54.5", "description": "Dor lombar baixa" }],
    "confidence_flags": [{ "span": "...", "reason": "audio_unclear" }],
    "source_refs": [{ "origin": "CREFITO", "version": "2024" }],
    "status": "draft"
  }
  ```
- 402 `quota_exceeded` (Free atingiu 10): `{ "error": {...}, "upgrade": true }` (FR-011).
- 422 `audio_too_short` / `audio_empty`: não debita cota.
- 413 `audio_too_long` (>120s).
- Observação: o servidor descarta o áudio após gerar (FR-017b) e não armazena PII do
  paciente (FR-017a). Streaming opcional via `Accept: text/event-stream`.

### PATCH /api/evolutions/{id}
Editar a evolução antes de copiar/finalizar (FR-006). Apenas Pro (Free é efêmero).
- Body: `{ "label?", "soap?": {...}, "cid_suggestions?", "status?": "finalized" }`
- 200: evolução atualizada. 404 se não existir/sem acesso.

### GET /api/evolutions
Histórico (FR-014) — **apenas Pro**. Free recebe 403 `pro_required`.
- Query: `?page`, `?limit`.
- 200: `{ "items": [{ "id", "label", "created_at", "status" }], "page", "total" }`.

### GET /api/evolutions/{id}
Detalhe para visualizar/copiar de novo (FR-014). 200 com o mesmo shape do POST.

## Assinatura / Billing

### GET /api/subscription
- 200: `{ "plan": "free|pro", "status", "current_period_end", "quota": { "used": int, "limit": 10|null } }`

### POST /api/subscription/checkout
Inicia upgrade para Pro (Stripe Checkout).
- 200: `{ "checkout_url": string }`.

### POST /api/webhooks/stripe
Webhook do Stripe (sem JWT; valida assinatura). Ativa/desativa Pro e libera histórico.
- 200 `{ "received": true }`.

## Saúde

### GET /api/healthz
- 200: `{ "status": "ok" }`.

## Notas de contrato

- Todas as rotas autenticadas exigem `Authorization: Bearer <jwt>`.
- Acesso a evolução registra `AuditLog` (FR-017).
- `cid_suggestions` são sempre sugestões a validar (FR-005); nunca atribuição automática.
- Testes de contrato (pacotes Go `*_test.go` com `net/http/httptest`) verificam status codes,
  shapes e a regra de débito de cota; testes de guardrail verificam que o SOAP não contém
  procedimento/diagnóstico ausente do áudio (SC-003).
