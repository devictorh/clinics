# Clinics API

![CI](https://github.com/devictorh/clinics/actions/workflows/ci.yml/badge.svg)

API REST em Go para gestão de clínicas odontológicas: cadastro de clínicas com dados bancários, dentistas vinculados e recebimento de pagamentos via Pix (simulado).

## Funcionalidades

- **Clínicas** — CRUD completo com documento (CPF ou CNPJ, incluindo o formato alfanumérico vigente desde jul/2026), razão social, nome fantasia e dados bancários alteráveis por endpoint dedicado.
- **Dentistas** — sempre vinculados a uma clínica (sub-recurso na URL), com flag de administrador/responsável legal.
- **Pagamentos Pix** — cobrança com código copia-e-cola no formato BR Code (simulado) e ciclo de status `pending → approved`, confirmado em background após 2–5s, como um webhook de PSP real.
- **Soft delete em tudo que é cadastro** — dados financeiros exigem trilha de auditoria: exclusões marcam `deleted_at`, somem da API (404) e preservam o histórico de pagamentos.
- Logs estruturados JSON (`log/slog`) correlacionados por `X-Request-ID`, graceful shutdown e documentação viva em `/docs` (Swagger UI) e `/openapi.yaml`.

## Como executar

Requisitos: Go 1.27+ e Make.

```bash
make run            # sobe a API em :8080 (configurável via PORT)
```

- Documentação interativa: http://localhost:8080/docs
- Contrato OpenAPI 3.0: http://localhost:8080/openapi.yaml
- Health check: http://localhost:8080/healthz
- Roteiro completo de requisições: [docs/api.http](docs/api.http) (VS Code REST Client)

### Exemplo rápido

```bash
# criar clínica
curl -s -X POST localhost:8080/api/v1/clinics -H 'Content-Type: application/json' \
  -d '{"document":"11.222.333/0001-81","legal_name":"Clínica Sorriso LTDA","trade_name":"Sorriso Odonto"}'

# criar cobrança pix (use o id retornado acima)
curl -s -X POST localhost:8080/api/v1/payments -H 'Content-Type: application/json' \
  -d '{"clinic_id":"<clinic_id>","amount":15000}'

# acompanhar a confirmação simulada (pending → approved em 2–5s)
curl -s localhost:8080/api/v1/payments/<payment_id>
```

Valores monetários são sempre **centavos** (`15000` = R$ 150,00).

## Testes e qualidade

```bash
make test       # suíte completa com -race e cobertura
make cover      # relatório de cobertura por função
make lint       # golangci-lint (instala pinado em bin/ na primeira execução)
```

| Camada | Tipo de teste | Dublês | Cobertura |
|---|---|---|---|
| `core/domain` | Unitário puro | nenhum | 98% |
| `core/service` | Unitário | fakes manuais dos ports | 91% |
| `adapter/memory` | Integração + concorrência | nenhum | 98% |
| `adapter/httpapi` | Integração ponta-a-ponta (`httptest`) | repositórios reais | 87% |
| `adapter/pixsim` | Unitário | delay injetado | 100% |

Toda a suíte roda sob o race detector, incluindo testes de disputa real (criações concorrentes pelo mesmo documento, aprovação de pagamento em paralelo com leituras). A camada de service é testada exclusivamente contra **fakes escritos à mão** das interfaces — nenhum teste depende de banco ou serviço externo.

## Arquitetura

Arquitetura Hexagonal (Ports & Adapters): o core não conhece HTTP nem persistência; adapters dependem do core, nunca o contrário.

```
                    ┌──────────────────────────────────────────┐
 HTTP (driving)     │                  core                    │     driven
┌─────────────┐     │  ┌─────────┐   ┌─────────┐   ┌────────┐  │  ┌──────────────┐
│   httpapi   │ ──▶ │  │ service │ ─▶│  port   │◀─ │ domain │  │  │ memory       │
│ ServeMux    │     │  │ casos   │   │ (inter- │   │ entid. │  │  │ map+RWMutex  │
│ middleware  │     │  │ de uso  │   │  faces) │   │ + VOs  │  │  │──────────────│
│ DTOs        │     │  └─────────┘   └────┬────┘   └────────┘  │  │ pixsim       │
└─────────────┘     │                     │ implementado por   │  │ BR Code sim. │
                    └─────────────────────┼────────────────────┘  └──────▲───────┘
                                          └──────────────────────────────┘
```

```
cmd/api/            composition root: wire manual + graceful shutdown
internal/
├── core/
│   ├── domain/     entidades (Clinic, Dentist, Payment) + VOs (Document, Amount) + erros sentinela
│   ├── port/       interfaces de repositório e do provedor pix
│   └── service/    casos de uso, orquestração e o worker de confirmação
└── adapter/
    ├── httpapi/    router, handlers, DTOs, mapeamento de erros, middlewares
    ├── memory/     repositórios in-memory (map + sync.RWMutex)
    └── pixsim/     provedor pix simulado
docs/               openapi.yaml (contrato, fonte da verdade) + api.http
```

Fluxo de uma requisição: `middleware (request-id → log → recovery) → handler → interface de service → service → ports → adapters`. Handlers dependem de interfaces definidas no consumidor; services recebem ports pelo construtor — injeção de dependência por construtor, com wire manual no `main.go` (grafo pequeno e estático dispensa container de DI).

## Decisões técnicas

### As três mais importantes

1. **Soft delete universal + pagamentos imutáveis.** Sistema que movimenta dinheiro não apaga registro: `DELETE` marca `deleted_at`, leituras filtram (API responde 404) e a exclusão de clínica cascateia para os dentistas. Pagamentos vão além — não têm update nem delete; a única mutação em toda a vida do registro é a transição `pending → approved`, validada no domínio. O histórico financeiro existe por construção e sobrevive à exclusão da clínica.

2. **Stdlib-first com fronteiras hexagonais.** `net/http` puro (ServeMux 1.22+ com métodos e path params), `log/slog`, persistência `map` + `sync.RWMutex` — tudo atrás de ports. Única dependência externa: `google/uuid`. O custo (middlewares e decode manuais) compra transparência total: cada linha do projeto é explicável, e trocar o in-memory por Postgres ou o simulador por um PSP real não toca o core. As invariantes ficam onde conseguem ser atômicas: unicidade de documento dentro do repositório (índice sob o mesmo lock), validações de vínculo no service.

3. **Contrato OpenAPI escrito à mão, servido pela própria aplicação.** O `docs/openapi.yaml` é artefato de design (contract-first), não subproduto de anotações: nasceu com o desenho dos endpoints e cobre o contrato completo. A aplicação o serve via `go:embed` em `/openapi.yaml` com Swagger UI em `/docs`. Geração por anotações (ex.: swaggo/swag) foi avaliada e descartada: não elimina drift (comentários também dessincronizam), gera Swagger 2.0 legado e polui os handlers — numa API de ~15 endpoints estáveis, o manual custa menos e entrega OpenAPI 3.0.

### Outras decisões relevantes

- **`Amount` em centavos (`int64`)** — elimina erros de ponto flutuante; conversão só na borda.
- **CNPJ alfanumérico** — validação de dígitos verificadores cobre o formato novo da RFB (valor do caractere = ASCII−48), verificada contra o exemplo oficial.
- **Repositórios devolvem cópias** — valores, não ponteiros: o worker de aprovação nunca compartilha memória mutável com leitores (provado sob `-race`).
- **Dentista de outra clínica responde 404** — não vaza a existência de registros entre clínicas.
- **PUT completo + endpoint dedicado de dados bancários** — semântica simples e espelha o requisito de alteração independente.
- **Confirmação com delay injetável** — 2–5s em produção, zero nos testes; `Shutdown` encerra os workers sem vazar goroutines, com prioridade para aprovações cujo prazo já venceu (determinismo).

### O que faria diferente com mais tempo

- **CRC16-CCITT real no BR Code** — tornaria o copia-e-cola estruturalmente válido para leitores de QR (seguiria não-pagável, pois a chave é fictícia).
- **Contract testing** — validar as respostas dos testes de integração contra o `openapi.yaml`, tornando o contrato executável; numa API maior, avaliaria `oapi-codegen` (spec-first com tipos gerados).
- **Paginação e filtros** nas listagens; **idempotency key** no `POST /payments`.
- **Persistência real** (Postgres) atrás dos mesmos ports — constraints de unicidade e transações substituiriam as limitações documentadas do in-memory.
- **Observabilidade além de logs** — métricas e tracing (OpenTelemetry).

### Uso de IA no desenvolvimento

O projeto foi desenvolvido com assistência de IA (Claude Code) operando sob diretrizes explícitas versionadas em [CLAUDE.md](CLAUDE.md) — regras de arquitetura, negócio e estilo que o assistente segue em toda sessão. O processo foi de direção ativa, não delegação; os pontos onde a revisão humana mudou o rumo estão no histórico de PRs:

- O plano inicial propunha **hard delete**; a revisão impôs soft delete pela natureza financeira dos dados — decisão que redesenhou repositórios, services e API.
- A primeira validação de CNPJ cobria apenas o formato numérico; a revisão exigiu suporte ao **CNPJ alfanumérico** já vigente.
- Uma versão do value object `Document` gerada por IA veio com `sql.Scanner`, `driver.Valuer` e serializadores — código especulativo, descartado por YAGNI após revisão crítica; do mesmo material, o teste de propriedade dos dígitos verificadores foi aproveitado.
- Sugestões de biblioteca (container de DI, geradores de OpenAPI) foram avaliadas e recusadas quando o custo em dependências e clareza superava o benefício no escopo.

## Processo de desenvolvimento

Git Flow (`feature/* → develop → release/* → main`) com todo merge via Pull Request, título em Conventional Commits, merge sem squash e CI (lint + testes com `-race`) como status check obrigatório. O roadmap executado está em [PLAN.md](PLAN.md).
