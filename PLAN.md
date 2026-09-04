# Plano de Execução — API de Gestão de Clínicas (Go)

## Contexto

Construir uma API em Go para gestão de clínicas odontológicas e seus dentistas, com recebimento de pagamentos via Pix simulado. Requisitos funcionais e decisões arquiteturais definidos pelo engenheiro responsável pelo projeto: Arquitetura Hexagonal, `net/http` nativo (Go 1.22+), `log/slog` em JSON, persistência in-memory (`map` + `sync.RWMutex`) atrás de interfaces, Git Flow com Conventional Commits, qualidade garantida por `golangci-lint`, testes por camada e automação via `Makefile`. O repositório está zerado (apenas `go.mod`, módulo `github.com/devictorh/clinics`, Go 1.27).

---

## 1. Escopo e Requisitos

### Core

**Clínicas**
- CRUD completo: criar, alterar, visualizar e excluir.
- Campos: documento (CPF/CNPJ), razão social, nome fantasia.
- Dados bancários (banco, conta, agência) — também alteráveis.

**Dentistas**
- Atrelados **necessariamente** a uma clínica.
- Campos: nome, telefone, email.
- Uma clínica pode ter um ou mais dentistas como **administrador e responsável legal**.

**Pagamentos (Pix)**
- `POST /payments` — recebe `clinic_id`, valor e `dentist_id` (opcional). Retorna identificador de cobrança + código Pix copia-e-cola **simulado**.
- Ciclo de status: `pending → approved`.
- Sem provedor real: processo em background aprova após delay aleatório (ex.: 2–5s).
- Cobrança só pode ser criada para clínica/dentista **ativos** (nunca para registro soft-deletado).

**Requisitos técnicos**
- Go com stdlib como base; dependências externas apenas quando justificadas.
- Código documentado (godoc) e boas práticas idiomáticas.
- Testes unitários e de integração cobrindo **sucesso e erro**.
- Camadas de serviço/negócio testadas com **fakes/mocks das interfaces** (sem banco/serviço externo real).
- Persistência **in-memory**.
- Exclusão sempre via **soft delete** — trata-se de dados financeiros; precisamos de trilha de auditoria.
- README com instruções de execução/testes e registro das decisões arquiteturais e trade-offs.
- Contrato da API formalizado em **OpenAPI 3.0** (`docs/openapi.yaml`, escrito à mão), servido pela própria aplicação com UI navegável — sem geradores por anotação nem dependências Go extras.

### Critérios de aceite
| Critério | O que significa pronto |
|---|---|
| Funcionalidade | CRUD de clínicas e dentistas correto ponta-a-ponta; fluxo Pix simulado funcional |
| Organização | Separação clara entre lógica de negócio e interface da API; naming e estrutura coerentes |
| Testes | Sucesso e erro cobertos, com fakes das interfaces de persistência e do provedor de pagamento |
| Interface | API intuitiva, fácil de integrar num front-end |
| Documentação | README completo com decisões técnicas defensáveis; spec OpenAPI fiel ao comportamento da API |

---

## 2. Contrato da API (proposta)

Base: `net/http` nativo (Go 1.22+ `http.ServeMux` com métodos e path params). Prefixo `/api/v1`. JSON em snake_case. Valores monetários em **centavos (int64)**. IDs UUID v4.

```
POST   /api/v1/clinics                                  201 | 400 | 409 (documento duplicado)
GET    /api/v1/clinics                                  200 (lista — facilita integração/validação)
GET    /api/v1/clinics/{clinicID}                       200 | 404
PUT    /api/v1/clinics/{clinicID}                       200 | 400 | 404 (dados cadastrais)
PUT    /api/v1/clinics/{clinicID}/bank-account          200 | 400 | 404 (dados bancários — endpoint dedicado)
DELETE /api/v1/clinics/{clinicID}                       204 | 404 (SOFT delete; cascata: soft delete dos dentistas)

POST   /api/v1/clinics/{clinicID}/dentists              201 | 400 | 404 (clínica inexistente ou deletada)
GET    /api/v1/clinics/{clinicID}/dentists              200 | 404
GET    /api/v1/clinics/{clinicID}/dentists/{dentistID}  200 | 404
PUT    /api/v1/clinics/{clinicID}/dentists/{dentistID}  200 | 400 | 404
DELETE /api/v1/clinics/{clinicID}/dentists/{dentistID}  204 | 404 (SOFT delete)

POST   /api/v1/payments                                 201 | 400 | 404  (REJEITA clínica/dentista deletado)
GET    /api/v1/payments/{paymentID}                     200 | 404       (permite acompanhar pending→approved)
GET    /api/v1/clinics/{clinicID}/payments              200 | 404       (extra)

GET    /healthz                                         200
GET    /openapi.yaml                                    200 (spec OpenAPI 3.0 embarcada)
GET    /docs                                            200 (Swagger UI estático consumindo /openapi.yaml)
```

