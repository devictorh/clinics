# CLAUDE.md

Diretrizes operacionais para desenvolvimento assistido neste repositório. O roadmap completo e as justificativas das decisões estão em `PLAN.md` — este arquivo é a destilação prática.

## Projeto

API REST em Go para gestão de clínicas odontológicas: CRUD de clínicas (com dados bancários), dentistas vinculados a clínicas e recebimento de pagamentos Pix simulados.

## Arquitetura (regras invioláveis)

- **Hexagonal**: dependências fluem `adapter → core`, nunca o contrário.
- `internal/core/*` importa apenas stdlib (+ `google/uuid`, única dependência externa permitida).
- Handlers HTTP dependem de interfaces de service; services dependem de ports (`internal/core/port`); adapters implementam ports.
- DTOs de request/response vivem no adapter HTTP, nunca no domínio.

## Regras de negócio fixas

- **Soft delete sempre** (`DeletedAt *time.Time`): nada é removido fisicamente; leituras filtram deletados; a API responde 404 para registro deletado. Excluir clínica soft-deleta seus dentistas em cascata.
- Documento (CPF/CNPJ) com validação de dígitos verificadores; único entre clínicas **ativas** (liberado para recadastro após soft delete).
- Valores monetários: VO `Amount`, `int64` em centavos — nunca float.
- Dentista existe apenas vinculado a uma clínica (sub-recurso na API).
- Pagamento só pode ser criado para clínica/dentista ativos.
- Status de pagamento: `pending → approved`, transição validada no domínio.

## Stack e convenções

- HTTP: `net/http` nativo com `http.ServeMux` (Go 1.22+, métodos e path params nos patterns). Sem frameworks.
- Logs: `log/slog` com handler JSON; `request_id` propagado via `context.Context`.
- Erros: sentinelas de domínio + `errors.Is/As`; envelope JSON consistente na API.
- Testes: stdlib pura (sem testify), table-driven, `t.Parallel()` onde seguro; fakes manuais dos ports nos testes de service.
- Contrato da API: `docs/openapi.yaml` (OpenAPI 3.0 escrito à mão) é a fonte da verdade; mudou o contrato, o YAML muda no mesmo commit.
- Comentários e documentação em português; godoc em todos os pacotes e símbolos exportados.

## Diretrizes de Comentários no Código

- **Foque no "Porquê", não no "O quê":** Comente apenas para explicar decisões complexas, regras de negócio ou comportamentos não intuitivos. Nunca comente o que o código já deixa evidente.
- **Proibido contexto de controle de versão:** Jamais insira comentários como "Implementado na feature X", "Alterado por Y" ou datas. O histórico do Git cumpre esse papel.
- **Evite o óbvio:** Não adicione comentários redundantes (ex: `// Incrementa x` sobre `x++` ou `// Função para buscar usuário` sobre `getUser()`).
- **Código limpo é a prioridade:** Se o código precisa de um comentário longo para explicar *o que* ele faz, prefira refatorá-lo usando nomes claros para variáveis e funções.

## Comandos

```
make build      # compila em bin/api
make run        # executa a API
make test       # testes com -race e cobertura
make test-race  # testes com -race, sem cache
make cover      # relatório de cobertura
make lint       # golangci-lint (instala pinado em bin/ se necessário)
```

`make lint` e `make test` devem passar antes de qualquer commit.

## Git

- Git Flow: `feature/*` parte de `develop` e volta via PR; `release/*` → `main` via PR. `main` é protegida — nunca commitar direto em `main` ou `develop`.
- Conventional Commits (`feat:`, `fix:`, `docs:`, `chore:`, `test:`, `ci:`, `refactor:`); um commit por unidade lógica.
- Merges com merge commit (`--no-ff`), sem squash — o histórico da branch é preservado.
