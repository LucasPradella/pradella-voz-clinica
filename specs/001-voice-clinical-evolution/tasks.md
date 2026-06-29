---
description: "Task list for Evolução Clínica por Voz"
---

# Tasks: Evolução Clínica por Voz

**Input**: Design documents from `specs/001-voice-clinical-evolution/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/api.md

**Tests**: INCLUÍDOS e OBRIGATÓRIOS por exigência da constituição (Princípio II: domínio
clínico, formatação SOAP e guardrails têm testes antes de "concluído"; guardrails não podem
ser ignorados). CI bloqueia o merge em qualquer falha.

**Organization**: Tarefas agrupadas por user story para implementação e teste independentes.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: pode rodar em paralelo (arquivos diferentes, sem dependências pendentes)
- **[Story]**: a qual user story a tarefa pertence (US1, US2, US3, US4)
- Caminhos de arquivo assumem o layout do plano: `backend/` (Go), `frontend/` (React PWA), `infra/`

## Path Conventions

- Backend Go: `backend/cmd/`, `backend/internal/`, `backend/migrations/`
- Frontend: `frontend/src/`
- Infra: `infra/`

---

## Phase 1: Setup (inicialização do projeto)

- [X] T001 Criar estrutura do repositório (`backend/`, `frontend/`, `infra/`) conforme plan.md
- [X] T002 [P] Inicializar módulo Go em `backend/go.mod` com deps (chi, pgx, pgvector-go, goose, anthropic-sdk-go, openai-go, stripe-go, golang-jwt, bcrypt, testify)
- [X] T003 [P] Inicializar frontend (Vite + React + TS + vite-plugin-pwa + Tailwind) em `frontend/`
- [X] T004 [P] Subir PostgreSQL+pgvector local via `infra/docker-compose.yml`
- [X] T005 [P] Configurar lint/format: `golangci-lint` (`backend/.golangci.yml`) e ESLint/Prettier (`frontend/.eslintrc`)
- [X] T006 [P] Pipeline de CI (lint + testes, bloqueante) em `.github/workflows/ci.yml`
- [X] T007 [P] Centralizar tokens de design (navy/grafite/cinza, sem verde) em `frontend/tailwind.config.ts` e `frontend/src/styles/tokens.css`

---

## Phase 2: Foundational (pré-requisitos bloqueantes — antes de qualquer user story)

- [X] T008 Loader de configuração (env/Secrets Manager) em `backend/internal/core/config.go`
- [X] T009 Pool pgx + registro do tipo vetor (pgvector) em `backend/internal/core/db.go`
- [X] T010 Tooling goose + migration inicial (tabelas users, subscriptions, usage_quotas, evolutions, clinical_sources com `vector`, audit_logs) em `backend/migrations/0001_init.sql`
- [X] T011 [P] Structs de domínio + DTOs em `backend/internal/models/` (User, Subscription, UsageQuota, Evolution, ClinicalSource, AuditLog)
- [X] T012 Router base chi + envelope de erro `{error:{code,message}}` + `GET /api/healthz` em `backend/internal/api/router.go` e `health.go`
- [X] T013 JWT (emitir/verificar) + middleware de autenticação em `backend/internal/core/auth.go`
- [X] T014 Serviço de audit log (LGPD, sem PII do paciente) em `backend/internal/services/audit.go`
- [X] T015 Store de usuário + hash bcrypt em `backend/internal/store/user.go`
- [X] T016 Endpoints `POST /api/auth/register` e `/api/auth/login` (identidade por e-mail) em `backend/internal/api/auth.go`
- [X] T017 [P] Teste de contrato de auth (register/login, 409, 401) em `backend/internal/api/auth_contract_test.go`
- [X] T018 Wrapper do cliente Claude (anthropic-sdk-go) com structured outputs + prompt caching do prefixo em `backend/internal/services/soap_client.go`
- [X] T019 Wrapper do cliente Whisper (openai-go, STT pt-BR) em `backend/internal/services/transcription.go`
- [X] T020 RAG: store de ClinicalSource + busca por similaridade (pgvector) em `backend/internal/rag/store.go`
- [X] T021 Comando de ingestão das fontes clínicas (CREFITO/protocolos) em `backend/cmd/ingest/main.go`
- [X] T022 [P] Cliente de API + contexto de auth/sessão no frontend em `frontend/src/services/api.ts`
- [X] T023 [P] App shell + roteamento + manifest/service worker PWA em `frontend/src/pwa/` e `frontend/src/App.tsx`

**Checkpoint**: base pronta — autenticação, DB, IA e RAG disponíveis para todas as stories.

---

## Phase 3: User Story 1 - Gravar e gerar evolução em SOAP (Priority: P1) 🎯 MVP

**Goal**: profissional grava áudio (1 toque) e recebe SOAP estruturado, copiável, sem PII e
sem invenção clínica.

**Independent Test**: gravar áudio de exemplo e verificar SOAP segmentado em S/O/A/P, CID como
sugestão, nada inventado, e cópia em 1 ação.

### Tests (OBRIGATÓRIOS — constituição) ⚠️

- [X] T024 [P] [US1] Testes de guardrail áudio→SOAP (nada inventado, baixa confiança sinalizada, CID como sugestão) em `backend/internal/services/guardrails_test.go`
- [X] T025 [P] [US1] Teste de contrato `POST /api/evolutions` (200 shape, 422 áudio curto/vazio sem débito, 413 >120s) em `backend/internal/api/evolutions_contract_test.go`

### Implementation

- [X] T026 [US1] Serviço de geração SOAP (prompt + RAG + structured output JSON S/O/A/P + cid_suggestions + confidence) em `backend/internal/services/soap.go`
- [X] T027 [US1] Verificação pós-geração de guardrail (cada procedimento/diagnóstico ancorado na transcrição) em `backend/internal/services/guardrails.go`
- [X] T028 [US1] Store de Evolution (cria no Pro; efêmero no Free; sem PII; source_refs) em `backend/internal/store/evolution.go`
- [X] T029 [US1] Handler `POST /api/evolutions` (pipeline áudio→Whisper→RAG→SOAP, descarta áudio, audit log) em `backend/internal/api/evolutions.go`
- [X] T030 [P] [US1] Componente Recorder (MediaRecorder, 1 toque, limite 120s + aviso, permissão negada) em `frontend/src/components/Recorder.tsx`
- [X] T031 [P] [US1] View do resultado SOAP + estados loading/erro/sucesso em `frontend/src/components/SoapResult.tsx`
- [X] T032 [P] [US1] Botão "Copiar" (1 ação, texto completo) em `frontend/src/components/CopyButton.tsx`
- [X] T033 [P] [US1] Exibição de sugestões CID + flags de confiança em `frontend/src/components/CidSuggestions.tsx`
- [X] T034 [US1] Página Home conectando gravar→processar→exibir SOAP em `frontend/src/pages/Home.tsx`

**Checkpoint**: US1 entregue e testável de forma independente — MVP utilizável.

---

## Phase 4: User Story 2 - Revisar e editar antes de copiar (Priority: P1)

**Goal**: profissional revisa e edita o SOAP/CID gerado antes de copiar/finalizar.

**Independent Test**: gerar evolução, editar um campo, confirmar que a versão copiada/salva
reflete a edição.

### Tests (OBRIGATÓRIOS) ⚠️

- [X] T035 [P] [US2] Teste de contrato `PATCH /api/evolutions/{id}` (edita campos, finaliza, 404 sem acesso) em `backend/internal/api/evolutions_patch_contract_test.go`

### Implementation

- [X] T036 [US2] Handler `PATCH /api/evolutions/{id}` + update no store (Pro persiste; Free em sessão) em `backend/internal/api/evolutions.go`
- [X] T037 [P] [US2] Editor das seções SOAP + editar/remover sugestão CID em `frontend/src/components/SoapEditor.tsx`
- [X] T038 [P] [US2] Destaque de trechos de baixa confiança no resultado em `frontend/src/components/SoapResult.tsx`

**Checkpoint**: US1 + US2 funcionando juntas (gerar + revisar + copiar com confiança).

---

## Phase 5: User Story 3 - Histórico de evoluções (Priority: P2)

**Goal**: usuário Pro consulta, reabre e copia evoluções anteriores.

**Independent Test**: como Pro, gerar evolução, sair e voltar, e vê-la no histórico ordenado
por data; Free recebe 403.

### Tests (OBRIGATÓRIOS) ⚠️

- [X] T039 [P] [US3] Teste de contrato `GET /api/evolutions` e `GET /{id}` (Pro ok; Free 403 `pro_required`; audit log) em `backend/internal/api/evolutions_list_contract_test.go`

### Implementation

- [X] T040 [US3] Store: listar/obter evoluções por usuário (ordenado por data) em `backend/internal/store/evolution.go`
- [X] T041 [US3] Handlers `GET /api/evolutions` + `GET /api/evolutions/{id}` (gate Pro, audit) em `backend/internal/api/evolutions.go`
- [X] T042 [P] [US3] Página de Histórico (lista por data, reabrir, copiar de novo) em `frontend/src/pages/History.tsx`

**Checkpoint**: histórico Pro disponível, sem afetar US1/US2.

---

## Phase 6: User Story 4 - Cadastro, autenticação e limite freemium (Priority: P2)

**Goal**: cota Free de 10/mês com bloqueio e convite de upgrade; Pro ilimitado via Stripe.
(Auth básico de register/login já entregue na Foundational; esta fase cobre cota + assinatura.)

**Independent Test**: gerar 10 evoluções no Free e confirmar que a 11ª é bloqueada (402) com
convite de upgrade; reinício no novo mês.

### Tests (OBRIGATÓRIOS) ⚠️

- [X] T043 [P] [US4] Testes de cota (10 ok, 11ª 402, reset mensal, sem débito em falha/áudio insuficiente) em `backend/internal/services/quota_test.go`

### Implementation

- [X] T044 [US4] Serviço de cota (período mensal de calendário; débito só em sucesso) em `backend/internal/services/quota.go`
- [X] T045 [US4] Aplicar cota em `POST /api/evolutions` (402 `quota_exceeded` + `upgrade:true`) em `backend/internal/api/evolutions.go`
- [X] T046 [US4] Store de assinatura + `GET /api/subscription` (plan, status, quota used/limit) em `backend/internal/api/subscription.go`
- [X] T047 [US4] `POST /api/subscription/checkout` (Stripe Checkout) em `backend/internal/services/billing.go`
- [X] T048 [US4] `POST /api/webhooks/stripe` (valida assinatura; ativa/desativa Pro; libera histórico) em `backend/internal/api/webhooks.go`
- [X] T049 [P] [US4] Páginas de Cadastro/Login em `frontend/src/pages/Auth.tsx`
- [X] T050 [P] [US4] Página de Conta/assinatura + CTA de upgrade em `frontend/src/pages/Account.tsx`
- [X] T051 [US4] UI de cota esgotada (convite de upgrade) integrada ao fluxo da Home em `frontend/src/pages/Home.tsx`

**Checkpoint**: monetização e controle de uso completos.

---

## Phase 7: Polish & Cross-Cutting Concerns

- [X] T052 [P] Teste de performance do pipeline (p95 ≤10s para áudio ≤30s — SC-002) em `backend/internal/api/evolutions_perf_test.go`
- [X] T053 [P] Verificar prompt caching (`cache_read_input_tokens`) + métrica de custo por evolução em `backend/internal/services/soap.go`
- [X] T054 [P] Teste LGPD: nenhuma PII do paciente persistida e áudio não armazenado (SC-008) em `backend/internal/services/lgpd_test.go`
- [X] T055 [P] Acessibilidade (contraste, alvos de toque, foco de teclado) no frontend em `frontend/src/`
- [X] T056 [P] E2E Playwright do caminho feliz (gravar→SOAP→copiar) em `tests/e2e/happy_path.spec.ts`
- [X] T057 [P] IaC: Fargate + ALB + RDS PostgreSQL/pgvector + Secrets Manager em `infra/` (Terraform)
- [X] T058 [P] Observabilidade: logging estruturado + métricas de request em `backend/internal/core/observability.go`
- [X] T059 [P] Rodar `quickstart.md` ponta a ponta e ajustar divergências

---

## Dependencies & Execution Order

- **Setup (Fase 1)** → **Foundational (Fase 2)** → user stories.
- **Foundational bloqueia tudo**: auth, DB, IA, RAG e PWA shell são pré-requisitos.
- **US1 (P1)** depende só da Foundational → primeiro a entregar (MVP).
- **US2 (P1)** depende de US1 (precisa de uma evolução para editar).
- **US3 (P2)** depende da Foundational + Evolution store (US1); independe de US2.
- **US4 (P2)** depende da Foundational; a aplicação da cota toca `POST /api/evolutions` (US1).
- **Polish (Fase 7)** por último.

Ordem recomendada: US1 → US2 → (US3 ‖ US4) → Polish.

## Parallel Execution Examples

- **Setup**: T002, T003, T004, T005, T006, T007 em paralelo.
- **Foundational**: T011, T017, T022, T023 em paralelo (após T008–T010).
- **US1**: T024 e T025 (testes) juntos; depois T030, T031, T032, T033 (componentes frontend) em paralelo enquanto T026–T029 (backend) avançam.
- **US4**: T049 e T050 (páginas) em paralelo com os handlers T046–T048.

## Implementation Strategy

- **MVP primeiro**: completar Fase 1 + Fase 2 + **US1** = produto demonstrável (gravar→SOAP→copiar).
- **Incremental**: adicionar US2 (revisão), depois US3 (histórico Pro) e US4 (freemium/billing).
- **Quality gates**: cada PR exige review + CI verde (lint + testes, incl. guardrails) + metas
  de performance declaradas quando aplicável (constituição).
