# Feature Specification: Evolução Clínica por Voz

**Feature Branch**: `001-voice-clinical-evolution`

**Created**: 2026-06-27

**Status**: Draft

**Input**: User description: "SaaS de Evolução Clínica por Voz (GenAI + RAG) — profissional grava um áudio curto entre atendimentos e o sistema transcreve, interpreta e devolve a evolução formatada em SOAP, pronta para copiar no prontuário."

## Clarifications

### Session 2026-06-27

- Q: Como tratar os dados de identificação do paciente contidos no áudio/evolução? → A: Não persistir PII do paciente — o sistema processa e exibe, mas não armazena nome/identificação do paciente; o histórico guarda apenas a evolução com rótulo livre/identificador interno definido pelo profissional.
- Q: O que fazer com a gravação de áudio bruta após gerar a evolução? → A: Descartar imediatamente — o áudio é usado só para transcrição e não é persistido após a geração.
- Q: Como identificar/validar o profissional no cadastro? → A: Cadastro mínimo só com e-mail/senha; não coleta nem valida registro de conselho no MVP. Identidade única = e-mail.
- Q: Qual o limite máximo de duração de uma gravação? → A: 120 segundos (2 minutos), com alvo de ~30s e aviso ao se aproximar do limite.
- Q: No plano Free, a evolução é armazenada após ser exibida/copiada? → A: Não — no Free a evolução é efêmera (só na sessão); histórico na nuvem é exclusivo do Pro.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Gravar e gerar evolução em SOAP (Priority: P1)

Um fisioterapeuta, entre um atendimento e outro, abre o app no celular, toca no botão
"Gravar Evolução", fala por até ~30 segundos o resumo da sessão e, em poucos segundos,
recebe na tela o texto estruturado no padrão SOAP (Subjetivo, Objetivo, Avaliação, Plano)
com terminologia técnica correta, pronto para copiar.

**Why this priority**: É o coração do produto e a única funcionalidade indispensável para
entregar valor. Sem ela, nada mais importa. Sozinha já constitui um MVP utilizável.

**Independent Test**: Pode ser testada de ponta a ponta gravando um áudio de exemplo e
verificando que o resultado retornado está corretamente segmentado em S/O/A/P, sem
informação inventada, e copiável com uma ação.

**Acceptance Scenarios**:

1. **Given** o profissional autenticado na tela inicial, **When** toca em "Gravar Evolução",
   fala um resumo da sessão e encerra a gravação, **Then** o sistema apresenta o texto
   formatado nas quatro seções do SOAP em poucos segundos.
2. **Given** uma evolução gerada na tela, **When** o profissional toca em "Copiar",
   **Then** o texto completo é copiado para a área de transferência para colar no prontuário
   externo.
3. **Given** um áudio que menciona um quadro com código CID aplicável e identificável,
   **When** o sistema processa o áudio, **Then** o código CID correspondente é sugerido
   junto à evolução, claramente marcado como sugestão.
4. **Given** um áudio que não menciona determinado procedimento ou diagnóstico,
   **When** o sistema gera o texto, **Then** o resultado NÃO inclui procedimentos ou
   diagnósticos ausentes do áudio original.

---

### User Story 2 - Revisar e editar antes de copiar (Priority: P1)

Antes de copiar, o profissional precisa revisar o texto gerado e poder ajustá-lo (corrigir
um termo, completar uma informação) com confiança de que o conteúdo reflete o que ele disse.

**Why this priority**: Responsabilidade clínica e legal exige que o profissional valide o
conteúdo. Sem revisão/edição, o produto não é seguro para uso real no prontuário.

**Independent Test**: Gerar uma evolução, editar manualmente um campo e confirmar que a
versão copiada/salva reflete a edição.

**Acceptance Scenarios**:

1. **Given** uma evolução gerada, **When** o profissional edita o texto de qualquer seção
   do SOAP, **Then** as alterações são preservadas para cópia e para o histórico.
2. **Given** uma evolução com código CID sugerido, **When** o profissional remove ou altera
   a sugestão, **Then** a versão final reflete a decisão do profissional.
