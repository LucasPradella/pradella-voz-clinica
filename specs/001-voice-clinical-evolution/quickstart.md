# Quickstart / Validation Guide: Evolução Clínica por Voz

Guia para subir o ambiente local e validar a feature ponta a ponta. Detalhes de implementação
ficam em `tasks.md`/implementação — aqui é setup + cenários de validação.

## Pré-requisitos

- Go 1.23, Node 20+, Docker (para PostgreSQL + pgvector local)
- Chaves: `ANTHROPIC_API_KEY` (Claude — estruturação SOAP), `OPENAI_API_KEY` (Whisper — STT),
  `STRIPE_SECRET_KEY` + `STRIPE_WEBHOOK_SECRET` (billing). Nunca versionar (Princípio I).

## Setup

```bash
# 1. Banco (PostgreSQL + pgvector)
docker run -d --name pradella-db -e POSTGRES_PASSWORD=dev -p 5432:5432 pgvector/pgvector:pg16

# 2. Backend (Go)
cd backend
go mod download
cp .env.example .env          # preencher chaves e DATABASE_URL
goose -dir migrations postgres "$DATABASE_URL" up   # cria o schema
go run ./cmd/ingest           # indexa as fontes clínicas (CREFITO/protocolos) no pgvector
go run ./cmd/api              # sobe a API

# 3. Frontend (PWA)
cd ../frontend
npm install
npm run dev                   # abre o PWA; usar HTTPS/localhost p/ permissão de microfone
```

## Cenários de validação (mapeados aos requisitos)

1. **Gravar → SOAP (US1, SC-002)**: na Home, 1 toque em "Gravar Evolução", fale o exemplo do
   PRD, encerre. Esperado: SOAP em S/O/A/P em ≤10s (p95) com terminologia correta.
2. **Copiar (FR-007)**: botão "Copiar" coloca o texto completo na área de transferência.
3. **Guardrail (SC-003)**: grave um áudio que NÃO cita um procedimento. Esperado: o SOAP não
   inclui procedimentos/diagnósticos ausentes. Cobertura automatizada em
   `backend/tests/guardrails/`.
4. **CID como sugestão (FR-005)**: áudio com quadro identificável → `cid_suggestions` presente,
   claramente marcado como sugestão e editável.
5. **Revisar/editar (US2, Pro)**: editar uma seção do SOAP e confirmar que a versão
   copiada/salva reflete a edição.
6. **Cota Free (US4, SC-006)**: gere 10 evoluções; a 11ª retorna 402 `quota_exceeded` com
   convite de upgrade. Vire o mês (ajuste `period`) e confirme reinício.
7. **Free é efêmero (FR-014a)**: no Free, sair e voltar → evolução não aparece em histórico.
8. **Histórico Pro (US3)**: como Pro, evoluções aparecem ordenadas por data e são reabríveis.
9. **LGPD (SC-008)**: confirme que nenhuma PII do paciente foi gravada (tabela `evolutions`) e
   que o áudio não foi persistido; verifique `AuditLog` de acesso.
10. **UX/identidade (FR-016)**: paleta navy/grafite/cinza, sem verde; estados loading/erro/sucesso.

## Testes

```bash
# Backend (inclui guardrails — obrigatórios, não puláveis)
cd backend && go test ./...

# Frontend
cd frontend && npm run test

# E2E
npx playwright test
```

CI deve rodar lint + todos os testes e bloquear o merge em qualquer falha (Constituição:
Padrões de Teste + Quality Gates).

## Referências

- Entidades: [data-model.md](./data-model.md)
- Contratos de API: [contracts/api.md](./contracts/api.md)
- Decisões técnicas: [research.md](./research.md)
