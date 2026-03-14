# work-schedule Architecture

## 1. Domain Scope
Provides a read-only view of the weekly work schedule for clinic staff (doctors, nurses, etc.).
Reads from two legacy tables maintained by an external system.

## 2. Core Entities
- **Staff:** The clinic employee. `password_hash` is excluded from ORM output (`db:"-"`).
- **WorkCalendar:** One row per staff member per working day. `is_active = false` indicates
  a non-working day; `start_time`/`end_time` are meaningful only when `is_active = true`.

## 3. Architectural Patterns
1. **Read-Only / No DDL:** This module does NOT call `db.CreateTable()`. Tables are managed
   by the legacy system. `New(db)` is side-effect free apart from storing the DB reference.
2. **Dependency Injection:** `New(db *orm.DB)` returns `*Module`. No global state.
3. **MCP Self-Registration:** `*Module` implements `mcp.ToolProvider` via `GetMCPTools()`.
   Registered via `RegisterTools(srv)`.
4. **Safe Conversion:** `staff_id` from MCP args is converted via `fmt.Convert(raw).Int64()`
   to safely handle both `float64` (JSON default) and `int64` inputs.

## 4. MCP Tools
| Tool | Parameters | Returns |
|------|-----------|---------|
| `get_work_schedule` | `staff_id` (number, required) | `staff_name`, `staff_role`, `schedule[]` (day, day_name, is_active, start, end) |

Spanish day names: Domingo, Lunes, Martes, Miércoles, Jueves, Viernes, Sábado.
`start`/`end` fields are omitted when `is_active = false`.

## 5. Schema
See [`docs/diagrams/database.md`](diagrams/database.md).
