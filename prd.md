# **Plano Estratégico e Especificação de Produto (PRD): Plataforma PWA de Ensino de Inglês para Profissionais de Tecnologia**

A concepção de um aplicativo educacional voltado para o ensino de inglês para falantes de português do Brasil (pt-br), estruturado sob a arquitetura de *Progressive Web App* (PWA) e direcionado a profissionais de tecnologia e viajantes, representa uma iniciativa de alto valor estratégico. A digitalização acelerada e o modelo de trabalho remoto global ampliaram drasticamente a demanda por fluência em inglês técnico. No entanto, o desenvolvimento de uma plataforma que unifique reconhecimento de voz sofisticado, testes adaptativos precisos e uma experiência de usuário impecável em navegadores móveis exige um rigoroso planejamento arquitetônico e pedagógico.  
Avaliando a ideia inicial sob a ótica de engenharia de produto e análise de negócios, identifica-se que a proposta é sólida, mas apresenta vulnerabilidades técnicas críticas em sua execução web, especialmente no ecossistema iOS. A dependência de APIs nativas de reconhecimento de voz em ambientes PWA frequentemente resulta em falhas de permissão e degradação da experiência do usuário. Consequentemente, este documento propõe refinamentos essenciais à ideia original, substituindo dependências frágeis por soluções híbridas de Inteligência Artificial e estabelecendo um currículo fundamentado na ciência da aquisição de segunda língua. As seções a seguir detalham as fundações teóricas, a arquitetura tecnológica e culminam no Documento de Requisitos de Produto (PRD) estruturado para orientar as equipes de desenvolvimento.

## **Análise de Viabilidade Técnica e Otimização da Proposta Inicial**

A escolha do modelo PWA oferece vantagens operacionais inegáveis, incluindo a capacidade de instalação direta pelo navegador (contornando as taxas e a fricção das lojas de aplicativos tradicionais) e o suporte a funcionalidades offline através de *Service Workers*1. Contudo, a premissa de capturar e avaliar a fala do usuário através do navegador impõe o maior desafio técnico do projeto.  
O *Web Speech API*, padrão nativo dos navegadores para reconhecimento de voz, apresenta suporte inconsistente no mercado. Embora navegadores baseados em Chromium (como Chrome e Edge) ofereçam suporte integral enviando o áudio para os servidores do Google, o Safari da Apple impõe barreiras severas3. Análises técnicas revelam que o Safari em dispositivos móveis (iOS) frequentemente bloqueia o *Speech Recognition API* quando o aplicativo web é instalado na tela inicial (modo *Standalone*), falhando ao solicitar as permissões adequadas de microfone4. Além disso, depender de motores nativos significa que a qualidade da transcrição variará drasticamente dependendo do dispositivo do usuário, impossibilitando uma avaliação padronizada da pronúncia de um falante brasileiro.  
Para mitigar este risco e elevar a qualidade do produto, a arquitetura deve abandonar o *Web Speech API* em favor de um motor de processamento de linguagem natural determinístico e universal. A solução recomendada é a utilização da tecnologia Whisper, desenvolvida pela OpenAI6. Ao invés de delegar o reconhecimento ao navegador, o aplicativo utilizará a API padrão de mídia (MediaDevices.getUserMedia()) para capturar o áudio bruto, que será então processado por modelos Whisper. Este processamento pode ocorrer de forma híbrida: através de chamadas de API em nuvem de baixíssimo custo para a maioria dos usuários móveis, ou através de execução local diretamente no navegador utilizando *WebAssembly* (WASM) e *WebGPU* para usuários em desktops potentes, garantindo privacidade total e custo zero de servidor7.  
Outra consideração vital para a arquitetura PWA é a política restritiva da Apple quanto ao armazenamento. O Safari impõe um limite de 50MB para o cache local e, de forma mais crítica, aplica uma política de expiração de 7 dias10. Se um usuário não abrir o aplicativo por uma semana, todos os arquivos cacheados e dados de lições offline são purgados automaticamente pelo sistema operacional10. O aplicativo deve ser projetado com um modelo de dados resiliente, garantindo que o progresso do usuário seja sincronizado de forma otimista com um banco de dados em nuvem, minimizando o impacto dessa limpeza agressiva de cache.

## **Diretrizes Pedagógicas e Aquisição Rápida de Idiomas**

