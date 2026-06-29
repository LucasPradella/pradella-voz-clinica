<!--
SYNC IMPACT REPORT
==================
Versão: TEMPLATE (não versionado) → 1.0.0
Tipo de bump: Ratificação inicial (MAJOR — primeira definição concreta dos princípios)

Princípios definidos:
  - I. Qualidade de Código (novo)
  - II. Padrões de Teste (novo)
  - III. Consistência da Experiência do Usuário (novo)
  - IV. Requisitos de Performance (novo)

Seções adicionadas:
  - Segurança de IA e Proteção de Dados (Seção 2)
  - Fluxo de Desenvolvimento e Quality Gates (Seção 3)
  - Governança

Seções removidas: nenhuma (todos os placeholders do template foram preenchidos)

Templates dependentes:
  - .specify/templates/plan-template.md ✅ compatível (gate "Constitution Check" é genérico e
    referencia este arquivo; nenhuma alteração necessária)
  - .specify/templates/spec-template.md ✅ compatível (Success Criteria mensuráveis alinham com
    os Princípios III e IV)
  - .specify/templates/tasks-template.md ✅ compatível (tarefas de teste são opcionais por padrão;
    o Princípio II reforça as exigências quando o domínio clínico estiver envolvido)
  - .specify/templates/checklist-template.md ✅ compatível
  - .specify/templates/commands/ ⚠ inexistente (nenhum arquivo de comando a sincronizar)

TODOs pendentes: nenhum.
-->

# Pradella Voz Clínica Constitution

## Core Principles

### I. Qualidade de Código

O código DEVE ser legível, coeso e revisável por outra pessoa sem explicação verbal.
Regras não-negociáveis:

- Toda mudança DEVE passar por revisão de ao menos um outro desenvolvedor antes do merge;
  ninguém aprova o próprio código.
- Linters e formatadores automáticos DEVEM rodar no CI e bloquear o merge em caso de falha;
  formatação não é assunto de revisão humana.
- Funções e módulos DEVEM ter responsabilidade única; lógica de domínio clínico, integração
  com IA e camada de apresentação NÃO DEVEM se misturar no mesmo componente.
- Segredos, chaves de API e credenciais NUNCA DEVEM ser versionados; configuração sensível
  DEVE vir de variáveis de ambiente ou cofre gerenciado.
- Complexidade adicional (novo serviço, nova dependência, novo padrão) DEVE ser justificada
  por escrito; na ausência de justificativa, vale a opção mais simples (YAGNI).

**Racional**: O sistema lida com dados clínicos e com saída gerada por IA. Código obscuro
esconde erros que, neste domínio, podem chegar ao prontuário do paciente. Clareza e revisão
são a primeira linha de defesa.

### II. Padrões de Teste

Todo comportamento de que o usuário ou outro sistema depende DEVE ser coberto por teste
automatizado. Regras não-negociáveis:

- Lógica de domínio clínico, formatação SOAP e os guardrails de IA DEVEM ter testes
  automatizados antes de serem considerados concluídos (test-first para essas áreas).
- A suíte de testes DEVE rodar no CI e o merge DEVE ser bloqueado se qualquer teste falhar.
- Toda correção de bug DEVE incluir um teste de regressão que falha antes da correção e
  passa depois.
- Contratos de API e integrações externas (transcrição, banco vetorial RAG, RDS) DEVEM ter
  testes de contrato/integração; mudanças de contrato DEVEM atualizar esses testes no mesmo PR.
- Os guardrails DEVEM ter testes que comprovem que a IA não inventa diagnóstico, procedimento
  ou código CID ausente no áudio original. Esses testes são obrigatórios e NÃO podem ser
  marcados como ignorados.

**Racional**: A confiabilidade do produto é o seu diferencial. Sem testes que provem que a IA
apenas estrutura — e nunca cria — informação médica, o produto perde a confiança do
profissional de saúde e cria risco clínico real.

### III. Consistência da Experiência do Usuário

A interface DEVE priorizar atrito zero para um profissional cansado e com pressa. Regras
não-negociáveis:

- A identidade visual DEVE seguir a paleta definida (azul profundo/navy, grafite e cinza claro)
  e NÃO DEVE introduzir verde como cor de marca; tokens de design DEVEM ser centralizados, não
  redefinidos por tela.
- A ação primária ("Gravar Evolução") DEVE ser alcançável em no máximo um toque a partir da
  tela inicial.
