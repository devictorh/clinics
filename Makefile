GOLANGCI_LINT         := bin/golangci-lint
GOLANGCI_LINT_VERSION := v2.13.2

.PHONY: build run test test-race cover lint clean

## build: compila a API em bin/api
build:
	go build -o bin/api ./cmd/api

## run: executa a API
run:
	go run ./cmd/api

## test: roda todos os testes com race detector e cobertura
test:
	go test -race -cover ./...

## test-race: roda todos os testes com race detector, sem cache
test-race:
	go test -race -count=1 ./...

## cover: gera e exibe o relatório de cobertura
cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## lint: roda a análise estática (instala o golangci-lint pinado se necessário)
lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run

$(GOLANGCI_LINT):
	GOBIN=$(CURDIR)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

## clean: remove binários e relatórios
clean:
	rm -rf bin coverage.out coverage.html
