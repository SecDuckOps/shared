# 🦆 DuckOps Shared

The core foundational library for the DuckOps DevSecOps ecosystem.

---

## Overview

The `shared` module contains all cross-cutting concerns, domain types, communication protocols, and utility abstractions used heavily by both the **DuckOps Server** and **DuckOps Agent**.

It is designed to be highly modular, completely stateless, and strictly decoupled from any internal business logic of the services that consume it.

---

## Architecture

This module acts as the "Primitive Layer" of our Hexagonal Architecture.

> 📖 Full architecture and dependency rules: [docs/architecture_guide.md](docs/architecture_guide.md)

### Directory Structure

```
shared/
├── docs/                           # Architecture documentation
├── types/                          # Core domain structures (AppError)
├── ports/                          # Common interface definitions
├── logger/                         # Architecturally pure logging abstraction
├── llm/                            # Structured Output LLM Registry
├── events/                         # RabbitMQ Pub/Sub models
├── proto/                          # gRPC definitions & stubs
├── protocol/                       # Base communication contracts
├── secrets/                        # Secret management primitives
└── client/                         # Base client abstractions
```

---

## 🏗️ 1. AppError System (`/types`)

We reject standard Go `error` strings in favor of a structured, traceable error system that carries contextual metadata across service boundaries.

```go
// Usage Strategy
types.New(types.ErrCodeInvalidInput, "missing field")
types.Wrap(err, types.ErrCodeInternal, "db save failed").WithContext("id", 123)
```

## 🪵 2. Architecturally Pure Logger (`/logger`)

A domain-agnostic logging abstraction wrapping Uber Zap.

1. **Correlation ID**: Extracts `correlation_id` from Context automatically.
2. **Level Mapping**: Automatically maps `AppError` codes to appropriate log levels.
3. **Zap Independence**: Usage of `ports.Field{Key, Value}` prevents infrastructure leakage.

## 🤖 3. Shared AI Capability (`/llm`)

A unified registry managing LLM provider lifecycle and enforcing structured JSON extraction.

```go
llm := registry.MustGet("openai")
// GenerateJSON strips markdown, calls the LLM, and unmarshals into the struct.
err := llm.GenerateJSON(ctx, prompt, &myStruct)
```

---

## 🖇️ Dependency Rules

- **`shared/types`**: ZERO dependencies.
- **`shared/ports`**: ZERO dependencies.
- **`shared/logger`**: Depends on `ports` and `types`.
- **`shared/llm`**: Depends on `types` and `ports`.

> ⚠️ **CRITICAL:** `shared` must never import `server` or `agent` packages.

---

## Testing

```bash
go test ./... -cover
```

---

## Contributing

1. **Stability**: Breakages here break the entire ecosystem.
2. **Purity**: No business logic belongs in this repository.
3. **Hexagonal Rules**: Adhere to the Port-and-Adapter separation for all external boundaries (like logging and LLMs).

---

_DuckOps Shared: Engineering Consistency._ 🦆