- Dentista como **sub-recurso** de clínica: torna o vínculo obrigatório explícito na URL.
- **Soft delete** em clínicas e dentistas: `DELETE` marca `deleted_at` em vez de remover; registros deletados somem das leituras/listagens (respondem 404) mas permanecem no armazenamento. Excluir clínica soft-deleta seus dentistas em cascata.
- **Pagamentos validam registros ativos**: `POST /payments` rejeita `clinic_id` inexistente ou soft-deletado (404) e `dentist_id` inexistente, deletado ou não pertencente à clínica (400/404).
- Flag `admin` (bool) no dentista = "administrador e responsável legal".
- Erros num envelope consistente: `{"error": {"code": "...", "message": "...", "details": [...]}}`.
- Resposta do payment: `{id, clinic_id, dentist_id?, amount, status, pix_copy_paste_code, created_at, ...}`.

---

## 3. Estrutura de Pastas (Arquitetura Hexagonal)

```
clinics/
├── .github/
│   └── workflows/
│       └── ci.yml                   # lint + test em todo PR (status check da proteção de branch)
├── cmd/
│   └── api/
│       └── main.go                  # composition root: monta adapters→services, slog JSON, graceful shutdown
├── internal/
│   ├── core/                        # o hexágono — SEM imports de net/http ou adapters
│   │   ├── domain/                  # entidades, value objects, erros de domínio
│   │   │   ├── clinic.go            # Clinic + BankAccount (VO)
│   │   │   ├── dentist.go
│   │   │   ├── payment.go           # Payment + PaymentStatus
│   │   │   ├── document.go          # VO CPF/CNPJ com validação de dígitos verificadores
│   │   │   ├── amount.go            # VO centavos int64
│   │   │   └── errors.go            # ErrNotFound, ErrInvalidInput, ErrDocumentAlreadyExists...
│   │   ├── port/                    # Ports (interfaces) — contratos do hexágono
│   │   │   ├── repository.go        # ClinicRepository, DentistRepository, PaymentRepository
│   │   │   └── pix.go               # PixProvider (gera cobrança) + Delayer p/ confirmação testável
│   │   └── service/                 # casos de uso (application services)
│   │       ├── clinic.go
│   │       ├── dentist.go
│   │       └── payment.go           # cria cobrança + dispara confirmação em background
│   └── adapter/
│       ├── httpapi/                 # driving adapter (lado esquerdo)
│       │   ├── router.go            # http.ServeMux com padrões "POST /api/v1/clinics/{clinicID}"
│       │   ├── clinic_handler.go
│       │   ├── dentist_handler.go
│       │   ├── payment_handler.go
│       │   ├── docs_handler.go      # serve /openapi.yaml (embed) + /docs (Swagger UI estático)
│       │   ├── dto.go               # request/response structs + mapeamento domínio↔JSON
│       │   ├── errors.go            # domain error → status HTTP + envelope JSON
│       │   └── middleware/
│       │       ├── requestid.go     # gera/propaga X-Request-ID via context.Context
│       │       ├── logging.go       # access log slog (método, rota, status, duração, request_id)
│       │       └── recovery.go      # panic → 500 JSON
│       ├── memory/                  # driven adapter: map + sync.RWMutex por repositório
│       │   ├── clinic_repo.go       # índice secundário por documento p/ unicidade
│       │   ├── dentist_repo.go
│       │   └── payment_repo.go
│       └── pixsim/                  # driven adapter: provedor Pix simulado
│           └── provider.go          # gera txid + BR Code simulado; delay aleatório 2–5s injetável
├── docs/
│   ├── openapi.yaml                 # spec OpenAPI 3.0 escrita à mão — contrato formal da API
│   ├── embed.go                     # package docs: go:embed openapi.yaml p/ servir via HTTP
│   └── api.http                     # exemplos de requisições (VS Code REST Client / curl)
├── Makefile                         # build, run, test, test-race, cover, lint
├── .golangci.yml
├── .gitignore
├── CLAUDE.md                        # diretrizes operacionais p/ sessões de desenvolvimento assistido
├── PLAN.md                          # este plano
├── README.md                        # execução, arquitetura, testes, decisões técnicas
└── go.mod
```