A eficácia de uma plataforma de idiomas baseia-se na aplicação de metodologias cientificamente validadas para a cognição adulta. A andragogia demonstra que adultos, especialmente profissionais de nível técnico, não retêm informações eficientemente através de traduções gramaticais descontextualizadas11. O aplicativo deve, portanto, estruturar-se sobre o *Task-Based Language Teaching* (Ensino Baseado em Tarefas \- TBLT) e o *Communicative Language Teaching* (CLT). Estas metodologias priorizam a fluência comunicativa e a resolução de problemas reais em detrimento da precisão sintática isolada11.  
O currículo será desenhado para simular interações autênticas. Em vez de memorizar listas de verbos, o desenvolvedor praticará a articulação de conceitos lógicos, a explicação de falhas em sistemas e a participação em cerimônias ágeis. O sistema de ensino incorporará também a Prática de Recuperação Espaçada (*Spaced Retrieval Practice*). Pesquisas em psicologia cognitiva provam que o cérebro consolida memórias de longo prazo de forma mais eficaz quando é forçado a recuperar uma informação no momento exato em que está prestes a esquecê-la13. O aplicativo utilizará algoritmos de repetição espaçada para rotacionar vocabulários técnicos e estruturais, garantindo que termos aprendidos no primeiro módulo sejam revisitados estrategicamente semanas depois.  
A tabela a seguir delineia a matriz de conteúdo proposta, cruzando os níveis do Quadro Europeu Comum de Referência para as Línguas (CEFR) com as necessidades dos dois perfis de usuários identificados: o viajante em busca de sobrevivência e o engenheiro de software.

| Nível CEFR | Foco de Sobrevivência e Viagem (Travel) | Foco em Engenharia de Software (Tech) |
| :---- | :---- | :---- |
| **Básico (A1-A2)** | Navegação em aeroportos, alfândega, solicitação de informações e pedidos em restaurantes. | Vocabulário fundacional de TI (*Bug, Array, Function, Syntax, Install*) e leitura de instruções simples15. |
| **Intermediário (B1-B2)** | Relato de problemas em hotéis, interações sociais fluidas, aluguel de veículos e emergências. | Participação em *Daily Scrums*, discussão de *features*, explicação de atualizações de código e resolução de conflitos no GitHub18. |
| **Avançado (C1-C2)** | Compreensão de nuances culturais, jargões, ironia e negociações de serviços complexos. | Arquitetura de sistemas, liderança técnica, condução de entrevistas e comunicação executiva global18. |

## **Dinâmica do Teste de Nivelamento Adaptativo**

Para determinar com precisão o ponto de partida do usuário, o aplicativo implementará um Teste de Nivelamento Adaptativo. Historicamente, testes de formato fixo (*Fixed Form*) apresentam desvantagens severas: eles são excessivamente longos, frustram iniciantes com questões complexas e entediam alunos avançados com questões triviais21.  
A plataforma adotará o modelo de Teste Adaptativo Computadorizado em Múltiplos Estágios (*Computer Adaptive Multi-Stage Test* \- CAMST). Nesta arquitetura, o algoritmo analisa a resposta do usuário em tempo real21. Todos os usuários iniciam com um agrupamento de questões de dificuldade média. Se a taxa de acerto for elevada, o sistema recalcula a trajetória e apresenta imediatamente questões de nível avançado (B2/C1). Se o usuário demonstrar dificuldade, o teste ajusta-se para o nível básico (A1/A2)23. Esta tecnologia psicométrica reduz o tempo de avaliação de quarenta e cinco minutos para aproximadamente dez minutos, oferecendo um diagnóstico preciso e preservando o engajamento inicial do usuário22.

## **Sistemas de Retenção e Gamificação Temática**

A motivação é a variável mais crítica no aprendizado de idiomas a longo prazo. Aplicativos genéricos utilizam mascotes e moedas virtuais, mas uma plataforma direcionada a profissionais de tecnologia deve empregar uma gamificação que ressoe com o seu contexto diário.  
O engajamento visual central do aplicativo será modelado através de um *Heatmap* de Contribuições, uma metáfora visual diretamente extraída do ecossistema do GitHub26. Cada lição concluída, áudio gravado ou revisão espaçada realizada representará um "commit" na plataforma. O usuário verá seu painel preencher-se com tonalidades progressivamente mais intensas de cor à medida que estuda consecutivamente, criando uma representação tangível do seu esforço ao longo do ano26.  
Adicionalmente, o sistema de progressão substituirá os arquétipos tradicionais por insígnias (*Badges*) técnicas. A conclusão de módulos de pronúncia exata pode render o título de "Syntax Master", enquanto a resolução de exercícios de tradução de documentações pode conceder a insígnia "Bug Squasher". A introdução de tabelas de classificação (*Leaderboards*) opcionais permite que os alunos comparem seus níveis de experiência (XP) com seus pares, promovendo uma competição saudável e elevando as taxas de retenção diária27.