3. **Given** uma transcrição com baixa confiança (áudio ruidoso/inaudível), **When** o
   resultado é exibido, **Then** o sistema sinaliza os trechos incertos para revisão.

---

### User Story 3 - Histórico de evoluções (Priority: P2)

O profissional (no plano pago) consulta evoluções geradas anteriormente para reutilizar,
comparar a progressão de um paciente ou recuperar um texto que não foi colado a tempo.

**Why this priority**: Agrega valor recorrente e sustenta o plano Pro, mas o MVP entrega
valor mesmo sem histórico persistido.

**Independent Test**: Gerar uma evolução, sair e voltar ao app, e confirmar que ela aparece
no histórico com data e identificação.

**Acceptance Scenarios**:

1. **Given** evoluções geradas anteriormente, **When** o profissional abre o histórico,
   **Then** vê a lista ordenada por data com identificação do paciente/sessão.
2. **Given** uma evolução no histórico, **When** o profissional a abre, **Then** pode
   visualizar e copiar o texto novamente.

---

### User Story 4 - Cadastro, autenticação e limite freemium (Priority: P2)

Um profissional novo cria uma conta, recebe acesso ao plano Free (10 evoluções/mês) e,
ao atingir o limite, é convidado a assinar o plano Pro para uso ilimitado.

**Why this priority**: Necessária para monetização e controle de uso, mas a experiência
central (US1) pode ser validada com contas de teste antes da cobrança estar pronta.

**Independent Test**: Criar conta Free, gerar 10 evoluções e confirmar que a 11ª é
bloqueada com convite claro para upgrade.

**Acceptance Scenarios**:

1. **Given** um visitante, **When** se cadastra e confirma a conta, **Then** recebe acesso
   ao plano Free com cota de 10 evoluções no mês corrente.
2. **Given** um usuário Free que usou 10 evoluções no mês, **When** tenta gerar a 11ª,
   **Then** o sistema bloqueia a geração e apresenta o convite de upgrade para o Pro.
3. **Given** um usuário Pro ativo, **When** gera evoluções, **Then** não há limite mensal
   de quantidade.
4. **Given** o início de um novo mês, **When** o ciclo de cota reinicia, **Then** a contagem
   do usuário Free volta a zero.

---

### Edge Cases

- **Áudio vazio ou muito curto**: o sistema informa que não há conteúdo suficiente e não
  consome cota.
- **Áudio inaudível/ruidoso**: trechos incertos são sinalizados; o profissional pode regravar.
- **Áudio acima da duração máxima** (alvo ~30s, limite de 120s): ao se aproximar do limite o
  sistema avisa; ao atingir 120s a gravação é encerrada automaticamente e o profissional pode
  gerar a evolução do que foi captado ou regravar.
- **Perda de conexão durante o processamento**: o sistema preserva o áudio e permite
  retomar/reenviar sem perda e sem cobrança dupla de cota.
- **Conteúdo fora de escopo clínico** (fala não relacionada a uma evolução): o sistema
  estrutura o que for possível e sinaliza baixa confiança, sem inventar dados clínicos.
- **Idioma/sotaque regional e jargão**: a terminologia técnica deve ser preservada/corrigida
  sem alterar o sentido do que foi dito.
- **Permissão de microfone negada**: o app orienta como habilitar antes de gravar.
- **Cota esgotada no meio de uma gravação**: a regra de quando a cota é debitada deve ser
  consistente (ver Assumptions).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O sistema MUST permitir gravar um áudio de voz a partir de uma única ação
  primária ("Gravar Evolução") na tela inicial, com alvo de ~30s e limite máximo de 120s
  (encerramento automático ao atingir o limite, com aviso prévio ao se aproximar).
- **FR-002**: O sistema MUST transcrever o áudio em texto e gerar uma evolução clínica
  estruturada nas quatro seções do padrão SOAP (Subjetivo, Objetivo, Avaliação, Plano).
- **FR-003**: O sistema MUST corrigir/normalizar a terminologia técnica de saúde com base
  em fontes clínicas controladas, sem alterar o sentido do que foi dito.
- **FR-004**: O sistema MUST NOT incluir diagnósticos, procedimentos, medicações ou códigos
  que não tenham sido mencionados no áudio original (guardrail clínico).
