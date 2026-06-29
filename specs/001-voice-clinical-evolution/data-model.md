# Phase 1 Data Model: Evolução Clínica por Voz

Entidades persistidas no Amazon RDS PostgreSQL. Regras de validação derivadas dos requisitos
(FR-*) e das Clarifications. **Nenhuma PII do paciente é persistida** (FR-017a); o áudio é
transitório e nunca gravado (FR-017b).

## Profissional (User)

Pessoa que usa o app. Identidade única por e-mail (FR-010).

| Campo | Tipo | Regras |
|---|---|---|
| id | UUID (PK) | gerado |
| email | string | único, obrigatório, formato e-mail |
| password_hash | string | bcrypt; nunca exposto |
| plan | enum(`free`,`pro`) | default `free` |
| created_at | timestamp | |
| updated_at | timestamp | |

Notas: registro de conselho (CREFITO/CRM) **não** é coletado no MVP.

## Assinatura (Subscription)

Vínculo do profissional ao plano e ao ciclo de cobrança (FR-012).

| Campo | Tipo | Regras |
|---|---|---|
| id | UUID (PK) | |
| user_id | UUID (FK→User) | obrigatório |
| type | enum(`free`,`pro`) | |
| status | enum(`active`,`canceled`,`past_due`) | |
| stripe_customer_id | string | nullable (Pro) |
| stripe_subscription_id | string | nullable (Pro) |
| current_period_end | timestamp | nullable |

## CotaDeUso (UsageQuota)

Controle da cota mensal do Free (FR-011, FR-013). Débito só em sucesso (FR-018).

| Campo | Tipo | Regras |
|---|---|---|
| id | UUID (PK) | |
| user_id | UUID (FK→User) | obrigatório |
| period | string (`YYYY-MM`) | ciclo de calendário; único por user+period |
| count | int | ≥0; Free bloqueia geração ao atingir 10 |

Regra: usuário Pro não tem limite de quantidade. Reinício automático no novo `period`.

## EvoluçãoClínica (Evolution)

Registro gerado. Persistida apenas no plano **Pro** (FR-014); no **Free** é efêmera e não
gravada (FR-014a). **Sem PII do paciente** (FR-017a).

| Campo | Tipo | Regras |
|---|---|---|
| id | UUID (PK) | |
| user_id | UUID (FK→User) | obrigatório |
| label | string | rótulo livre/identificador interno definido pelo profissional (sem PII) |
| soap_s | text | Subjetivo |
| soap_o | text | Objetivo |
| soap_a | text | Avaliação |
| soap_p | text | Plano |
| cid_suggestions | jsonb | lista de sugestões `{code, description}` (marcadas como sugestão) |
| confidence_flags | jsonb | trechos de baixa confiança sinalizados (FR-008) |
| status | enum(`draft`,`finalized`) | |
| source_refs | jsonb | referências às fontes clínicas usadas (FR-019) |
| created_at | timestamp | |

Transições de estado: `draft` → (edição do profissional) → `finalized`.

## Áudio de Sessão (transitório — NÃO persistido)

Existe apenas em memória durante o processamento. Atributos em runtime: duração (≤120s),
qualidade/confiança. Descartado imediatamente após a geração (FR-017b). Não há tabela.

## FonteClínica (ClinicalSource) — base de conhecimento RAG

Diretrizes do CREFITO e protocolos usados para normalizar terminologia (FR-003, FR-019).

| Campo | Tipo | Regras |
|---|---|---|
| id | UUID (PK) | |
| title | string | |
| origin | string | ex.: "CREFITO", protocolo X |
| version | string | versionada (rastreabilidade) |
| chunk_text | text | trecho indexado |
| embedding | vector (pgvector) | dimensão do modelo de embedding |
| updated_at | timestamp | |

## AuditLog

Registro de acesso a dados pessoais/de saúde (FR-017, LGPD).

| Campo | Tipo | Regras |
|---|---|---|
| id | UUID (PK) | |
| user_id | UUID (FK→User) | nullable em eventos de sistema |
| action | string | ex.: `evolution.create`, `evolution.view`, `login` |
| resource_id | UUID | nullable |
| created_at | timestamp | |
| metadata | jsonb | sem PII do paciente |

## Licença de Clínica (B2B) — FORA do MVP

Prevista para fase futura (ver spec → Out of Scope). A modelagem acima não impede a evolução
(FR-020): um futuro `ClinicLicense` poderia agrupar vários `User` sem reescrever o núcleo.
