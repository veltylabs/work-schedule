# work-schedule — Enhancement Plan (ToolProvider Self-Registration)

> **Goal:** Implement `mcp.ToolProvider` so the module self-registers its MCP tools
> via `srv.RegisterProvider(m)`.
>
> **Note:** This module reads LEGACY tables (`workcalendar`, `staff`) that are READ-ONLY.
> No `db.CreateTable()` call — migration is managed externally.
>
> **Depends on:** `tinywasm/mcp` `RegisterProvider` + fixed `ToolExecutor` (see `tinywasm/mcp/docs/PLAN.md`).
> **Status:** Pending execution

---

## Development Rules

- **Testing Runner:** `go install github.com/tinywasm/devflow/cmd/gotest@latest`
- **Build Tag:** All backend files must use `//go:build !wasm`.
- **No log injection:** The module receives only `db *orm.DB`. No log parameter in `New()`.
- **Read-Only:** No DDL operations. No `db.CreateTable()` for legacy tables.

---

## Step 1 — Implement `ToolProvider`

**Target File:** `mcp.go`

Add `GetMCPToolsMetadata()` to make `*Module` implement `mcp.ToolProvider`.
The `Execute` field points directly to the existing handler method — no adapter needed.

```go
func (m *Module) GetMCPToolsMetadata() []mcp.ToolMetadata {
    return []mcp.ToolMetadata{
        {
            Name:        "get_work_schedule",
            Description: "Returns the work calendar for a staff member.",
            Parameters: []mcp.ParameterMetadata{
                {
                    Name:        "staff_id",
                    Description: "The staff member's numeric ID.",
                    Required:    true,
                    Type:        "number",
                },
            },
            Execute: m.GetWorkSchedule,
        },
    }
}
```

---

## Step 2 — Add `RegisterTools`

**Target File:** `mcp.go`

```go
// RegisterTools registers all work-schedule MCP tools on the given server.
// Call once during application startup after New(db).
func (m *Module) RegisterTools(srv *mcp.MCPServer) {
    srv.RegisterProvider(m)
}
```

---

## Step 3 — Update Tests

- Update `mcp_test.go` to verify `GetMCPToolsMetadata()` returns the expected tool
  names and parameter schema.
- Run `gotest` — 100% pass required.

---

## Step 4 — Verify & Submit

1. Run `gotest` from project root.
2. Run `gopush 'feat: ToolProvider self-registration'`
