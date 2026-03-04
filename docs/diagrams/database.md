# work-schedule — Database Diagram

```mermaid
flowchart TD
    S[staff — LEGACY READ-ONLY]
    S --> S1[id: int64 PK]
    S --> S2[name: string NOT NULL]
    S --> S3[role: string NOT NULL]
    S --> S4[email: string UNIQUE]
    S --> S5[password_hash: string excluded from ORM]

    W[workcalendar — LEGACY READ-ONLY]
    W --> W1[id: int64 PK]
    W --> W2[staff_id: int64 NOT NULL]
    W --> W3[day_of_week: int NOT NULL<br/>0=Sunday … 6=Saturday]
    W --> W4[start_time: string NOT NULL HH:MM]
    W --> W5[end_time: string NOT NULL HH:MM]
    W --> W6[is_active: bool NOT NULL]

    W2 --> S1
```

> **No DDL allowed.** Tables are owned by the legacy system.
> Read strategy: first fetch `staff` by `staff_id`, then fetch all `workcalendar`
> rows for that staff ordered by `day_of_week ASC`.
