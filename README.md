# FluentDev — English Learning PWA for Tech Professionals

PWA instalável com suporte offline que ensina inglês comunicativo e técnico a falantes de
português do Brasil. Personas: **dev júnior** e **viajante executivo**. UI em pt-BR, conteúdo de aprendizado em en-US.

> **Projeto-laboratório:** construído para explorar o fluxo de desenvolvimento **Spec-Driven Development (SDD)** com o modelo **Claude Sonnet 4.6**, validando como IA generativa pode guiar de ponta a ponta a especificação, planejamento e implementação de um produto real.

---

## Acesso Online

A aplicação está disponível em produção, hospedada gratuitamente no **Fly.io** (região GRU — São Paulo):

**[https://fluentdev.fly.dev](https://fluentdev.fly.dev)**

### Como está disponível gratuitamente

O deploy usa exclusivamente o free tier do Fly.io, sem cartão de crédito obrigatório:

| Recurso | Configuração |
|---------|-------------|
| CPU | 1 vCPU compartilhada (`shared`) |
| Memória | 256 MB RAM |
| Região | `gru` — São Paulo (menor latência para pt-BR) |
| Scale-to-zero | `auto_stop_machines = stop` — a máquina hiberna quando ociosa |
| Cold start | `auto_start_machines = true` — acorda na primeira requisição (~2 s) |
| Mínimo de máquinas | `0` — zero custo quando sem tráfego |

O build é feito remotamente pelo Fly.io a partir do `Dockerfile` multi-stage no repositório.
Nenhum dado sensível vai para o repositório: segredos (DATABASE_URL, chaves de API, OAuth secrets) são injetados via `fly secrets set`.

---

## O que é

| Funcionalidade | Detalhe |
|---|---|
| **Teste de nivelamento adaptativo** | CAMST band-walk A1–C1, ≤12 questões, diagnóstico Basic/Intermediate/Advanced |
| **Lições baseadas em tarefas** | Temas travel + tech, escrita com tolerância a typos, listening (múltipla escolha + word ordering) |
| **Prática de fala** | Gravação ≤30 s → Groq Whisper → similaridade WER (≥80% = aprovado), highlight de palavras erradas |
| **Repetição espaçada** | Revisões em 1 → 3 → 7 → 21 dias |
| **Retenção gamificada** | Streak diário, heatmap de atividade 90 dias (estilo GitHub), XP imutável no `progress_logs` |
| **PWA** | Instalável no Android/iOS, shell offline, writes offline reprocessados via outbox idempotente |

---

## Construção com Spec-Driven Development (SDD) e Fable 5

Este projeto foi construído usando **Spec-Driven Development**, uma metodologia que coloca a especificação formal como artefato central — não código — e usa IA generativa para guiar cada fase:

```
Ideia  →  Spec  →  Plan  →  Tasks  →  Implement  →  Validate
          (AI)    (AI)     (AI)       (AI + dev)     (AI + dev)
```

### Por que SDD com Fable 5

O **Claude Fable 5** foi usado como par de engenharia em todo o ciclo:
- Refinar o PRD em [User Stories com critérios de aceitação](specs/001-fluentdev-pwa/spec.md) testáveis
- Gerar o [plano de implementação](specs/001-fluentdev-pwa/plan.md) com sequência de dependências
- Produzir tasks granulares com contratos claros (entrada/saída, camada responsável)
- Escrever código alinhado ao contrato [OpenAPI](specs/001-fluentdev-pwa/contracts/openapi.yaml)
- Revisar segurança, cobertura de testes e consistência arquitetural

Artefatos do ciclo SDD estão em [`specs/001-fluentdev-pwa/`](specs/001-fluentdev-pwa/).

---

## Arquitetura

### Visão geral

```
┌──────────────────────────────────────────────────┐
│                    Frontend (PWA)                 │
│  React 19 · TypeScript 5 · Vite · Workbox         │
│  TanStack Query · Dexie (IndexedDB) · Zustand     │
│                                                    │
│   ┌─────────────┐    ┌──────────────────────────┐ │
│   │ Service     │    │  Offline Outbox           │ │
│   │ Worker      │    │  (UUID v7 · idempotent)   │ │
│   └─────────────┘    └──────────────────────────┘ │
└───────────────────────┬──────────────────────────┘
                        │ HTTPS /api/v1
┌───────────────────────▼──────────────────────────┐
│                    Backend (Go)                   │
│                                                    │
│  domain ──► usecase ──► adapter ──► infra         │
│   (puro)    (puro)      (chi)       (pgx · groq)  │
└───────────────────────┬──────────────────────────┘
                        │
              ┌─────────▼──────────┐
              │   PostgreSQL 16    │
              │  (fonte da verdade)│
              └────────────────────┘
```

### Backend — Clean Architecture

Dependências apontam **sempre para dentro**: `infra` depende de `adapter`, que depende de `usecase`, que depende de `domain`. `domain` tem **zero dependências externas**.

```
internal/
├── domain/         # entidades e regras puras (WER, CAMST, streak, XP)
│   ├── placement/  # lógica do teste adaptativo
│   ├── speech/     # similarity score (1 − WER)
│   ├── progress/   # streak, heatmap
│   └── content/    # estrutura de lições e exercises
├── usecase/        # orquestração de casos de uso
├── adapter/
│   ├── http/       # handlers chi + middleware (CSRF, rate limit, headers)
│   └── postgres/   # queries sqlc geradas, migrations
└── infra/
    ├── config/     # parsing de env com validação
    ├── transcriber/ # porta Transcriber (Groq → OpenAI failover)
    └── oauth/      # GitHub + Google PKCE
```

### Frontend — Camadas

```
src/
├── features/       # feature slices (placement, lesson, speech, progress)
├── api/            # clientes TanStack Query + outbox offline
├── store/          # Zustand (session, ui state)
├── components/     # UI compartilhado (a11y-first, dark theme)
└── sw/             # Workbox strategies (App Shell + stale-while-revalidate)
```

### Contrato de API

REST sob `/api/v1`, contract-first contra [`contracts/openapi.yaml`](specs/001-fluentdev-pwa/contracts/openapi.yaml).
Erros seguem **RFC 9457** (`application/problem+json`).

---

## Boas Práticas

### Segurança (OWASP ASVS L1)

| Prática | Implementação |
|---------|--------------|
| Autenticação | Cookie `HttpOnly / Secure / SameSite=Lax` + CSRF double-submit token |
| Senhas | argon2id (configurável: time, memory, parallelism) |
| SQL | 100% parameterizado via sqlc — zero SQL dinâmico |
| Rate limiting | Endpoints de auth e speech com janela deslizante |
| Headers HTTP | `Content-Security-Policy`, `X-Frame-Options`, `Referrer-Policy` via middleware |
| OAuth | PKCE obrigatório (GitHub + Google), state CSRF |
| Logs | Nenhum PII ou áudio em logs (`slog` JSON estruturado) |
| Scan | `scripts/zap-baseline.sh` (OWASP ZAP) em CI |

### Integridade de dados

- Todos os PKs são **UUID v7** gerados no servidor (ou no cliente para offline-outbox, deduplicados no sync)
- `progress_logs` é **INSERT-only** — nenhum UPDATE/DELETE permitido (log imutável de atividade)
- Day-bucketing de streak e heatmap usa `users.timezone` (IANA) — nunca UTC assumido
- PostgreSQL 16 é a fonte da verdade; IndexedDB é cache leitura + outbox de escrita

### Resiliência offline

- **App Shell** cacheado pelo Service Worker (Workbox)
- Progresso e writes vão para o `outbox` (IndexedDB) quando offline
- Sync com retry idempotente ao reconectar (UUID v7 garante deduplicação)
- Compatível com política de 7 dias do Safari/iOS (dados críticos sempre sincronizados na nuvem)

### Transcrição de fala

A porta `Transcriber` isola completamente os provedores de IA:
- **Primário**: Groq `whisper-large-v3-turbo` (mais rápido, menor custo)
- **Failover**: OpenAI `gpt-4o-mini-transcribe` (automático, transparente)
- Nenhum código fora de `internal/infra/transcriber` chama provedores diretamente

---

## Stack

| Camada | Tecnologia |
|--------|-----------|
| **Backend** | Go 1.24, chi, pgx/v5, sqlc, golang-migrate, argon2id, slog (JSON), golang.org/x/oauth2 |
| **Frontend** | React 19, TypeScript 5, Vite, vite-plugin-pwa (Workbox), TanStack Query, Dexie, Zustand |
| **Banco** | PostgreSQL 16 |
| **Fala** | Groq `whisper-large-v3-turbo` → OpenAI `gpt-4o-mini-transcribe` (failover) |
| **Deploy** | Docker multi-stage, Fly.io (GRU), GitHub Actions CI/CD |

---

## Rodando localmente

### Pré-requisitos

- Go 1.24+
- Node.js 20+
- Docker

### Passo a passo

```bash
# 1. Banco de dados
docker compose up -d postgres

# 2. Backend — aplica migrations + seed (20 lições, 75 questões de nivelamento), :8080
cd backend
cp .env.example .env   # preencha as chaves — nunca commite .env
go run ./cmd/api -migrate -seed

# 3. Frontend — dev server :5173 com proxy /api → :8080
cd frontend
npm install && npm run dev

# Build produção (Service Worker ativo)
npm run build && npm run preview
```

---

## Testes e gates de qualidade

```bash
# Backend
cd backend
make lint             # golangci-lint + go vet + govulncheck
make test             # unit + httptest contract tests
make test-integration # Postgres real via testcontainers (requer Docker)
make coverage         # gate: ≥80% em internal/domain + internal/usecase

# Frontend
cd frontend
npm run lint && npm run typecheck          # ESLint + tsc --noEmit
npm run test                               # Vitest + Testing Library
npm run test:e2e                           # Playwright: jornadas US1–US5, offline, axe-core a11y
```

### Gates obrigatórios para merge

- Cobertura ≥ 80% nas camadas `domain` e `usecase`
- `tsc --noEmit` sem erros
- `npm audit` sem vulnerabilidades críticas/altas
- Todos os testes E2E (incluindo modo offline) passando
- Baseline ZAP sem findings de severidade média ou superior

---

## CI/CD

GitHub Actions em `.github/workflows/`:

1. **lint-test** — lint, vet, vuln scan, unit tests, coverage gate
2. **e2e** — build Docker + Playwright (incl. offline)
3. **deploy** — `fly deploy` automatico em merge para `main`

---

## Estrutura do Repositório

```
.
├── backend/              # API Go (Clean Architecture)
│   ├── cmd/api/          # entrypoint (flags: -migrate, -seed, -static-dir)
│   ├── internal/         # domain · usecase · adapter · infra
│   ├── migrations/       # SQL migrations (golang-migrate)
│   └── seed/             # dados iniciais (lições + banco de questões)
├── frontend/             # PWA React/Vite
│   ├── src/              # features · api · store · components · sw
│   └── tests/            # Playwright E2E
├── specs/001-fluentdev-pwa/  # Artefatos SDD
│   ├── spec.md           # 5 user stories P1–P5, FR-001…FR-025
│   ├── plan.md           # plano de implementação
│   ├── tasks.md          # tasks granulares com contratos
│   ├── data-model.md     # modelo de dados
│   ├── research.md       # pesquisa técnica e pedagógica
│   ├── contracts/        # openapi.yaml (contract-first)
│   └── quickstart.md     # validação e critérios de aceite
├── Dockerfile            # multi-stage: frontend build → backend build → runtime Alpine
├── docker-compose.yml    # PostgreSQL 16 local
├── fly.toml              # configuração Fly.io (GRU, scale-to-zero)
└── scripts/
    └── zap-baseline.sh   # scan OWASP ZAP
```

---

## Variáveis de ambiente

Copie `backend/.env.example` para `backend/.env`. Em produção, use `fly secrets set`.

| Variável | Descrição |
|----------|-----------|
| `DATABASE_URL` | PostgreSQL connection string |
| `SESSION_SECRET` | segredo HMAC para cookies (≥32 bytes aleatórios) |
| `GITHUB_CLIENT_ID / SECRET` | OAuth app GitHub |
| `GOOGLE_CLIENT_ID / SECRET` | OAuth app Google |
| `GROQ_API_KEY` | Groq Whisper (transcrição primária) |
| `OPENAI_API_KEY` | OpenAI (failover de transcrição) |
| `APP_BASE_URL` | URL pública da aplicação (ex.: `https://fluentdev.fly.dev`) |
| `TRANSCRIBE_PRIMARY` | `groq` (padrão) ou `openai` |

---

## Licença

MIT