## **Engenharia Arquitetônica e Gestão de Estado Offline**

A espinha dorsal de um *Progressive Web App* resiliente reside na implementação de *Service Workers*. Estes roteadores virtuais operam em uma *thread* separada do navegador, interceptando todas as requisições de rede para fornecer respostas cacheadas, viabilizando o funcionamento parcial da plataforma mesmo em cenários de total ausência de internet1.  
O desenvolvimento exigirá o emprego de estratégias de cache cirúrgicas, balanceando a velocidade da interface com a necessidade de dados atualizados. O modelo arquitetônico utilizará o padrão *App Shell*, onde a estrutura básica da interface de usuário (barras de navegação, cabeçalhos, painéis estruturais) é armazenada de forma estática.

| Tipo de Recurso | Estratégia de Cache Recomendada | Justificativa Arquitetônica |
| :---- | :---- | :---- |
| **Arquivos HTML/CSS/JS (App Shell)** | *Cache-First* (com controle de versão) | Garante tempo de carregamento inicial quase instantâneo, contornando a latência da rede1. |
| **Imagens, Ícones e Fontes** | *Cache-First* | Ativos estáticos pesados que raramente sofrem mutações e consomem alta largura de banda1. |
| **Dados do Perfil e Histórico** | *Stale-While-Revalidate* | A interface renderiza o dado salvo localmente para velocidade máxima e, simultaneamente, atualiza o cache em segundo plano com novos dados do servidor1. |
| **Processamento de Áudio e Login** | *Network-Only* | Mutação de dados sensíveis e inferência pesada de IA requerem conectividade com a infraestrutura primária1. |

Dados estruturados adicionais, como o progresso detalhado das lições e o dicionário de revisão espaçada, serão alocados no *IndexedDB*. Ao contrário do localStorage, que é bloqueante e limitado a 5MB, o *IndexedDB* suporta o armazenamento assíncrono de centenas de megabytes, permitindo que módulos inteiros de exercícios em áudio sejam baixados para estudo durante voos ou viagens sem conexão1.

## **Tecnologias Avançadas de Reconhecimento de Voz**

O diferencial competitivo da plataforma reside na capacidade de decodificar e avaliar o sotaque de falantes de português brasileiro comunicando-se em inglês. A arquitetura de processamento de voz pode ser escalada em duas frentes independentes para otimizar custos e performance.  
A primeira frente é o processamento em nuvem otimizado financeiramente. A OpenAI disponibiliza a API de transcrição Whisper com múltiplos modelos. Para o contexto deste produto, o modelo gpt-4o-mini-transcribe apresenta-se como a escolha ideal para o MVP, oferecendo alta precisão por um custo estimado de apenas US$ 0,003 por minuto de áudio transcrito7. Esse valor, significativamente inferior ao modelo legado whisper-1 (US$ 0,006/min), permite sustentar um alto volume de usuários no nível gratuito sem inviabilizar o fluxo de caixa do projeto7.  
A segunda frente, projetada para a evolução tecnológica da plataforma, é a inferência 100% local através do navegador utilizando tecnologias como *Transformers.js*, *whisper.cpp* compilado para *WebAssembly*, e aceleração *WebGPU*8. Esta abordagem elimina completamente os custos de servidor, pois o modelo neural (versões quantizadas de 75MB a 140MB) é baixado diretamente para a RAM do dispositivo do usuário32. Entretanto, devido às limitações de *single-thread* do *WebAssembly* em muitos navegadores móveis, a transcrição local baseada em CPU pode demorar até 40 segundos para processar 30 segundos de áudio32. Portanto, a arquitetura implementará uma lógica condicional: se o navegador do usuário suportar aceleração *WebGPU* ou possuir recursos abundantes de RAM (tipicamente usuários desktop), o sistema operacional executará o modelo Whisper localmente, garantindo privacidade e agilidade. Caso contrário, o áudio será roteado para a nuvem via API8.

## **Design de Acessibilidade e a Engenharia do Tema Escuro**