Regra de dependência: `adapter → core` (nunca o contrário). `core` importa apenas stdlib. Handlers HTTP dependem de interfaces de serviço; services dependem de ports.

### Decisão de persistência: `map` + `sync.RWMutex`

Padrão adotado como decisão de projeto para todos os repositórios. As duas invariantes que ele deve garantir são tratadas assim (trade-off completo na seção 6):
1. **Unicidade de documento**: garantida DENTRO do repositório de clínicas (índice secundário `map[documento]id` protegido pelo mesmo mutex) — invariante atômica sob um único lock.
2. **Integridade referencial clínica↔dentista** (check-then-act entre dois repositórios): validada no service; janela de corrida teórica documentada como limitação conhecida. Repositórios independentes e mockáveis por entidade em vez de aggregate root com lock único.

---

## 4. Fases da Implementação (ordem cronológica)

Git Flow: `main` (estável, protegida — merge somente via Pull Request) + `develop`; cada fase em branch `feature/*` integrada a `develop` **via PR**; entrega via `release/1.0.0 → main` também via PR + tag. Todo merge passa por PR com título em Conventional Commits, descrição do escopo da etapa e merge commit sem squash (`--no-ff`), preservando o histórico incremental da branch. Conventional Commits em todos os commits. CI (GitHub Actions) executa `make lint` e `make test` em todo PR, servindo de status check obrigatório da proteção de branch.

### Etapa 0 — Setup & Fundações (`feature/project-setup`)
- `.gitignore`, `Makefile` (build, run, test, test-race, cover, lint), `.golangci.yml`, esqueleto de pastas, README inicial.
- `.golangci.yml`: `errcheck`, `govet`, `staticcheck`, `revive`, `gofumpt`, `goimports`, `errorlint`, `gocritic`, `unparam`, `bodyclose`, `noctx`, `tparallel`, `thelper`, `copyloopvar`; relaxamentos pontuais em `_test.go` (sem `misspell`: o dicionário é de inglês e a documentação do projeto é em português).
- `.github/workflows/ci.yml`: workflow mínimo rodando `make lint` e `make test` em PRs para `develop` e `main`.
- `CLAUDE.md`: diretrizes operacionais do projeto destiladas deste plano — regra de dependência `adapter → core`, stdlib-first, soft delete obrigatório, VO `Amount`, comandos do Makefile, fluxo de PRs e Conventional Commits. Documento vivo, refinado conforme os padrões se consolidam nas etapas seguintes.
- Commits: `chore: add makefile and golangci-lint config`, `ci: add lint and test workflow`, `docs: add claude.md with project conventions`, `docs: add initial readme`.

### Etapa 1 — Domínio (`feature/domain-model`)
- Entidades `Clinic` (com `BankAccount`), `Dentist` — ambas com `DeletedAt *time.Time` e método `IsDeleted()` (soft delete); VOs `Document` (CPF/CNPJ com dígitos verificadores), `Amount`.
- Erros sentinela de domínio (`errors.go`) para mapeamento idiomático com `errors.Is/As`.
- Construtores validam invariantes (`NewClinic`, `NewDentist`); email/telefone com validação pragmática.
- Testes unitários table-driven do domínio (foco: validação de CPF/CNPJ, casos de erro).
- Commits: `feat(domain): add clinic entity with document validation`, etc.

### Etapa 2 — Ports & Repositórios In-Memory (`feature/in-memory-repository`)
- Interfaces em `core/port` (assinaturas com `context.Context`, retornando erros de domínio).
- Implementações `memory` com `map` + `sync.RWMutex`; índice de documento no repo de clínica (considera apenas registros ativos — documento liberado para recadastro após soft delete); `Delete` marca `deleted_at` (registro preservado no map); `Get`/`List` filtram deletados; `SoftDeleteByClinicID` no repo de dentistas (cascata).
- Testes de integração dos repositórios, incluindo semântica de soft delete (deletado não aparece em Get/List) e teste de concorrência (goroutines simultâneas, validado com `-race`).
- Commits: `feat(port): define repository interfaces`, `feat(memory): implement clinic repository with document index`.

