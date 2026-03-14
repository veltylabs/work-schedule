# work-schedule — LLM Skill Summary

## Purpose
Read-only adapter over legacy `staff` and `workcalendar` tables.
Exposes a single MCP tool (`get_work_schedule`) that returns a staff member's weekly schedule.

## Key Files
| File | Role |
|------|------|
| `model.go` | `Staff` + `WorkCalendar` structs, both `TableName()` methods |
| `model_orm.go` | Auto-generated ORM helpers — DO NOT EDIT |
| `mcp.go` | `Module`, `New(db)`, `GetMCPTools()`, `RegisterTools()`, `GetWorkSchedule()`, `buildStaffResponse()` |
| `mcp_test.go` | All tests (`workschedule` package, `:memory:` SQLite) |

## Constraints
- **No DDL.** Never call `db.CreateTable()` for `staff` or `workcalendar`.
- `password_hash` has `db:"-"` — it is never read or written by ORM.
- `staff_id` from MCP args must be converted via `fmt.Convert(raw).Int64()` (handles float64 from JSON).
- `start`/`end` omitted from response when `is_active = false`.

## MCP Registration
```go
m := workschedule.New(db)
m.RegisterTools(srv) // srv is *mcp.MCPServer
```
