# Implementation Plan: Evolução Clínica por Voz

**Branch**: `001-voice-clinical-evolution` | **Date**: 2026-06-27 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/001-voice-clinical-evolution/spec.md`

## Summary

PWA mobile-first onde o profissional grava um áudio curto (alvo ~30s, máx. 120s) e recebe a
evolução clínica estruturada em SOAP, pronta para copiar. Pipeline: gravação no navegador →
upload → transcrição (Whisper/OpenAI) → estruturação SOAP via LLM (Claude) ancorada por RAG
(diretrizes CREFITO + protocolos) com guardrails que impedem a IA de inventar diagnóstico ou
procedimento. Sem persistência de PII do paciente; áudio descartado após a geração; histórico
na nuvem exclusivo do plano Pro. Cota Free de 10 evoluções/mês; Pro ilimitado (Stripe).

## Technical Context

**Language/Version**: Backend **Go 1.23**; Frontend TypeScript 5.x (React 18 + Vite, PWA via vite-plugin-pwa)

**Primary Dependencies**:
- Backend (Go): `chi` (roteador HTTP) ou `gin`; `pgx` (driver PostgreSQL) + `pgvector-go`;
  `goose` (migrations); `anthropic-sdk-go` (SDK oficial Claude); `openai-go` (apenas Whisper STT);
  `stripe-go`; `golang-jwt`; `golang.org/x/crypto/bcrypt`
- Frontend: React, Vite, vite-plugin-pwa, MediaRecorder API, TanStack Query, Tailwind (tokens de design centralizados)

**Storage**: Amazon RDS PostgreSQL 16 com extensão `pgvector` para o índice vetorial do RAG
(uma única base gerenciada, custo controlado). Sem bucket de áudio durável (áudio é transitório).

**Testing**: Backend Go `testing` + `testify` (testes de contrato com `net/http/httptest`);
guardrails com suíte dedicada de casos áudio→SOAP; Frontend `vitest` + `@testing-library/react`;
E2E `playwright`.

**Target Platform**: PWA em navegador móvel (iOS Safari 16+, Android Chrome); backend em
container Linux no AWS Fargate atrás de um Application Load Balancer (ALB).

**Project Type**: Web application (frontend PWA + backend API).

**Modelo de IA (decisão)**:
- **Transcrição**: OpenAI Whisper API (`whisper-1`) — rápido e barato para áudio→texto pt-BR
  (provedor de STT definido no PRD; não há LLM generativo aqui).
- **Estruturação SOAP + RAG**: **Claude Sonnet 4.6** (`claude-sonnet-4-6`) como padrão —
  melhor equilíbrio custo/inteligência para extração estruturada de alto volume
  ($3/$15 por 1M tokens, contexto 1M). **Claude Haiku 4.5** (`claude-haiku-4-5`, $1/$5) como
  opção de menor custo para o tier Free, e **Claude Opus 4.8** (`claude-opus-4-8`, $5/$25)
  disponível se a qualidade clínica exigir. Saída via **structured outputs** (`output_config.format`)
  para garantir o JSON SOAP; **prompt caching** do system prompt + guardrails + contexto RAG
  recuperado (TTL 5min) para reduzir ~90% do custo do prefixo repetido.

**Performance Goals**: pipeline áudio→SOAP em ≤10s no p95 para áudios de até 30s (SC-002);
ação primária em 1 toque (SC-001). Streaming da resposta do LLM para minimizar TTFB percebido.

**Constraints**:
- Guardrail clínico não-negociável: 0% de diagnóstico/procedimento inventado (SC-003).
- LGPD: não persistir PII do paciente; descartar áudio após geração; criptografia em trânsito
  e em repouso; audit log de acesso.
- Custo por evolução monitorado (viabilidade do freemium).
- PWA sem instalação de loja na primeira experiência; paleta navy/grafite/cinza, sem verde.

**Scale/Scope**: pico de uso no fim da tarde; escalonamento horizontal no Fargate. MVP B2C
(profissional autônomo); ~5 telas (login/cadastro, home/gravar, resultado SOAP, histórico, conta).

## Constitution Check

*GATE: Deve passar antes da Fase 0. Reavaliar após a Fase 1.*

Baseado em `.specify/memory/constitution.md` v1.0.0:

| Princípio / Seção | Como o plano atende | Status |
|---|---|---|
| I. Qualidade de Código | Camadas separadas (domínio clínico / integração IA / apresentação); lint+format no CI; segredos via env/Secrets Manager; sem complexidade injustificada (uma única RDS com pgvector em vez de banco vetorial extra) | ✅ PASS |
| II. Padrões de Teste | Suíte de guardrails obrigatória (áudio→SOAP, casos negativos) antes de "concluído"; testes de contrato para API/Whisper/RAG; CI bloqueante; teste de regressão por bug | ✅ PASS |
| III. Consistência de UX | Tokens de design centralizados (navy/grafite/cinza, sem verde); "Gravar Evolução" em 1 toque; estados loading/erro/sucesso explícitos; "Copiar" em 1 ação; PWA acessível | ✅ PASS |
| IV. Requisitos de Performance | Meta p95 ≤10s declarada; streaming do LLM; Fargate horizontal; custo/evolução monitorado; testes de performance no pipeline | ✅ PASS |
| Segurança de IA & LGPD | Guardrails testados (FR-004); RAG com fontes rastreáveis (FR-019); sem PII do paciente (FR-017a); descarte de áudio (FR-017b); criptografia + audit log | ✅ PASS |
| Fluxo & Quality Gates | Spec→plan→tasks→implement; PR exige review + CI verde + metas declaradas | ✅ PASS |

Sem violações. Nenhuma entrada necessária em Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/001-voice-clinical-evolution/
├── plan.md              # Este arquivo
├── spec.md              # Especificação da feature
├── research.md          # Fase 0 (decisões técnicas)
├── data-model.md        # Fase 1 (entidades)
├── quickstart.md        # Fase 1 (guia de validação)
├── contracts/           # Fase 1 (contratos de API)
│   └── api.md
└── checklists/
    └── requirements.md
```

