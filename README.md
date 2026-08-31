# Clinics API

API REST em Go para gestão de clínicas odontológicas: cadastro de clínicas com dados bancários, dentistas vinculados e recebimento de pagamentos via Pix (simulado).

> 🚧 Em desenvolvimento — o roadmap de implementação está em [PLAN.md](PLAN.md).

## Requisitos

- Go 1.27+
- Make

## Como executar

```bash
make run
```

## Testes e qualidade

```bash
make test   # testes com race detector e cobertura
make cover  # relatório de cobertura por função
make lint   # análise estática (golangci-lint)
```

## Arquitetura

O projeto segue Arquitetura Hexagonal (Ports & Adapters):

```
internal/
├── core/        # domínio, ports e casos de uso — sem dependências de infraestrutura
└── adapter/     # HTTP (driving), persistência in-memory e provedor Pix simulado (driven)
```

A documentação completa das decisões arquiteturais e trade-offs será consolidada aqui ao final do desenvolvimento.
