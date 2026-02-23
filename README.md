# DuckOps Shared: The Core Foundation

The `shared` repository provides the foundational cross-cutting concerns for the entire DuckOps ecosystem. It is designed to be highly modular, stateless, and strictly decoupled from internal business logic.

---

## 🏗️ 1. AppError System (`/types`)

We reject standard Go `error` strings in favor of a structured, traceable error system.

### **The AppError Struct**

```go
type AppError struct {
    Code      ErrorCode              // Machine-readable (e.g., ErrCodeInternal)
    Message   string                 // Human-readable context
    Context   map[string]interface{} // Dynamic metadata
    Timestamp time.Time              // When it happened
    Cause     error                  // Wrapped underlying error
}
```

### **Usage Strategy**

- **Creation**: `types.New(types.ErrCodeInvalidInput, "missing field")`
- **Wrapping**: `types.Wrap(err, types.ErrCodeInternal, "db save failed").WithContext("id", 123)`
- **Assertion**: Use `errors.As(err, &appErr)` to extract metadata for logging or UI.

---

## 🪵 2. Architecturally Pure Logger (`/logger`)

The logger implementation uses **Uber Zap** but ensures the domain layer remains agnostic through a port abstraction.

### **Abstraction (Port)**

```go
type Logger interface {
    Debug(ctx context.Context, msg string, fields ...Field)
    Info(ctx context.Context, msg string, fields ...Field)
    ErrorErr(ctx context.Context, err error, msg string, fields ...Field)
}
```

### **Key Features**

1.  **Correlation ID**: Automatically extracts `correlation_id` from `context.Context` to link logs across distributed systems.
2.  **Level Mapping**: Automatically maps `AppError` codes to log levels:
    - `ErrCodeInvalidInput` / `ErrCodeNotFound` → **Warn**
    - `ErrCodeInternal` / `ErrCodeAgentFailed` → **Error**
3.  **Zap Independence**: All fields are passed as `ports.Field{Key, Value}`, preventing `zap` from leaking into your UseCases.

---

## 🤖 3. Shared AI Capability Layer (`/llm`)

A unified registry for AI model interaction, focusing on **Structured Output enforcement**.

### **Registry Pattern**

The `LLMRegistry` manages model lifecycle and configuration.

```go
sharedCfg := llmdomain.Config{
    Default: "openai",
    Providers: map[string]ProviderConfig{...}
}
registry, _ := llmapp.NewLLMRegistry(sharedCfg)
llm := registry.MustGet("openai")
```

### **GenerateJSON (The Magic)**

The `GenerateJSON` method centralizes complex AI output handling:

1.  Calls the model with the provided prompt.
2.  **Markdown Stripping**: Automatically removes ` ```json ` blocks.
3.  **Validation**: Unmarshals directly into your target `interface{}` struct.
4.  **Error Mapping**: Wraps failures into `ErrCodeInvalidInput` if the AI response misses the schema.

---

## 🖇️ Dependency Rules

- **shared/types**: ZERO dependencies (used by everyone).
- **shared/llm**: Depends on `shared/types`.
- **shared/logger**: Depends on `shared/ports` and `shared/types`.

---

_DuckOps Shared: Engineering Consistency._