- Estados de carregamento, erro e sucesso DEVEM ser explícitos e consistentes em todas as telas;
  o usuário nunca DEVE ficar sem feedback durante o processamento do áudio.
- A saída em padrão SOAP DEVE ser sempre copiável com uma única ação de "Copiar".
- O produto DEVE funcionar como PWA em navegador móvel sem exigir instalação na primeira
  experiência; quebras de acessibilidade básica (contraste, alvos de toque, foco de teclado)
  DEVEM ser tratadas como bug.

**Racional**: O valor do produto é economizar o tempo do profissional entre atendimentos.
Qualquer inconsistência ou passo extra na jornada destrói diretamente a proposta de valor.

### IV. Requisitos de Performance

O sistema DEVE ser rápido nos momentos de uso real, com custo de infraestrutura controlado.
Regras não-negociáveis:

- O pipeline de áudio→texto→SOAP DEVE devolver o resultado formatado em até 10 segundos (p95)
  para um áudio de até 30 segundos; metas de performance que mudem DEVEM ser registradas no
  plano da feature.
- Toda feature DEVE declarar suas metas de performance e limites de recurso no `plan.md`
  (latência alvo, throughput, memória) — "NEEDS CLARIFICATION" não é aceitável no merge.
- A arquitetura DEVE suportar escalonamento horizontal para absorver picos de uso no fim da
  tarde sem degradação perceptível para o usuário.
- Regressões de performance medidas DEVEM bloquear o merge até serem justificadas ou corrigidas.
- Custo por evolução processada DEVE ser monitorado; otimizações NÃO DEVEM sacrificar os
  guardrails ou a qualidade clínica da saída.

**Racional**: O profissional usa o app em segundos entre pacientes; lentidão anula o ganho de
produtividade. Ao mesmo tempo, o modelo freemium só é viável se o custo por requisição for
previsível e baixo.

## Segurança de IA e Proteção de Dados

Esta seção é não-negociável por se tratar de dados clínicos sob a LGPD.

- A IA SOMENTE estrutura e refina o conteúdo do áudio; NUNCA DEVE criar diagnóstico,
  procedimento, medicação ou código CID que não tenha sido mencionado pelo profissional.
- O motor RAG DEVE basear a terminologia técnica em fontes controladas (diretrizes do CREFITO,
  protocolos clínicos versionados); a origem dessas fontes DEVE ser rastreável.
- Dados pessoais e de saúde DEVEM ser tratados conforme a LGPD: criptografia em trânsito e em
  repouso, minimização de dados e registro de acesso (audit log).
- Áudios e transcrições DEVEM ter política explícita de retenção e descarte; nada é retido sem
  base legal e finalidade declarada.
- Qualquer envio de dados a serviços externos (transcrição, modelos de linguagem) DEVE ser
  documentado e coberto por contrato/termos compatíveis com o tratamento de dados de saúde.

## Fluxo de Desenvolvimento e Quality Gates

- Toda feature segue o fluxo Spec Kit: especificação → plano → tarefas → implementação,
  com o "Constitution Check" do `plan-template.md` validado antes do início do desenvolvimento.
- Nenhum PR é mergeado sem: revisão aprovada, CI verde (lint + testes) e metas de performance
  declaradas quando aplicável.
- Violações dos guardrails de IA ou da consistência de UX são tratadas como bloqueadoras de
  release, não como dívida técnica negociável.
- Decisões de arquitetura que adicionam complexidade DEVEM ser registradas na seção
  "Complexity Tracking" do plano da feature.

## Governance

Esta constituição prevalece sobre quaisquer outras práticas e convenções do projeto.

- Emendas DEVEM ser propostas por escrito (PR alterando este arquivo), descrevendo a mudança,
  o racional e o impacto nos templates dependentes.
- O versionamento desta constituição segue SemVer:
  - MAJOR: remoção ou redefinição incompatível de princípios ou regras de governança.
  - MINOR: adição de princípio/seção ou expansão material de orientação.
  - PATCH: esclarecimentos, correções de redação e refinamentos não-semânticos.
- Toda revisão de PR DEVE verificar conformidade com os princípios aqui definidos; desvios DEVEM
  ser justificados explicitamente ou corrigidos antes do merge.
- A conformidade DEVE ser reavaliada a cada release significativa e sempre que um princípio for
  emendado.

**Version**: 1.0.0 | **Ratified**: 2026-06-27 | **Last Amended**: 2026-06-27