A exigência de criar um aplicativo "lindo, intuitivo e com temas escuros" exige uma abordagem de design baseada em acessibilidade, superando a simples inversão de cores de uma interface clara. Modos escuros desenhados inadequadamente agravam condições de astigmatismo e induzem rápida fadiga ocular devido a problemas de vibração de contraste33.  
A fundação do tema escuro repousa na recusa sistemática do preto puro (\#000000). Fundos excessivamente escuros, quando contrastados com textos claros, forçam a íris a abrir-se demasiadamente para captar luz, criando um efeito de halo ao redor das letras, prejudicando severamente a legibilidade33. A interface deve adotar escalas tonais de cinza profundo ou carvão, como o hexadecimal \#121212 ou \#1A1A1A, que absorvem o brilho excessivo enquanto mantêm a profundidade visual35.  
A tipografia deve aderir rigorosamente aos padrões de legibilidade da Web Content Accessibility Guidelines (WCAG) Nível AA, que estipulam uma proporção mínima de contraste de 4.5:1 para textos normais33. Para mitigar o ofuscamento, o texto principal será renderizado em cinza claro (ex: \#E0E0E0) em vez de branco puro (\#FFFFFF)35. Além disso, as cores de marca que indicam ação primária (botões de submissão, alertas de erro em vermelho, confirmações em verde) sofrerão dessaturação. Cores altamente vibrantes em fundos escuros criam uma ilusão ótica de vibração que desconforta a retina. O design visual utilizará tons pastel ou semi-transparentes para garantir que a interface seja fluida, elegante e ergonômica para sessões prolongadas de estudo33.

# **Especificação de Produto: Documento de Requisitos (PRD)**

Este documento traduz a visão analítica e estratégica consolidada acima em requisitos técnicos e funcionais acionáveis. Projetado na sintaxe Markdown, este PRD serve como o artefato fundamental para o consumo direto de equipes de engenharia de software e arquitetos de soluções baseadas em Inteligência Artificial, guiando o desenvolvimento desde o protótipo até o produto viável mínimo (MVP).

## **1\. Visão do Produto e Perfis de Utilizadores**

O **FluentDev PWA** (nome de projeto provisório) é uma plataforma educacional progressiva voltada para o ensino de inglês comunicativo e técnico, com foco inicial no público brasileiro (falantes de pt-br). Através da convergência de inteligência artificial de reconhecimento de voz, algoritmos de repetição espaçada e metodologias de ensino baseadas em tarefas reais, o produto visa destravar a comunicação oral de profissionais que possuem barreira no idioma.  
A modelagem de produto contempla duas personas principais, orientando a hierarquia de funcionalidades e a linguagem de interface:

1. **O Desenvolvedor Júnior Globalizado:** Possui entre 20 e 35 anos, consome literatura técnica e documentação de APIs em inglês diariamente, mas enfrenta paralisação cognitiva ao tentar articular raciocínios lógicos em reuniões de áudio e vídeo com equipes estrangeiras. A busca principal é por simulações técnicas contextuais e superação da fobia de falar.  
2. **O Profissional em Preparação Executiva/Viagem:** Especialista em sua área que necessita realizar viagens internacionais de trabalho para os Estados Unidos. O domínio gramatical profundo é secundário; a prioridade absoluta é o "Inglês de Sobrevivência" rápido, funcional e situacional (passagem pela imigração alfandegária, reserva de locadoras de veículos, comunicação de contratempos hospitalares ou hoteleiros).

## **2\. Requisitos Funcionais do MVP (Épicos e Histórias de Utilizador)**

O MVP será estruturado através de grandes épicos ágeis, delimitando estritamente os domínios que devem ser construídos para comprovar a viabilidade técnica e a adesão do usuário.

### **Épico 1: Onboarding Dinâmico e Nivelamento Adaptativo**

O primeiro contato do utilizador com a plataforma deve minimizar o tempo de preenchimento de formulários e direcioná-lo rapidamente para a avaliação do seu nível atual de proficiência.

* **Requisito Funcional (RF-1.1):** O sistema deve oferecer suporte a Autenticação Única (SSO) via provedores OAuth (GitHub e Google) para reduzir a fricção de registro, além do fluxo padrão de e-mail e credencial criptografada.  
* **Requisito Funcional (RF-1.2):** Imediatamente após a criação da conta, o sistema ativará o Motor de Avaliação Adaptativa (Baseado em CAMST).  
* **Requisito Funcional (RF-1.3):** O algoritmo de avaliação deve possuir um banco restrito de perguntas calibradas entre os níveis CEFR A1 a C1. O motor apresentará *testlets* (conjuntos de 3 questões). Acertos superiores a 70% elevam o nível do próximo *testlet*; acertos inferiores a 40% reduzem a complexidade. A avaliação encerrará obrigatoriamente após 12 interações, emitindo um diagnóstico preciso.  
* **Requisito Funcional (RF-1.4):** Com base no resultado do nivelamento, o sistema liberará acesso a uma Trilha de Aprendizado Modular (Básico, Intermediário ou Avançado), bloqueando as trilhas de maior complexidade temporalmente.

### **Épico 2: Motor de Exercícios e Metodologia Situacional**

A essência do aprendizado fluirá através da execução de lições compostas por múltiplos tipos de avaliação contextual.

* **Requisito Funcional (RF-2.1):** A arquitetura da lição deve seguir o modelo TBLT (Ensino Baseado em Tarefa). Em vez de apresentar blocos teóricos exaustivos, a tela apresentará um problema de negócios ou viagem (ex: "Um erro interno 500 ocorreu no servidor, explique a situação").  
* **Requisito Funcional (RF-2.2):** Módulos de Escrita (*Writing*): A interface solicitará a tradução ou conclusão textual de frases idiomáticas. O validador de resposta de texto deve possuir tolerância a pequenos erros de digitação (*typos*) baseados na distância de Levenshtein, focando no acerto semântico.  
* **Requisito Funcional (RF-2.3):** Módulos de Escuta (*Listening*): O aplicativo deve reproduzir áudios sintetizados ou pré-gravados e disponibilizar opções de múltipla escolha ou ordenação de blocos de palavras para formar a sentença escutada.

### **Épico 3: Avaliação de Fala Neural (Speech-to-Text Analytics)**

A capacidade de gravar e receber avaliação imediata sobre a dicção é o núcleo de inovação da plataforma.

* **Requisito Funcional (RF-3.1):** Em exercícios de *Speaking*, a interface exibirá uma sentença em inglês-americano. O utilizador deve pressionar o botão de microfone, disparando a API nativa do navegador para gravação do áudio em buffer. O tempo limite de gravação por interação é de 30 segundos.  
* **Requisito Funcional (RF-3.2):** O sistema tentará processar o áudio gravado transcrevendo o conteúdo falado. A transcrição retornada pelo motor de IA será comparada com a string alvo (gabarito) utilizando algoritmos de análise de similaridade.  
* **Requisito Funcional (RF-3.3):** O *feedback* visual na tela mostrará a percentagem de similaridade. Se a similaridade exceder 80%, a tarefa é marcada como bem-sucedida. Palavras omitidas ou pronunciadas de forma ininteligível serão destacadas em cor de contraste (vermelho dessaturado) para correção imediata do utilizador.

### **Épico 4: Engajamento Visual e Gamificação Técnica**

Os sistemas de retenção devem aplicar mecânicas comportamentais conhecidas pela comunidade de desenvolvedores para formar hábitos diários de estudo.

* **Requisito Funcional (RF-4.1):** O painel principal (*Dashboard*) exibirá permanentemente um contador de ofensiva (*Streak*), marcando os dias consecutivos em que pelo menos uma lição ou revisão foi completada.  
* **Requisito Funcional (RF-4.2):** Abaixo do contador, um Gráfico de Contribuições (*Activity Heatmap*) exibirá os últimos 90 dias de atividade em uma matriz de quadrados coloridos. A saturação da cor do quadrado de um dia específico será diretamente proporcional ao volume de interações realizadas.  
* **Requisito Funcional (RF-4.3):** O sistema deve gerenciar uma fila de Revisão Espaçada. Frases e vocabulários técnicos cujo utilizador tenha falhado recentemente devem ser agendados automaticamente para reaparecer como exercícios rápidos nos acessos dos dias subsequentes.

## **3\. Requisitos Não Funcionais (Limites e Qualidade de Software)**

As diretrizes arquiteturais garantem que a fundação de engenharia suporte o crescimento do produto e não degrade a experiência móvel.

| Identificador | Categoria | Descrição Arquitetural e Limites |
| :---- | :---- | :---- |
| **RNF-01** | Performance (Latência de IA) | O fluxo completo de gravação de áudio, processamento remoto via API do Whisper e renderização do resultado na interface não deve exceder um p95 de 3.5 segundos em redes 4G estáveis. |
| **RNF-02** | Conformidade PWA | O produto deve registrar obrigatoriamente um ficheiro manifest.json com ícones escaláveis (192px e 512px) e fornecer a experiência *Standalone* de ecrã inteiro em dispositivos móveis Android e iOS. |
| **RNF-03** | Resiliência Offline (App Shell) | O *Service Worker* deve armazenar em cache prioritário todos os artefatos visuais essenciais (HTML base, folhas de estilo CSS minimizadas, bundles de scripts vitais) assegurando carregamento da casca do aplicativo inferior a 1.5 segundos, mesmo em condições de ausência de rede (*offline-first*). |
| **RNF-04** | Acessibilidade Contraste (UI) | O tema escuro da aplicação basear-se-á em fundos atenuados (ex: hex \#121212). Todo texto, ícone instrucional e botão interativo deverá passar nos validadores de contraste WCAG 2.1 Nível AA (proporção mínima 4.5:1 em relação ao plano de fundo). |
| **RNF-05** | Fallback de Processamento | A arquitetura avaliará as capacidades de inferência do navegador cliente. Caso o dispositivo do utilizador não suporte execução local fluida do modelo de IA via *WebAssembly/WebGPU*, o sistema automaticamente comutará a requisição para processamento seguro no lado do servidor (Cloud API). |

## **4\. Arquitetura de Dados Recomendada (Entidades Principais)**

A normalização de dados orienta-se para a alta consistência nas transações de progresso do usuário. A modelagem primária sugerida para a implantação de um banco de dados relacional (ex: PostgreSQL) engloba:

* **Entidade Users**: Gere a identidade e telemetria base.  
  * id (UUID), oauth\_provider\_id, email, display\_name, proficiency\_level\_assigned, current\_streak, longest\_streak, created\_at.  
* **Entidade Modules**: Agrupadores semânticos de conteúdo (Ex: Modulo de Viagem Básica, Módulo de Engenharia Intermediária).  
  * id, title, description, theme\_type (Enum: TRAVEL / TECH), difficulty\_level, sequential\_order.  
* **Entidade Lessons**: Blocos unitários de ensino atrelados a um módulo.  
  * id, module\_id (FK), lesson\_title, pedagogical\_focus, experience\_points\_reward.  
* **Entidade Exercises**: Tarefas interativas acopladas às lições.  
  * id, lesson\_id (FK), exercise\_type (Enum: TRANSLATE, SPEAKING, LISTENING, FILL\_BLANK), prompt\_context, target\_answer\_text, audio\_asset\_url (se aplicável).  
* **Entidade User\_Progress\_Logs**: Registo imutável de interações para cálculo do heatmap e nivelamento.  
  * id, user\_id (FK), exercise\_id (FK), completion\_timestamp, accuracy\_score, is\_spaced\_repetition\_review.

## **5\. Roteiro Estratégico de Evolução (Roadmap de Lançamento)**

A execução da engenharia será modularizada, isolando a complexidade e garantindo um *Time-to-Market* reduzido para coleta de métricas reais de uso.  
**Fase 1: Minimum Viable Product (Mês 1 ao Mês 3\)**

* Configuração da infraestrutura de repositório, CI/CD e implantação do banco de dados relacional (Supabase, Vercel Postgres, ou similar).  
* Construção do Frontend React/Vue configurado com integração robusta PWA (manifest, workers básicos).  
* Implementação do Teste de Nivelamento (CAMST) simplificado.  
* Disponibilização das primeiras 20 lições de conteúdo englobando o eixo Tech e o eixo Viagem.  
* Integração da captura de áudio com a API em nuvem OpenAI (gpt-4o-mini-transcribe) para o motor de *Speaking*.

**Fase 2: Retenção e Inteligência Local (Mês 4 ao Mês 6\)**

* Refinamento visual: Implantação profunda do *Dark Mode* acessível e animações de feedback semiótico.  
* Construção e ativação do Gráfico de *Heatmap* de atividades no painel do utilizador.  
* Estabilização avançada do *Service Worker*, ativando estratégias *Stale-While-Revalidate* e salvamento local profundo via *IndexedDB*.  
* Experimento alfa de inferência local (*whisper.cpp* / *Transformers.js*) para usuários de navegadores de mesa habilitados para *WebGPU*.

**Fase 3: Escala Dinâmica (Mês 7 ao Mês 12\)**

* Integração de Push Notifications (suportado pelo iOS 16.4+) para atuar como catalisador comportamental de retorno do usuário à plataforma.  
* Dinamização da geração de exercícios: utilização de grandes modelos de linguagem (LLMs) corporativos para gerar diálogos e *role-plays* únicos baseados nos erros sintáticos e lexicais específicos que um utilizador tem cometido historicamente.  
* Painéis corporativos (B2B): liberação de visões analíticas para gestores de engenharia acompanharem o desenvolvimento linguístico de suas equipes técnicas na plataforma.

#### **Referências citadas**

1. Frontend System Design: Offline Support and Progressive Web Apps (PWAs), [https://dev.to/zeeshanali0704/frontend-system-design-offline-support-and-progressive-web-apps-pwas-4k8m](https://dev.to/zeeshanali0704/frontend-system-design-offline-support-and-progressive-web-apps-pwas-4k8m)  
2. Caching \- Progressive web apps | MDN, [https://developer.mozilla.org/en-US/docs/Web/Progressive\_web\_apps/Guides/Caching](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Caching)  
3. Speech recognition in the browser using Web Speech API \- AssemblyAI, [https://www.assemblyai.com/blog/speech-recognition-javascript-web-speech-api](https://www.assemblyai.com/blog/speech-recognition-javascript-web-speech-api)  
4. Taming the Web Speech API \- Andrea Giammarchi \- Medium, [https://webreflection.medium.com/taming-the-web-speech-api-ef64f5a245e1](https://webreflection.medium.com/taming-the-web-speech-api-ef64f5a245e1)  
5. React Pwa issue · Issue \#104 · JamesBrill/react-speech-recognition \- GitHub, [https://github.com/JamesBrill/react-speech-recognition/issues/104](https://github.com/JamesBrill/react-speech-recognition/issues/104)  
6. Whisper 1 \- API Pricing & Providers \- OpenRouter, [https://openrouter.ai/openai/whisper-1](https://openrouter.ai/openai/whisper-1)  
7. OpenAI Whisper API Pricing 2026: Cost Per Minute, GPT-4o Transcribe and Cheaper Alternatives \- DIY AI, [https://diyai.io/ai-tools/speech-to-text/openai-whisper-api-pricing-2026/](https://diyai.io/ai-tools/speech-to-text/openai-whisper-api-pricing-2026/)  
8. Offline speech recognition with Whisper: Browser \+ Node.js implementations \- AssemblyAI, [https://www.assemblyai.com/blog/offline-speech-recognition-whisper-browser-node-js](https://www.assemblyai.com/blog/offline-speech-recognition-whisper-browser-node-js)  
9. Whisper Web \- Free AI Speech Recognition | Browser-Based Transcription, [https://whisperweb.dev/](https://whisperweb.dev/)  
10. PWA iOS Limitations and Safari Support \[2026\] \- MagicBell, [https://www.magicbell.com/blog/pwa-ios-limitations-safari-support-complete-guide](https://www.magicbell.com/blog/pwa-ios-limitations-safari-support-complete-guide)  
11. The Best Language Teaching Methods for Lasting Progress \- Global Lingua, [https://www.globallingua.ca/en/study-area/best-language-teaching-methods](https://www.globallingua.ca/en/study-area/best-language-teaching-methods)  
12. 12 Popular Language Learning Approaches \- Energiaa online newspaper \- VAMK, [https://energiaa.vamk.fi/en/articles/komptence/12-popular-language-learning-approaches/](https://energiaa.vamk.fi/en/articles/komptence/12-popular-language-learning-approaches/)  
13. Best Language Learning Methods, [https://altalang.com/beyond-words/best-language-learning-methods/](https://altalang.com/beyond-words/best-language-learning-methods/)  
14. The Top 10 Research-Backed Instructional Techniques for the language classroom, [https://gianfrancoconti.com/2025/03/27/the-science-of-modern-language-teaching-success-the-top-10-research-backed-instructional-techniques/](https://gianfrancoconti.com/2025/03/27/the-science-of-modern-language-teaching-success-the-top-10-research-backed-instructional-techniques/)  
15. English Vocabulary for Software Engineers, Developers, and Programmers, [https://www.languagetrainers.com/blog/tech-industry-english-vocabulary-for-software-engineers-developers-programmers/](https://www.languagetrainers.com/blog/tech-industry-english-vocabulary-for-software-engineers-developers-programmers/)  
16. English for Software Engineers, Developers, & Programmers \- Preply Business, [https://preply.com/en/blog/b2b-english-for-software-engineers-developers-and-programmers/](https://preply.com/en/blog/b2b-english-for-software-engineers-developers-and-programmers/)  
17. Technical English: Essential Vocabulary for Software Developers \- Immigo, [https://www.immigo.io/blog/technical-english-essential-vocabulary-for-software-developers](https://www.immigo.io/blog/technical-english-essential-vocabulary-for-software-developers)  
18. English for Software Engineers: 50 Free Resources & Exercises, [https://speaktechenglish.com/learn-english-for-software-engineers/](https://speaktechenglish.com/learn-english-for-software-engineers/)  
19. A2 English for Developers Certification (Beta) | freeCodeCamp.org, [https://www.freecodecamp.org/learn/a2-english-for-developers](https://www.freecodecamp.org/learn/a2-english-for-developers)  
20. English for Software Developers Course | Learn to Speak with Confidence | Cambly, [https://www.cambly.com/english/courses/632ca0947739bd98d3b297de?lang=en](https://www.cambly.com/english/courses/632ca0947739bd98d3b297de?lang=en)  
21. Fixed Form VS Adaptive Test Design in Language Proficiency Testing, [https://sealofbiliteracy.org/blog/fixed-form-vs-adaptive-test-design-language-proficiency-testing](https://sealofbiliteracy.org/blog/fixed-form-vs-adaptive-test-design-language-proficiency-testing)  
22. EV@LANG Placement Test | Campus Saint-Jean \- University of Alberta, [https://www.ualberta.ca/en/campus-saint-jean/programs/language-learning-and-assessment/evaluation-services/evalang.html](https://www.ualberta.ca/en/campus-saint-jean/programs/language-learning-and-assessment/evaluation-services/evalang.html)  
23. Adaptive Placement Test \[Beta\] \- Help Center, [https://help.off2class.com/betas/features/adaptive-placement-test](https://help.off2class.com/betas/features/adaptive-placement-test)  
24. How Does the Adaptive Placement Test Work? \- Support : MobyMax Help Center, [https://support.mobymax.com/support/solutions/articles/11000012081-how-does-the-adaptive-placement-test-work-](https://support.mobymax.com/support/solutions/articles/11000012081-how-does-the-adaptive-placement-test-work-)  
25. Adaptive Placement Test (A1-B2) \- Lingu, [https://lingu.no/tests/adaptiv-plasseringstest-i-norsk-lese-og-lyttetest](https://lingu.no/tests/adaptiv-plasseringstest-i-norsk-lese-og-lyttetest)  
26. Would u actually use this app if u are an student or developer ?? Need honest advice : r/ProgrammingBondha \- Reddit, [https://www.reddit.com/r/ProgrammingBondha/comments/1tt23l8/would\_u\_actually\_use\_this\_app\_if\_u\_are\_an\_student/](https://www.reddit.com/r/ProgrammingBondha/comments/1tt23l8/would_u_actually_use_this_app_if_u_are_an_student/)  
27. A Gamified Method for Teaching Version Control Concepts in Programming Courses Using the Git Education Game \- MDPI, [https://www.mdpi.com/2079-9292/13/24/4956](https://www.mdpi.com/2079-9292/13/24/4956)  
28. js13kGames: Making the PWA work offline with service workers \- Progressive web apps, [https://developer.mozilla.org/en-US/docs/Web/Progressive\_web\_apps/Tutorials/js13kGames/Offline\_Service\_workers](https://developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Tutorials/js13kGames/Offline_Service_workers)  
29. Offline-First PWAs: Service Worker Caching Strategies \- MagicBell, [https://www.magicbell.com/blog/offline-first-pwas-service-worker-caching-strategies](https://www.magicbell.com/blog/offline-first-pwas-service-worker-caching-strategies)  
30. OpenAI Transcription & Whisper API Pricing Calculator \- CostGoat, [https://costgoat.com/pricing/openai-transcription](https://costgoat.com/pricing/openai-transcription)  
31. GitHub \- ggml-org/whisper.cpp: Port of OpenAI's Whisper model in C/C++, [https://github.com/ggml-org/whisper.cpp](https://github.com/ggml-org/whisper.cpp)  
32. Speech-to-Text Local: Whisper e WebAssembly Offline-First | Tech Blog AI, [https://techlog.ia.br/posts/speech-to-text-local-whisper-webassembly-offline](https://techlog.ia.br/posts/speech-to-text-local-whisper-webassembly-offline)  
33. The Designer's Guide to Dark Mode Accessibility, [https://www.accessibilitychecker.org/blog/dark-mode-accessibility/](https://www.accessibilitychecker.org/blog/dark-mode-accessibility/)  
34. Inclusive Dark Mode: Designing Accessible Dark Themes For All Users, [https://www.smashingmagazine.com/2025/04/inclusive-dark-mode-designing-accessible-dark-themes/](https://www.smashingmagazine.com/2025/04/inclusive-dark-mode-designing-accessible-dark-themes/)  
35. Designing for Dark Mode: Best Practices for a Better User Experience | by Isabella Doyle, [https://medium.com/@poromepene42/designing-for-dark-mode-best-practices-for-a-better-user-experience-8f4dbbfedf44](https://medium.com/@poromepene42/designing-for-dark-mode-best-practices-for-a-better-user-experience-8f4dbbfedf44)  
36. Colour Contrast Tips for Creating Accessible Web Designs, [https://accessibilityinnovations.com/colour-contrast/](https://accessibilityinnovations.com/colour-contrast/)  
37. Accessible Colors: What They Are and How to Design With Color Accessibility \- AudioEye, [https://www.audioeye.com/post/accessible-colors/](https://www.audioeye.com/post/accessible-colors/)