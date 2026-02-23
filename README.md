# DuckOps Shared

Shared proto definitions and Go types used by both `agent` and `server`.

## Contents

- `proto/` — gRPC service definitions
- `types/` — Shared Go types (scan events, error codes, etc.)

## Usage

```go
import "github.com/duckops/dshared/types"
import pb "github.com/duckops/shared/proto/gen"
```

## Regenerate Proto

```bash
make proto
```
# shared
