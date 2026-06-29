# Pradella Voz Clínica

Projeto de **estudo** de uma aplicação SaaS de evolução clínica por voz com IA Generativa e RAG.

Fisioterapeutas e médicos gravam um áudio rápido (~30s) entre atendimentos e recebem a evolução clínica formatada no padrão **SOAP** (Subjetivo, Objetivo, Avaliação, Plano), pronta para copiar no prontuário.

---

## Metodologia: Spec-Driven Development (SDD)

Este projeto adota **Spec-Driven Development** — uma abordagem de desenvolvimento orientada a especificações formais que garante que cada linha de código seja rastreável a um requisito documentado antes de qualquer implementação começar.

### Fluxo SDD utilizado

```
prd.md (visão do produto)
    └─► /speckit-specify   → spec.md        (requisitos, user stories, critérios de sucesso)
            └─► /speckit-plan      → plan.md        (arquitetura, decisões técnicas, estrutura)
                    └─► /speckit-tasks     → tasks.md       (tarefas ordenadas por dependência)
                            └─► /speckit-implement  → código      (implementação guiada pelas tasks)
```

### Artefatos gerados (em `specs/001-voice-clinical-evolution/`)

| Artefato | Descrição |
|---|---|
| `spec.md` | Requisitos funcionais (FR-001 a FR-020), user stories, edge cases e critérios de sucesso mensuráveis |
| `plan.md` | Plano de implementação: stack, estrutura de diretórios, decisões de modelos de IA, constitution check |
| `tasks.md` | Tarefas atômicas ordenadas por dependência, prontas para execução sequencial |
| `quickstart.md` | Guia de validação ponta a ponta mapeado aos requisitos |
| `data-model.md` | Entidades do domínio e modelo de dados |
| `contracts/api.md` | Contratos de API (request/response por endpoint) |
| `research.md` | Decisões técnicas e justificativas (ADRs) |

A Constitution Check (em `plan.md`) valida o plano contra princípios de qualidade, teste, UX, performance e segurança antes de qualquer código ser escrito.

---

## Modelos de IA utilizados

O pipeline usa dois provedores complementares:

### 1. OpenAI Whisper (`whisper-1`) — Transcrição de voz

Responsável pela conversão **áudio → texto**. Escolhido por velocidade e custo-benefício para transcrição em português do Brasil. Não é um modelo generativo; apenas converte a fala em texto bruto para o próximo estágio.

### 2. Claude (Anthropic) — Estruturação SOAP + RAG

Responsável por interpretar o texto transcrito e gerar a evolução clínica estruturada, ancorado em fontes clínicas controladas (diretrizes CREFITO + protocolos) via RAG.

| Modelo | ID | Uso | Custo (entrada/saída por 1M tokens) |
|---|---|---|---|
| **Claude Sonnet 4.6** | `claude-sonnet-4-6` | Padrão — melhor equilíbrio custo/inteligência para alto volume | $3 / $15 |
| **Claude Haiku 4.5** | `claude-haiku-4-5-20251001` | Opção de menor custo para o tier Free | $1 / $5 |
| **Claude Opus 4.8** | `claude-opus-4-8` | Reserva para quando a qualidade clínica exigir máxima capacidade | $5 / $25 |

**Estratégias de otimização de custo:**
- **Prompt caching**: system prompt + guardrails + contexto RAG recuperado ficam em cache (TTL 5min), reduzindo ~90% do custo do prefixo repetido entre chamadas consecutivas.
- **Structured outputs** (`output_config.format`): garante que a saída seja sempre JSON SOAP válido, sem tokens desnecessários de parsing.
- **Streaming**: a resposta é transmitida progressivamente para minimizar o tempo até o primeiro byte percebido pelo usuário.

**Guardrails clínicos (SC-003):** o modelo é explicitamente instruído a estruturar apenas o que foi dito no áudio — nunca inventar diagnósticos, procedimentos, medicações ou códigos ausentes. A suíte de testes em `backend/tests/guardrails/` valida esse comportamento de forma automatizada e obrigatória.

---

## Stack