### Etapa 3 — Services / Casos de Uso (`feature/core-services`)
- `ClinicService`: Create (unicidade), Get, List, Update, UpdateBankAccount, Delete (soft delete + cascata soft dos dentistas). Update/UpdateBankAccount de clínica deletada → not found.
- `DentistService`: CRUD validando que a clínica existe e está ativa; operações sobre dentista deletado → not found.
- Testes unitários com **fakes escritos à mão** dos ports (padrão do projeto) — cenários de sucesso e todos os caminhos de erro.
- Commits: `feat(service): add clinic use cases`, `test(service): cover error scenarios with fake repositories`.

### Etapa 4 — Adapter HTTP (`feature/http-api`)
- `router.go` com `http.ServeMux` nativo (Go 1.22+): `mux.HandleFunc("PUT /api/v1/clinics/{clinicID}/bank-account", ...)`, `r.PathValue("clinicID")`.
- DTOs de request/response separados do domínio; mapeador domain error → HTTP status.
- Middlewares: request ID (gera UUID se ausente, injeta no `context` e no `slog` logger do request), logging JSON de acesso, recovery.
- `cmd/api/main.go`: composition root, `slog.NewJSONHandler`, `http.Server` com timeouts e graceful shutdown (`signal.NotifyContext`).
- `docs/openapi.yaml` (OpenAPI 3.0) formalizando o contrato completo da API — clínicas, dentistas e payments; servido pela aplicação via `go:embed` em `GET /openapi.yaml`, com Swagger UI estático em `GET /docs` (página HTML embarcada carregando a UI de CDN — zero dependências Go).
- Testes de integração com `httptest` usando repositórios in-memory reais (fluxo completo: criar clínica → criar dentista → alterar → excluir; erros 400/404/409).
- Commits: `feat(http): add clinic handlers with native servemux`, `feat(http): add request-id and logging middleware`, `docs(api): add openapi specification with embedded swagger ui`.

### Etapa 5 — Pagamentos Pix (`feature/payments-pix`)
- Domínio `Payment` — entidade com identidade própria (aggregate root separado de `Clinic`, referenciando `ClinicID`/`DentistID` por ID): identificador da cobrança, `Amount`, `Status` (`pending`/`approved`), código Pix copia-e-cola, `CreatedAt` e `ApprovedAt`. A transição de status é a única mutação permitida na vida do registro, validada no domínio (`ErrInvalidStatusTransition`); pagamento não tem update nem delete — é histórico financeiro imutável por construção, preservado inclusive após soft delete da clínica.
- Port `PixProvider` para a geração da cobrança.
- Repositório in-memory com índice por clínica para listagem do histórico ordenado por criação; armazena e devolve cópias, mantendo a transição de status sob o write lock — sem data race entre o worker de aprovação e leituras concorrentes.
- `PaymentService.Create` valida que a clínica existe **e está ativa** (não soft-deletada) e, se `dentist_id` informado, que o dentista está ativo e pertence à clínica — nunca criar cobrança para registro deletado.
- Adapter `pixsim`: gera txid e BR Code copia-e-cola simulado (formato inspirado em EMV, sem CRC real).
- Confirmação em background: goroutine por pagamento com delay aleatório 2–5s **injetável** (função `delay func() time.Duration`), respeitando `context` do servidor no shutdown (WaitGroup para não vazar goroutines).
- Endpoints `POST /payments`, `GET /payments/{id}`, `GET /clinics/{id}/payments`.
- Testes: service com provedor fake e delay zero (determinístico); integração aguardando transição com polling curto.
- Commits: `feat(payment): add pix charge creation with simulated confirmation`.

### Etapa 6 — Qualidade & Entrega (`release/1.0.0`)
- `make lint` limpo; cobertura revisada (`make cover`); revisão de docs/comentários godoc e da spec OpenAPI (status codes, campos, exemplos).
- README final: como executar/testar, exemplos curl, decisões arquiteturais e trade-offs, diagrama simples da hexagonal.
- `docs/api.http` com roteiro completo de validação manual.
- PR de `release/1.0.0` → `main` com CI verde; merge e tag `v1.0.0`.
- Commits: `docs: document architecture decisions`, `chore(release): v1.0.0`.

---

## 5. Mapeamento de Testes por Camada

