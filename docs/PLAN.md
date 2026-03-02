# work-schedule — Implementation Plan.

> **Module:** `github.com/veltylabs/work-schedule`
> **Package:** `workschedule`
> **Goal:** Implement the `GetWorkSchedule` handler. Reads from legacy tables `staff` and `workcalendar`. **NO DDL** — these tables preexist and must NEVER be modified.
>
> This module is a standalone Lego piece. It has no knowledge of which application will use it.

---

## Development Rules

- **SRP:** Every file has a single, well-defined purpose.
- **DI:** DB injected via `*orm.DB`. No global state.
- **Flat structure:** All files in repo root — no subdirectories.
- **Max 500 lines per file.**
- **Testing:** `gotest` (not `go test`). Mock all external interfaces. DDT.
- **ORM:** `tinywasm/orm` + `ormc` code generator. Run `ormc` from repo root.
- **Time:** Use `github.com/tinywasm/time`. NEVER use standard `time` package.
- **Errors:** `tinywasm/fmt` only — Noun+Adjective word order.

### Installation Prerequisites

```bash
go install github.com/tinywasm/devflow/cmd/gotest@latest
go install github.com/tinywasm/orm/cmd/ormc@latest
```

---

## go.mod Dependencies

```bash
go get github.com/tinywasm/orm@latest
go get github.com/tinywasm/fmt@latest
go get github.com/tinywasm/sqlite@latest
```

---

## Legacy Schema Reference

### Table: `staff` (LEGACY — READ-ONLY, do NOT touch DDL)

```sql
CREATE TABLE staff (
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    role          VARCHAR(255) NOT NULL,
    email         VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255)  -- NEVER expose via ORM — use db:"-"
);
```

### Table: `workcalendar` (LEGACY — READ-ONLY, do NOT touch DDL)

```sql
CREATE TABLE workcalendar (
    id          SERIAL PRIMARY KEY,
    staff_id    INTEGER NOT NULL REFERENCES staff(id),
    day_of_week SMALLINT NOT NULL,  -- 0=Sunday … 6=Saturday
    start_time  TIME NOT NULL,      -- stored as string "HH:MM" in ORM
    end_time    TIME NOT NULL,      -- stored as string "HH:MM" in ORM
    is_active   BOOLEAN NOT NULL
);
```

## ⚠️ Critical Constraint

```
NEVER call db.CreateTable() for Staff or WorkCalendar.
NEVER call db.DropTable() for Staff or WorkCalendar.
The ORM is used READ-ONLY for these structs.
```

---

## Handler Contract: `GetWorkSchedule`

- **Input:**
```json
{ "staff_id": 42 }
```

- **Output:**
```json
{
  "staff_name": "Dra. Ana González",
  "staff_role": "Médico General",
  "schedule": [
    { "day": 1, "day_name": "Lunes",     "is_active": true, "start": "09:00", "end": "13:00" },
    { "day": 3, "day_name": "Miércoles", "is_active": true, "start": "14:00", "end": "18:00" }
  ]
}
```

- **Errors:**

| Condition | Error |
|---|---|
| `staff_id` missing or non-integer | `fmt.Err("params", "invalid")` |
| Staff not found | `fmt.Err("staff", "not", "found")` |
| DB unreachable | propagate `err` |

---

## Files to Create

| File | Action | Purpose |
|---|---|---|
| `model.go` | Create | `Staff` + `WorkCalendar` structs (legacy schema mapping) |
| `model_orm.go` | Generate | Run `ormc` — do NOT hand-write |
| `mcp.go` | Create | `Module` struct + exported handler method |
| `mcp_test.go` | Create | Black-box tool tests with mock DB |

---

## Step 1 — Models (`model.go`)

```go
//go:build !wasm

package workschedule

// Staff maps to the legacy 'staff' table. READ-ONLY — no DDL allowed.
type Staff struct {
    ID           int64  `db:"pk"`
    Name         string `db:"not_null"`
    Role         string `db:"not_null"`
    Email        string `db:"unique"`
    PasswordHash string `db:"-"` // NEVER expose via ORM
}

func (s *Staff) TableName() string { return "staff" }

// WorkCalendar maps to the legacy 'workcalendar' table. READ-ONLY — no DDL allowed.
type WorkCalendar struct {
    ID        int64  `db:"pk"`
    StaffID   int64  `db:"ref=staff,not_null"`
    DayOfWeek int    `db:"not_null"` // 0=Sunday … 6=Saturday
    StartTime string `db:"not_null"` // "HH:MM"
    EndTime   string `db:"not_null"` // "HH:MM"
    IsActive  bool   `db:"not_null"`
}

func (w *WorkCalendar) TableName() string { return "workcalendar" }
```

**Generate ORM code:**
```bash
# From repo root:
ormc
# Creates: model_orm.go
```

---

