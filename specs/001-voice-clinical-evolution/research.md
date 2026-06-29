# Phase 0 Research: Evolução Clínica por Voz

Consolidação das decisões técnicas. Formato por item: **Decisão / Racional / Alternativas
consideradas**.

## 1. Modelo de estruturação SOAP (LLM)

- **Decisão**: Claude **Sonnet 4.6** (`claude-sonnet-4-6`) como padrão, com Haiku 4.5
  (`claude-haiku-4-5`) como opção de menor custo e Opus 4.8 (`claude-opus-4-8`) como fallback
  de qualidade. Saída via **structured outputs** (`output_config.format`) para o JSON SOAP.
- **Racional**: tarefa é extração/estruturação ancorada em RAG com guardrails rígidos — não
  exige o topo de raciocínio do Opus. Sonnet 4.6 ($3/$15 por 1M tokens, contexto 1M) entrega
  forte aderência a instruções e structured outputs a 1/5 do custo de saída do Opus, viável
  para freemium de alto volume. Haiku ($1/$5) cobre o tier Free se o custo apertar.
- **Alternativas**: Opus 4.8 (mais caro, ganho marginal para esta tarefa); Haiku 4.5 como
  padrão único (mais barato, mas menor robustez em terminologia clínica/guardrails).

## 2. Transcrição (STT)

- **Decisão**: OpenAI **Whisper API** (`whisper-1`) para áudio→texto em pt-BR.
- **Racional**: definido no PRD; rápido, barato e robusto para fala em português. É apenas STT
  (sem geração), separado do motor generativo Claude.
- **Alternativas**: transcrição on-device (latência/qualidade móvel inconsistente); outros STT
  gerenciados — reavaliar se custo/precisão exigir.

## 3. Arquitetura RAG e banco vetorial

- **Decisão**: `pgvector` na **mesma instância Amazon RDS PostgreSQL** que guarda usuários,
  histórico e assinaturas, acessado em Go via `pgx` + `pgvector-go`. Índice das fontes clínicas
  (CREFITO/protocolos) versionadas.
- **Racional**: Princípio I (simplicidade/YAGNI) e custo controlado no MVP — evita operar um
  banco vetorial dedicado. Recuperação por similaridade fornece o contexto que ancora a
  terminologia; a origem das fontes é registrada por evolução (FR-019).
- **Alternativas**: banco vetorial dedicado (Pinecone/OpenSearch) — complexidade e custo extra
  não justificados no MVP.

## 4. Guardrails clínicos (anti-alucinação)

- **Decisão**: (a) system prompt que restringe a IA a *estruturar/refinar* o que está na
  transcrição; (b) structured outputs com schema SOAP + campo `confidence` e `cid_suggestions`
  marcados como sugestão; (c) verificação pós-geração de que cada procedimento/diagnóstico no
  SOAP tem âncora na transcrição; (d) suíte de testes obrigatória de casos áudio→SOAP.
- **Racional**: SC-003 exige 0% de invenção; o Princípio II torna esses testes não-ignoráveis.
- **Alternativas**: confiar só no prompt (insuficiente); LLM-as-judge adicional (custo; manter
  como reforço futuro, não bloqueante no MVP).

## 5. Prompt caching e custo

- **Decisão**: aplicar `cache_control` ao prefixo estável (system prompt + instruções de
  guardrail + trechos RAG recuperados), deixando a transcrição variável após o último
  breakpoint. Monitorar `cache_read_input_tokens` e custo por evolução.
- **Racional**: o prefixo (guardrails + contexto clínico) repete a cada requisição; caching
  reduz ~90% do custo desse trecho (Princípio IV: custo previsível para o freemium).
- **Alternativas**: sem cache (custo desnecessário em alto volume).

## 6. Captura de áudio no PWA

- **Decisão**: MediaRecorder API no navegador; alvo ~30s, limite rígido de 120s com
  encerramento automático e aviso ao se aproximar; upload do blob ao backend; áudio nunca
  persistido em armazenamento durável.
- **Racional**: FR-001/FR-017b; PWA sem instalação de loja (FR-015). Tratamento explícito de
  permissão de microfone negada (edge case).
- **Alternativas**: app nativo (fora do escopo do MVP).

## 7. Autenticação e cota freemium

- **Decisão**: e-mail/senha (hash bcrypt) + JWT de sessão; identidade única por e-mail (sem
  conselho no MVP). Contagem de cota por ciclo mensal de calendário; débito só em geração
  bem-sucedida.
- **Racional**: decisões registradas nas Clarifications da spec (FR-010, FR-011, FR-013, FR-018).
- **Alternativas**: OAuth social (adicionar depois); validação de conselho (fase futura).

## 8. Pagamentos (Pro)

- **Decisão**: Stripe Checkout + Billing; webhook para ativar/desativar o plano Pro e liberar
  histórico na nuvem. Preço de referência R$ 49,90/mês.
- **Racional**: padrão de mercado, rápido de integrar; histórico Pro depende do status da
  assinatura (FR-012/FR-014).
- **Alternativas**: gateways locais (avaliar depois por taxa/AbacatePay/etc.).

## 9. Infraestrutura

- **Decisão**: API conteinerizada no AWS Fargate atrás de um ALB; RDS PostgreSQL gerenciado;
  segredos no AWS Secrets Manager; IaC em `infra/`. Escala horizontal para o pico do fim de tarde.
- **Racional**: alinhado ao PRD e ao Princípio IV (escala horizontal, custo controlado).
- **Alternativas**: servidores fixos (não elásticos); Lambda (limites de tempo/cold start para
  o pipeline de mídia).

## 10. Linguagem do backend

- **Decisão**: **Go 1.23** com `chi` (HTTP), `pgx`+`pgvector-go` (DB), `goose` (migrations),
  `anthropic-sdk-go` (Claude), `openai-go` (Whisper STT), `stripe-go`, `golang-jwt`, `bcrypt`.
- **Racional**: escolha do usuário. Binário único e enxuto, baixo uso de memória e cold start
  rápido — bom para o container no Fargate e para absorver o pico do fim de tarde com custo
  controlado (Princípio IV). Concorrência nativa (goroutines) ajuda no fan-out
  transcrição→RAG→LLM. SDKs oficiais de Claude, OpenAI e Stripe disponíveis em Go.
- **Alternativas**: Python/FastAPI (ecossistema de IA mais maduro, porém maior footprint de
  runtime) — preterido a pedido do usuário; Node/TypeScript (unificaria a linguagem com o
  frontend, mas Go foi a preferência).

## Itens resolvidos

Nenhum marcador `NEEDS CLARIFICATION` remanescente — as ambiguidades de produto foram
resolvidas em `/speckit-clarify`; as escolhas de stack acima fecham os pontos técnicos abertos.