| Camada | Tecnologia |
|---|---|
| Backend | Go 1.23 · `chi` · `pgx` · `goose` · `anthropic-sdk-go` · `openai-go` · `stripe-go` |
| Frontend | TypeScript 5 · React 18 · Vite · `vite-plugin-pwa` · TanStack Query · Tailwind CSS |
| Banco de dados | PostgreSQL 16 + extensão `pgvector` (Amazon RDS) |
| Infraestrutura | AWS Fargate + ALB (containers Linux) · Terraform/CDK em `infra/` |
| Auth | JWT + bcrypt |
| Billing | Stripe (plano Free: 10 evoluções/mês; Pro: R$ 49,90/mês ilimitado) |

---

## Como testar

### Pré-requisitos

- Go 1.23
- Node 20+
- Docker
- Chaves de API (nunca versionar — usar `.env` local):
  - `ANTHROPIC_API_KEY` — Claude (estruturação SOAP)
  - `OPENAI_API_KEY` — Whisper (transcrição)
  - `STRIPE_SECRET_KEY` + `STRIPE_WEBHOOK_SECRET` — billing

### Setup completo

```bash
# Sobe banco, instala dependências, roda migrations e indexa fontes clínicas
make setup
```

Ou passo a passo:

```bash
# 1. Banco (PostgreSQL + pgvector)
make db-up

# 2. Backend
make setup-backend
cp backend/.env.example backend/.env   # preencher chaves e DATABASE_URL
make migrate                           # cria o schema
make ingest                            # indexa diretrizes CREFITO no pgvector

# 3. Frontend
make setup-frontend
```

### Subir a aplicação

```bash
# Em terminais separados:
make backend    # API Go em localhost:8080
make frontend   # PWA em localhost:5173 (usar HTTPS/localhost para permissão de microfone)
```

### Rodar os testes

```bash
# Todos os testes (backend + frontend)
make test

# Backend apenas (inclui suíte de guardrails — obrigatória)
make test-backend

# Frontend apenas
make test-frontend

# E2E (Playwright)
npx playwright test
```

### Lint

```bash
make lint          # backend (golangci-lint) + frontend (eslint)
make lint-backend
make lint-frontend
```

### Cenários de validação manuais

| # | Cenário | Esperado |
|---|---|---|
| 1 | 1 toque em "Gravar Evolução", fale ~30s de sessão clínica | SOAP em S/O/A/P em ≤10s (p95) |
| 2 | Botão "Copiar" após geração | Texto completo na área de transferência |
| 3 | Áudio que NÃO menciona um procedimento | SOAP não inclui o procedimento ausente (guardrail) |
| 4 | Áudio com quadro com CID identificável | `cid_suggestions` presente, marcado como sugestão |
| 5 | Editar seção do SOAP e copiar | Versão copiada reflete a edição |
| 6 | Gerar 10 evoluções no Free e tentar a 11ª | Retorno 402 `quota_exceeded` com convite de upgrade |
| 7 | No Free, sair e voltar ao app | Evolução anterior não aparece (efêmera) |
| 8 | Como Pro, abrir histórico | Evoluções ordenadas por data, reabríveis |
| 9 | Verificar tabela `evolutions` após geração | Nenhuma PII do paciente gravada; áudio não persistido |

---

## Estrutura do projeto

```
pradella-voz-clinica/
├── prd.md                              # Visão do produto (input para o SDD)
├── specs/001-voice-clinical-evolution/ # Artefatos SDD da feature
│   ├── spec.md · plan.md · tasks.md
│   ├── quickstart.md · data-model.md
│   └── contracts/api.md
├── backend/                            # API Go
│   ├── cmd/api/                        # entrypoint do servidor
│   ├── cmd/ingest/                     # indexador de fontes clínicas
│   ├── internal/
│   │   ├── api/                        # handlers e rotas
│   │   ├── services/                   # transcription, soap, rag, guardrails, quota, billing
│   │   ├── rag/                        # busca vetorial (pgvector)
│   │   └── core/                       # config, JWT, audit log, db
│   ├── migrations/                     # SQL (goose)
│   └── tests/guardrails/               # suíte de guardrails clínicos (obrigatória)
├── frontend/                           # PWA React
│   └── src/
│       ├── components/                 # Recorder, SoapResult, CopyButton
│       └── pages/                      # Login, Home, Resultado, Histórico, Conta
├── infra/                              # IaC (Fargate, ALB, RDS) — Terraform/CDK
└── Makefile                            # comandos de setup, execução e teste
```