| Camada | Tipo | Dublês | O que cobre |
|---|---|---|---|
| `core/domain` | Unitário puro | nenhum | Validação CPF/CNPJ (dígitos verificadores), Amount, invariantes dos construtores, transição de status do Payment — sucesso e erro |
| `core/service` | Unitário | **fakes manuais** dos ports (repos + PixProvider) | Todos os casos de uso; erros: not found, documento duplicado, clínica inexistente/deletada ao criar dentista, **pagamento para clínica/dentista deletado rejeitado**, valor inválido |
| `adapter/memory` | Integração | nenhum | CRUD real, unicidade de documento, **semântica de soft delete** (deletado invisível em Get/List, cascata), **concorrência com `-race`** |
| `adapter/httpapi` | Integração | repos in-memory reais via `httptest` | Contratos JSON, status codes (200/201/204/400/404/409), 404 para recurso soft-deletado, request ID no header/log, fluxos ponta-a-ponta |
| `adapter/pixsim` | Unitário | delay injetado | Formato do BR Code, transição pending→approved determinística |

Convenções: table-driven tests, `t.Parallel()` onde seguro, `make test` roda `go test -race -cover ./...`. Sem framework de asserção (stdlib pura, coerente com a filosofia "nativo" do projeto).

---

## 6. Análise de Risco / Trade-offs Arquiteturais

| Decisão / Risco | Trade-off | Mitigação |
|---|---|---|
| **Repos separados sem transações** (in-memory) | Check-then-act entre clínica e dentista tem janela de corrida vs. aggregate root com lock único | Invariante crítica (unicidade de documento) resolvida atomicamente dentro do próprio repo; FK validada no service; limitação documentada no README (num banco real: constraint/transação) |
| **Goroutine de confirmação Pix** | Simplicidade vs. risco de leak no shutdown e testes flaky de 2–5s | Delay injetável (zero nos testes), `context` + `sync.WaitGroup` no service, shutdown aguarda workers |
| **`http.ServeMux` nativo** | Zero dependências vs. sem grupos de middleware/binding automático | Wrapping manual de middlewares (3 apenas); decode/validação explícitos por handler — mais verboso, porém transparente |
| **Soft delete + cascata** (dados financeiros) | Preserva auditoria vs. filtros de `deleted_at` em toda leitura e crescimento do storage | Filtro centralizado nos repositórios (única fonte da regra); API trata deletado como 404; documento CPF/CNPJ liberado para recadastro após delete (índice cobre só ativos — decisão documentada) |
| **PUT completo + endpoint dedicado de bank-account** | Menos flexível que PATCH parcial | Espelha o requisito (dados bancários alteráveis de forma independente) e mantém semântica simples |
| **Amount como int64 (centavos)** | Exige conversão na borda | Elimina erros de float em valores monetários |
| **UUID pelo pacote `uuid` da stdlib** (Go 1.27) | Exige toolchain 1.27+ vs. `google/uuid`, que roda em qualquer versão | `go.mod` já fixa `go 1.27.0` e o CI resolve por `go-version-file`; `uuid.New()` é v4, mesma semântica pretendida — o projeto fecha com zero dependências externas |
| **OpenAPI escrito à mão** (vs. geração por anotações, ex. swaggo/swag) | Risco de drift manual entre spec e código vs. anotações verbosas nos handlers, CLI extra e Swagger 2.0 | API pequena (~15 endpoints) e estável; spec revisada a cada etapa e conferida na entrega; UI servida estática sem dependências Go |
| **Hexagonal em projeto pequeno** | Overhead de camadas vs. clareza de fronteiras | Estrutura enxuta (3 pacotes no core) evita over-engineering mantendo os benefícios de testabilidade |

--- 

## Verificação (pós-implementação de cada fase)

1. `make lint` e `make test` (com `-race`) verdes a cada fase.
2. Etapa 4+: `make run` e roteiro manual via `docs/api.http`/curl — criar clínica → criar dentista → alterar bank-account → criar payment → observar `pending→approved` no `GET /payments/{id}` → excluir clínica (soft delete) → confirmar que `GET /clinics/{id}` e dentistas retornam 404 e que novo `POST /payments` para a clínica deletada é rejeitado.
3. Logs JSON no stdout com `request_id` correlacionando as requisições.
4. `GET /docs` abre o Swagger UI e a spec reflete todos os endpoints implementados (validável também colando `/openapi.yaml` no Swagger Editor).
5. `git log --oneline` legível com Conventional Commits; branches seguindo Git Flow; cada etapa integrada via PR com CI verde.