## Step 2 — Module (`mcp.go`)

```go
//go:build !wasm

package workschedule

import (
    "context"

    "github.com/tinywasm/fmt"
    "github.com/tinywasm/orm"
)

type Module struct {
    db *orm.DB
}

func New(db *orm.DB) *Module { return &Module{db: db} }

// GetWorkSchedule returns the work calendar for a staff member.
// Signature matches ToolHandler: func(context.Context, map[string]any) (any, error)
func (m *Module) GetWorkSchedule(ctx context.Context, args map[string]any) (any, error) {
    raw, ok := args["staff_id"]
    if !ok {
        return nil, fmt.Err("params", "invalid") // EN: Params Invalid
    }
    staffID, err := fmt.Convert(raw).Int64()
    if err != nil {
        return nil, fmt.Err("params", "invalid") // EN: Params Invalid
    }

    staffRow, err := ReadOneStaff(
        m.db.Query(&Staff{}).Where(StaffMeta.ID).Eq(staffID),
        &Staff{},
    )
    if err != nil || staffRow == nil {
        return nil, fmt.Err("staff", "not", "found") // EN: Staff Not Found
    }

    calRows, err := ReadAllWorkCalendar(
        m.db.Query(&WorkCalendar{}).
            Where(WorkCalendarMeta.StaffID).Eq(staffID).
            OrderBy(WorkCalendarMeta.DayOfWeek).Asc(),
    )
    if err != nil {
        return nil, err
    }

    return buildStaffResponse(staffRow, calRows), nil
}

var dayNames = [7]string{"Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"}

type scheduleEntry struct {
    Day      int    `json:"day"`
    DayName  string `json:"day_name"`
    IsActive bool   `json:"is_active"`
    Start    string `json:"start,omitempty"`
    End      string `json:"end,omitempty"`
}

type staffResponse struct {
    StaffName string          `json:"staff_name"`
    StaffRole string          `json:"staff_role"`
    Schedule  []scheduleEntry `json:"schedule"`
}

func buildStaffResponse(s *Staff, rows []*WorkCalendar) staffResponse {
    entries := make([]scheduleEntry, len(rows))
    for i, r := range rows {
        e := scheduleEntry{
            Day:      r.DayOfWeek,
            DayName:  dayNames[r.DayOfWeek],
            IsActive: r.IsActive,
        }
        if r.IsActive {
            e.Start = r.StartTime
            e.End   = r.EndTime
        }
        entries[i] = e
    }
    return staffResponse{StaffName: s.Name, StaffRole: s.Role, Schedule: entries}
}
```

---

## Step 3 — Tests (`mcp_test.go`)

Integration tests using `github.com/tinywasm/sqlite`. Since the schema is "legacy", we recreate it in the in-memory test DB.

```go
func setupTestModule(t *testing.T) *Module {
    db, _ := sqlite.Open(":memory:")
    db.CreateTable(&Staff{})
    db.CreateTable(&WorkCalendar{})
    return New(db)
}
```

```
TestGetWorkSchedule_ValidStaff
  - Seed 1 Staff + 3 WorkCalendar rows
  - Assert staff_name, staff_role populated correctly
  - Assert schedule has 3 entries with correct day names
  - Assert active entries have start/end; inactive entries omit them

TestGetWorkSchedule_StaffNotFound
  - Seed staffID 1, but call for staffID 99
  - Assert error contains "staff not found"

TestGetWorkSchedule_MissingParam
  - args = {} (no staff_id)
  - Assert error contains "invalid params"

TestGetWorkSchedule_InvalidParam
  - args = {"staff_id": "not-a-number"}
  - Assert error contains "invalid params"

TestGetWorkSchedule_DBFailure
  - Drop table staff before call
  - Assert handler returns error (not panics)
```

```bash
gotest -run TestGetWorkSchedule
```

---

## Checklist

- [ ] `model.go` — `Staff.PasswordHash` tagged `db:"-"` (NEVER in schema/values)
- [ ] `model.go` — `TableName()` returns `"staff"` and `"workcalendar"` respectively
- [ ] `ormc` run from repo root — `model_orm.go` generated (contains `ReadOneStaff`, `ReadAllWorkCalendar`, `StaffMeta`, `WorkCalendarMeta`)
- [ ] `mcp.go` — NO import of `mjosefa-cms` or any application package
- [ ] `mcp.go` — `GetWorkSchedule` is exported and matches signature `func(context.Context, map[string]any) (any, error)`
- [ ] `mcp.go` — does NOT call `db.CreateTable()` for staff or workcalendar
- [ ] `fmt.Convert(raw).Int64()` used for staff_id extraction
- [ ] All 5 test cases pass: `gotest -run TestGetWorkSchedule`
- [ ] `go build ./...` succeeds
- [ ] `gopush 'implement GetWorkSchedule handler'`
