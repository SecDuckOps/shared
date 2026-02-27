# DuckOps Shared — Architecture Overview & Development Guide

---

## 1️⃣ Executive Overview

The **DuckOps Shared** module contains the foundational, cross-cutting concerns for the entire DuckOps ecosystem (Agent and Server).

This document explains:

- The main parts of the shared module
- How they relate to each other
- Why the architecture is designed this way
- Extracted architectural patterns
- A clear development guide for future work

The goal is to provide a **mental model** of the shared libraries so any contributor can understand them quickly and extend them safely.

---

## 2️⃣ High-Level Architecture

The system follows a strict modular architecture with clear functional boundaries.

### Core Packages

1. **Primitive Layer**
   - `types`: Core domain structures (AppError, ErrorCodes) with zero dependencies.
   - `ports`: Common interface definitions.

2. **Communication Layer**
   - `events`: Event definitions for RabbitMQ pub/sub.
   - `proto`: Initial gRPC definitions and generated stubs.
   - `protocol`: Base communication contracts.

3. **Utility Layer**
   - `logger`: Architecturally pure logging abstraction over Zap.
   - `secrets`: Secret management primitives.
   - `client`: Base client abstractions.

4. **AI/LLM Layer**
   - `llm`: Unified registry for AI model interaction, focusing on Structured Output enforcement.

---

## 3️⃣ Main Components & Responsibilities

### 3.1 Error System (`types/`)

Responsible for:

- Providing a structured, traceable error system (`AppError`).
- Eliminating raw strings in favor of machine-readable error codes.

Why it exists:

- Allows frontends and logs to categorize errors deterministically.
- Zero dependencies, used universally.

### 3.2 Logger (`logger/`)

Responsible for:

- Structured, JSON-based logging via Uber Zap.
- Domain-agnostic logging via the `ports.Logger` interface.
- Correlation ID extraction.

Why separate?

- Prevents the underlying logging implementation (Zap) from leaking into domain logic across the Agent and Server.

### 3.3 LLM Registry (`llm/`)

Responsible for:

- Managing LLM provider lifecycle (OpenAI, OpenRouter, Gemini, Anthropic).
- Enforcing structured generation (JSON marshaling).
- Stripping markdown tags from responses.

Relationship:

- Used extensively by the Server's Prompt Engine and Remediation services, and the Agent's analytical tools.

### 3.4 Events & Proto (`events/`, `proto/`)

Responsible for:

- Defining schemas for inter-service communication.
- `events/` defines the Go structs used over RabbitMQ.
- `proto/` contains Protocol Buffers and generated Go stubs for gRPC control plane calls.

---

## 4️⃣ Dependency Direction (Critical Rule)

All dependencies within the DuckOps ecosystem (Agent, Server) depend on `shared/`.
**`shared/` must never depend on the Agent or Server.**

Internal `shared/` rules:
`shared/types` → Depends on NOTHING.
`shared/ports` → Depends on NOTHING.
`shared/logger` → Depends on `ports` and `types`.
`shared/llm` → Depends on `types` and `ports`.

**NEVER:**

- Introduce circular dependencies within shared packages.

---

## 5️⃣ Extracted Architectural Patterns

### ✅ 1. Port & Adapter (Hexagonal Base)

The `logger` is a perfect example. We define `ports.Logger`, and provide an adapter implementation wrapping Zap, ensuring the domain layer remains agnostic.

### ✅ 2. Registry Pattern

The `LLMRegistry` manages model instantiation cleanly based on configuration profiles. It prevents scattered dependency creation.

### ✅ 3. Wrapper Pattern

The `AppError` system wraps standard Go errors while injecting rich metadata (code, context, timestamp, correlation IDs).

---

## 6️⃣ Development Guide (Strict Rules)

### Rule 1: Respect Dependencies

Ensure `shared/` is free of business logic specific to the Agent or Server. If a concern is only used by the Orchestrator, it belongs in the Server, not Shared.

### Rule 2: No UI or Protocol Coupling in Types

`types.AppError` uses HTTP-agnostic codes (e.g., `ErrCodeNotFound`, not `404`). Let the HTTP adapters in the server translate these.

### Rule 3: Maintain Compatibility

Updates to `events/` or `types/` affect the entire ecosystem. Exercise extreme care, use versioning when making breaking changes (e.g., in proto).

---

## 7️⃣ How to Add New Features Safely

### Adding a New Shared Event

1. Define the struct in `events/`.
2. Ensure JSON tags are clean and deterministic.
3. Update consumers in Agent/Server after bumping the internal package version.

### Adding a New LLM Provider

1. Implement the provider interface in `llm/infrastructure/`.
2. Register it in `llm/application/registry.go`.
3. Add relevant configuration models.

---

## 8️⃣ Anti-Patterns to Avoid

❌ **Leaking Implementations**: Returning a `*zap.Logger` instead of `ports.Logger`.
❌ **Fat Shared**: Moving application-specific logic into `shared/` because it's "easier".
❌ **Generic Errors**: Using `fmt.Errorf` or `errors.New` instead of `types.New`.

---

## 9️⃣ Testing Strategy

- **Unit Tests**: Test utility isolation. Test the LLM parser against tricky markdown responses.
- **Mocks**: Provide mocks for `ports` interfaces to allow dependent microservices to test against them.

---

## 🔟 Final Philosophy

This library is the bedrock of the system.
Every change must prioritize:

- **Stability** (don't break downstreams)
- **Purity** (no business logic)
- **Performance** (minimize allocations in logging/errors)

END OF GUIDE
