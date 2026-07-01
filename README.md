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

O pipeline usa três etapas com provedores configuráveis:

### 1. Groq Whisper (`whisper-large-v3-turbo`) — Transcrição de voz

Responsável pela conversão **áudio → texto**. Usa o endpoint OpenAI-compatible da Groq com o modelo `whisper-large-v3-turbo`. O tier gratuito da Groq é generoso e não exige cartão de crédito.

### 2. LLM configurável — Estruturação SOAP + RAG

Responsável por interpretar o texto transcrito e gerar a evolução clínica estruturada no formato SOAP. O provedor é selecionado pela variável `LLM_PROVIDER` no `.env`:

| `LLM_PROVIDER` | Provedor | Modelo | Custo |
|---|---|---|---|
| `claude` (padrão) | Anthropic | `claude-sonnet-4-6` | Pago — $3/$15 por 1M tokens |
| `gemini` | Google AI Studio | `gemini-2.0-flash` | Free tier (requer ativar billing no projeto GCP para cota >0) |
| `groq` | Groq | `llama-3.3-70b-versatile` | Free tier — recomendado para desenvolvimento |

> **Recomendação para desenvolvimento local:** use `LLM_PROVIDER=groq` — reutiliza a mesma `GROQ_API_KEY` da transcrição, sem custo e sem configuração adicional.

### 3. RAG — Contexto clínico

Embeddings via OpenAI (`text-embedding-ada-002`) para busca de trechos de diretrizes CREFITO e protocolos clínicos no pgvector. A falha do RAG é **não-fatal**: o pipeline continua a geração SOAP sem contexto clínico se o embedding falhar.

**Guardrails clínicos (SC-003):** o modelo é explicitamente instruído a estruturar apenas o que foi dito no áudio — nunca inventar diagnósticos, procedimentos, medicações ou códigos ausentes.

---

## Stack

| Camada | Tecnologia |
|---|---|
| Backend | Go 1.23 · `chi` · `pgx` · `goose` · `anthropic-sdk-go` · `go-openai` · `stripe-go` |
| Frontend | TypeScript 5 · React 18 · Vite · TanStack Query · Tailwind CSS |
| Banco de dados | PostgreSQL 16 + extensão `pgvector` |
| Auth | JWT + bcrypt |
| Billing | Stripe (plano Free: 10 evoluções/mês; Pro: R$ 49,90/mês ilimitado) |

---

## Como testar

### Pré-requisitos

- Go 1.23
- Node 20+
- Docker
- Chaves de API (nunca versionar — usar `.env` local):
  - `GROQ_API_KEY` — **obrigatório** — transcrição (Whisper) e opcionalmente geração SOAP
  - `ANTHROPIC_API_KEY` — obrigatório se `LLM_PROVIDER=claude`
  - `GEMINI_API_KEY` — obrigatório se `LLM_PROVIDER=gemini`
  - `OPENAI_API_KEY` — opcional — embeddings RAG
  - `STRIPE_SECRET_KEY` + `STRIPE_WEBHOOK_SECRET` — opcional para dev local sem billing

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
cp backend/.env.example backend/.env   # preencher chaves
make migrate                           # cria o schema
make ingest                            # indexa diretrizes CREFITO no pgvector

# 3. Frontend
make setup-frontend
```

### Configuração do `.env`

```env
DATABASE_URL=postgres://pradella:dev@localhost:5432/pradella_dev?sslmode=disable
JWT_SECRET=<mínimo 32 caracteres>

# Toggle de LLM: "claude" (pago) | "gemini" (free tier GCP) | "groq" (free tier, recomendado)
LLM_PROVIDER=groq

ANTHROPIC_API_KEY=sk-ant-...   # obrigatório se LLM_PROVIDER=claude
GEMINI_API_KEY=AQ...           # obrigatório se LLM_PROVIDER=gemini
OPENAI_API_KEY=sk-...          # opcional — embeddings RAG
GROQ_API_KEY=gsk_...           # obrigatório — transcrição + SOAP se LLM_PROVIDER=groq
```

### Subir a aplicação

```bash
# Em terminais separados:
make backend    # API Go em localhost:8080
make frontend   # PWA em localhost:3000
```

### Rodar os testes

```bash
make test           # todos os testes
make test-backend   # backend (inclui guardrails)
make test-frontend  # frontend
```

---

## API — Endpoints principais

| Método | Rota | Plano | Descrição |
|---|---|---|---|
| `POST` | `/api/auth/register` | — | Cadastro (cria conta Free) |
| `POST` | `/api/auth/login` | — | Login, retorna JWT |
| `POST` | `/api/evolutions` | Free + Pro | Processa áudio → SOAP |
| `GET` | `/api/evolutions` | Pro | Lista histórico paginado |
| `GET` | `/api/evolutions/{id}` | Pro | Detalhe de uma evolução |
| `PATCH` | `/api/evolutions/{id}` | Pro | Edita rascunho antes de finalizar |
| `DELETE` | `/api/evolutions/{id}` | Pro | Exclui uma evolução |
| `GET` | `/api/subscription` | — | Status do plano e cota |

---

## Cenários de validação manuais

| # | Cenário | Esperado |
|---|---|---|
| 1 | Gravar ~30s de sessão clínica | SOAP em S/O/A/P gerado em ≤10s (p95) |
| 2 | Botão "Copiar" após geração | Texto completo na área de transferência |
| 3 | Áudio que NÃO menciona um procedimento | SOAP não inclui o procedimento ausente (guardrail) |
| 4 | Áudio com quadro com CID identificável | `cid_suggestions` presente, marcado como sugestão |
| 5 | Editar seção do SOAP e copiar | Versão copiada reflete a edição |
| 6 | Gerar 10 evoluções no Free e tentar a 11ª | Retorno 402 `quota_exceeded` com convite de upgrade |
| 7 | No Free, sair e voltar ao app | Evolução anterior não aparece (efêmera) |
| 8 | Como Pro, clicar em "Histórico" na tela inicial | Lista de evoluções ordenadas por data |
| 9 | No histórico Pro, abrir uma evolução | SOAP renderizado sem erros |
| 10 | No histórico Pro, excluir uma evolução | Item removido da lista imediatamente |
| 11 | No Free, acessar /history | Mensagem "Recurso exclusivo do plano Pro" |
| 12 | Verificar tabela `evolutions` após geração | Nenhuma PII do paciente gravada; áudio não persistido |

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
│   │   ├── services/                   # transcription, soap_client, gemini_soap_client,
│   │   │                               #   groq_soap_client, rag, guardrails, quota, billing
│   │   ├── rag/                        # busca vetorial (pgvector)
│   │   └── core/                       # config, JWT, audit log, db
│   ├── migrations/                     # SQL (goose)
│   └── tests/guardrails/               # suíte de guardrails clínicos (obrigatória)
├── frontend/                           # React SPA
│   └── src/
│       ├── components/                 # Recorder, SoapResult, SoapEditor, CopyButton
│       ├── pages/                      # Auth, Home, History, Account
│       └── services/api.ts             # cliente HTTP tipado
└── Makefile                            # comandos de setup, execução e teste
```