- **FR-005**: O sistema MUST sugerir códigos CID quando aplicável e identificável, sempre
  marcando-os explicitamente como sugestão sujeita a validação do profissional.
- **FR-006**: O sistema MUST permitir que o profissional edite qualquer parte da evolução
  gerada antes de copiar ou salvar.
- **FR-007**: O sistema MUST oferecer uma ação única de "Copiar" que coloca o texto completo
  da evolução na área de transferência.
- **FR-008**: O sistema MUST sinalizar trechos de baixa confiança da transcrição para
  revisão do profissional.
- **FR-009**: O sistema MUST fornecer feedback explícito dos estados de gravação,
  processamento, sucesso e erro, sem deixar o usuário sem retorno.
- **FR-010**: O sistema MUST permitir cadastro e autenticação de profissionais usando apenas
  e-mail/senha como identidade única (e-mail). O MVP NÃO coleta nem valida número de registro
  de conselho (CREFITO/CRM).
- **FR-011**: O sistema MUST aplicar a cota do plano Free de 10 evoluções por mês e bloquear
  a geração além da cota, apresentando convite de upgrade.
- **FR-012**: O sistema MUST oferecer um plano Pro com geração ilimitada de evoluções e
  armazenamento de histórico na nuvem.
- **FR-013**: O sistema MUST reiniciar a contagem de cota do plano Free a cada novo ciclo
  mensal.
- **FR-014**: O sistema MUST armazenar o histórico de evoluções para usuários do plano Pro,
  permitindo consulta, visualização e nova cópia.
- **FR-014a**: No plano Free, a evolução gerada MUST ser efêmera — disponível apenas durante
  a sessão/tela atual e NÃO persistida após o usuário sair. O histórico na nuvem é exclusivo
  do plano Pro.
- **FR-015**: O sistema MUST funcionar como aplicação web instalável (PWA) em navegador
  móvel, sem exigir instalação por loja de aplicativos na primeira experiência.
- **FR-016**: A interface MUST seguir a identidade visual definida (azul profundo/navy,
  grafite e cinza claro), sem usar verde como cor de marca.
- **FR-017**: O sistema MUST proteger dados pessoais e de saúde conforme a LGPD, incluindo
  registro de acesso, base legal de tratamento e política de retenção/descarte de áudios e
  transcrições.
- **FR-017a**: O sistema MUST NOT armazenar dados de identificação do paciente (ex.: nome,
  idade) extraídos do áudio. A evolução persistida no histórico deve usar apenas um rótulo
  livre/identificador interno definido pelo profissional (sem PII do paciente). Quando exibir
  a evolução em tela, eventuais menções a PII na transcrição não devem ser gravadas no
  histórico.
- **FR-017b**: O sistema MUST descartar a gravação de áudio bruta imediatamente após a
  geração da evolução; o áudio NÃO é persistido. Em caso de falha de geração, o áudio pode
  permanecer apenas em memória transitória para reprocessamento na mesma sessão e não deve
  ser gravado em armazenamento durável.
- **FR-018**: O sistema MUST não debitar cota quando a geração falhar por erro do sistema ou
  por áudio insuficiente.
- **FR-019**: O sistema MUST registrar, por evolução, a origem das fontes clínicas usadas
  para normalização terminológica, de modo rastreável.
- **FR-020**: O modelo de dados e a arquitetura MUST permitir, em fase futura, a evolução para
  licenciamento B2B (pacotes de licenças por clínica), sem exigir reescrita do núcleo. O
  gerenciamento de licenças B2B em si está **fora do escopo do MVP** (ver Out of Scope).

### Key Entities *(include if feature involves data)*

- **Profissional (Usuário)**: pessoa que usa o app; identidade única por e-mail; atributos
  relevantes: e-mail, credencial de acesso, plano (Free/Pro), cota usada no ciclo. (Registro
  de conselho profissional NÃO é coletado no MVP.)
- **Evolução Clínica**: o registro gerado; atributos: data/hora, rótulo livre/identificador
  interno definido pelo profissional (sem PII do paciente), texto estruturado em SOAP
  (S/O/A/P), códigos CID sugeridos, sinalizações de confiança, status (rascunho/finalizada),
  referência às fontes clínicas usadas. NÃO armazena nome/identificação do paciente.
