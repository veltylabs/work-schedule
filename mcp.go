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

	staffModel := &Staff{}
	staffRow, err := ReadOneStaff(
		m.db.Query(staffModel).Where(StaffMeta.ID).Eq(staffID),
		staffModel,
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
		dayIndex := r.DayOfWeek
		if dayIndex < 0 || dayIndex > 6 {
			dayIndex = 0 // fallback or handle error if needed, but safe bounds
		}
		e := scheduleEntry{
			Day:      r.DayOfWeek,
			DayName:  dayNames[dayIndex],
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