### Source Code (repository root)

```text
backend/                 # módulo Go (go.mod)
├── cmd/
│   └── api/             # main.go (entrypoint do servidor)
├── internal/
│   ├── api/             # handlers/rotas chi (auth, evolutions, subscription, webhooks)
│   ├── models/          # structs de domínio + DTOs
│   ├── store/           # acesso a dados (pgx) + queries
│   ├── services/        # transcription, soap, rag, guardrails, quota, billing
│   ├── rag/             # ingestão e busca de fontes clínicas (CREFITO/protocolos)
│   └── core/            # config, segurança (JWT), audit log, db
├── migrations/          # migrations SQL (goose)
└── (testes `_test.go` ao lado dos pacotes)
    # contract/ (httptest), integration/ (pipeline áudio→SOAP, cota, assinatura),
    # guardrails/ (nada inventado, baixa confiança, CID como sugestão)

frontend/
├── src/
│   ├── components/      # Recorder, SoapResult, CopyButton, design tokens
│   ├── pages/          # Login/Cadastro, Home (Gravar), Resultado, Histórico, Conta
│   ├── services/        # cliente da API, auth
│   └── pwa/             # manifest, service worker
└── tests/

infra/                   # IaC (Fargate, ALB, RDS) — Terraform/CDK
```

**Structure Decision**: Web application (Opção 2) — `frontend/` (PWA) + `backend/` (API),
mais `infra/` para IaC. A camada de domínio clínico, integração com IA e apresentação ficam
separadas conforme o Princípio I.

## Complexity Tracking

> Sem violações da Constitution Check. Nada a justificar.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| — | — | — |