- **Áudio de Sessão**: a gravação de origem, **transitória** (não persistida); atributos em
  memória durante o processamento: duração, qualidade/confiança. Descartada imediatamente
  após a geração da evolução.
- **Assinatura/Plano**: vínculo do profissional ao plano; atributos: tipo (Free/Pro),
  status, ciclo de cobrança, cota mensal.
- **Fonte Clínica (base de conhecimento)**: diretrizes e protocolos (ex.: CREFITO) usados
  para normalizar terminologia; atributos: origem, versão, data de atualização.

> Nota: a entidade *Licença de Clínica (B2B)* é prevista para fase futura e não faz parte
> do modelo do MVP (ver Out of Scope).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A partir da tela inicial, o profissional inicia uma gravação em no máximo 1
  toque/ação.
- **SC-002**: Em pelo menos 95% das gravações de até 30 segundos, o profissional recebe a
  evolução formatada em até 10 segundos após encerrar a gravação.
- **SC-003**: Em 0% dos casos a evolução final inclui diagnóstico ou procedimento não
  presente no áudio original, verificado por conjunto de testes de guardrail.
- **SC-004**: Pelo menos 90% dos profissionais conseguem gerar e copiar a primeira evolução
  com sucesso na primeira tentativa, sem ajuda externa.
- **SC-005**: O tempo médio para registrar uma evolução (gravar + revisar + copiar) é
  reduzido em pelo menos 60% frente à digitação manual.
- **SC-006**: A cota do plano Free é aplicada corretamente em 100% dos casos (11ª evolução
  bloqueada; reinício no novo ciclo).
- **SC-007**: A segmentação SOAP é considerada correta pelo profissional em pelo menos 90%
  das evoluções geradas (sem necessidade de reestruturar seções).
- **SC-008**: 100% dos áudios e transcrições seguem a política de retenção/descarte definida
  e possuem registro de acesso.

## Assumptions

- A duração-alvo de áudio é ~30s, com limite máximo de 120s (encerramento automático e aviso
  ao se aproximar do limite).
- A cota de uma evolução é debitada apenas quando uma geração é concluída com sucesso; falhas
  de sistema e áudios insuficientes não consomem cota.
- O ciclo de cota do plano Free é mensal por calendário, reiniciando no primeiro dia do mês.
- O idioma primário das gravações e da interface é o português do Brasil.
- A geração de códigos CID é sempre uma sugestão a ser validada pelo profissional, nunca uma
  atribuição automática definitiva.
- O profissional é o responsável legal pelo conteúdo final colado no prontuário; o produto é
  ferramenta de apoio à estruturação, não fonte de informação clínica.
- O preço de referência do plano Pro é R$ 49,90/mês (sujeito a ajuste comercial).
- A cópia para o prontuário externo é feita via área de transferência; não há integração
  direta com sistemas legados do SUS/clínicas no MVP.
- As fontes clínicas (diretrizes do CREFITO e protocolos) estarão disponíveis e licenciadas
  para uso na base de conhecimento.

## Out of Scope (MVP)

- Licenciamento e gerenciamento B2B para clínicas (compra de pacotes de licenças, painel
  administrativo da clínica, gestão de equipe, faturamento agregado e relatórios). O MVP é
  100% B2C (profissional autônomo, planos Free e Pro individuais). A arquitetura deve apenas
  não impedir essa evolução futura (ver FR-020).
- Integração direta com sistemas legados de prontuário do SUS ou de clínicas (no MVP, a
  transferência é feita por cópia para a área de transferência).
- Aplicativos nativos publicados em lojas (App Store / Google Play); o MVP é um PWA.

## Dependencies

- Disponibilidade de uma base de conhecimento clínica controlada (diretrizes do CREFITO e
  protocolos) para normalização terminológica.
- Capacidade de transcrição de voz para texto em português.
- Meio de pagamento/assinatura para o plano Pro e, eventualmente, faturamento B2B.
- Conformidade com a LGPD para tratamento de dados de saúde.
