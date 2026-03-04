package workschedule

import (
	"context"
	"strings"
	"testing"

	"github.com/tinywasm/mcp"
	"github.com/tinywasm/sqlite"
)

func setupTestModule(t *testing.T) *Module {
	db, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	db.CreateTable(&Staff{})
	db.CreateTable(&WorkCalendar{})
	return New(db)
}

func TestGetWorkSchedule_ValidStaff(t *testing.T) {
	m := setupTestModule(t)

	// Seed Staff
	err := m.db.Create(&Staff{
		ID:           1,
		Name:         "Dra. Ana González",
		Role:         "Médico General",
		Email:        "ana@example.com",
		PasswordHash: "hash123",
	})
	if err != nil {
		t.Fatalf("failed to seed staff: %v", err)
	}

	// Seed WorkCalendar
	calendars := []*WorkCalendar{
		{ID: 1, StaffID: 1, DayOfWeek: 1, StartTime: "09:00", EndTime: "13:00", IsActive: true}, // Monday
		{ID: 2, StaffID: 1, DayOfWeek: 3, StartTime: "14:00", EndTime: "18:00", IsActive: true}, // Wednesday
		{ID: 3, StaffID: 1, DayOfWeek: 5, StartTime: "", EndTime: "", IsActive: false},          // Friday
	}
	for _, c := range calendars {
		err := m.db.Create(c)
		if err != nil {
			t.Fatalf("failed to seed work calendar: %v", err)
		}
	}

	args := map[string]any{"staff_id": int64(1)}
	res, err := m.GetWorkSchedule(context.Background(), args)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	staffRes, ok := res.(staffResponse)
	if !ok {
		t.Fatalf("expected staffResponse type, got %T", res)
	}

	if staffRes.StaffName != "Dra. Ana González" {
		t.Errorf("expected staff_name 'Dra. Ana González', got %q", staffRes.StaffName)
	}
	if staffRes.StaffRole != "Médico General" {
		t.Errorf("expected staff_role 'Médico General', got %q", staffRes.StaffRole)
	}

	if len(staffRes.Schedule) != 3 {
		t.Fatalf("expected 3 schedule entries, got %d", len(staffRes.Schedule))
	}

	expectedNames := []string{"Lunes", "Miércoles", "Viernes"}
	for i, e := range staffRes.Schedule {
		if e.DayName != expectedNames[i] {
			t.Errorf("expected day name %q, got %q", expectedNames[i], e.DayName)
		}
		if e.IsActive {
			if e.Start == "" || e.End == "" {
				t.Errorf("expected active entry to have start/end times")
			}
		} else {
			if e.Start != "" || e.End != "" {
				t.Errorf("expected inactive entry to omit start/end times")
			}
		}
	}
}

func TestGetWorkSchedule_StaffNotFound(t *testing.T) {
	m := setupTestModule(t)

	// Seed staffID 1
	err := m.db.Create(&Staff{
		ID:           1,
		Name:         "Dra. Ana González",
		Role:         "Médico General",
		Email:        "ana@example.com",
		PasswordHash: "hash123",
	})
	if err != nil {
		t.Fatalf("failed to seed staff: %v", err)
	}

	// Call for staffID 99
	args := map[string]any{"staff_id": int64(99)}
	_, err = m.GetWorkSchedule(context.Background(), args)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "staff not found") {
		t.Errorf("expected error containing 'staff not found', got %q", err.Error())
	}
}

func TestGetWorkSchedule_MissingParam(t *testing.T) {
	m := setupTestModule(t)

	args := map[string]any{} // no staff_id
	_, err := m.GetWorkSchedule(context.Background(), args)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid params") && !strings.Contains(err.Error(), "params invalid") {
		t.Errorf("expected error containing 'invalid params' or 'params invalid', got %q", err.Error())
	}
}

func TestGetWorkSchedule_InvalidParam(t *testing.T) {
	m := setupTestModule(t)

	args := map[string]any{"staff_id": "not-a-number"}
	_, err := m.GetWorkSchedule(context.Background(), args)

	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid params") && !strings.Contains(err.Error(), "params invalid") {
		t.Errorf("expected error containing 'invalid params' or 'params invalid', got %q", err.Error())
	}
}

func TestGetWorkSchedule_DBFailure(t *testing.T) {
	m := setupTestModule(t)

	// Drop table staff before call to simulate DB failure
	m.db.DropTable(&Staff{})

	args := map[string]any{"staff_id": int64(1)}

	// Should return error, not panic
	_, err := m.GetWorkSchedule(context.Background(), args)
	if err == nil {
		t.Fatalf("expected error due to DB failure, got nil")
	}
}

func TestGetMCPToolsMetadata(t *testing.T) {
	m := setupTestModule(t)

	metadata := m.GetMCPToolsMetadata()
	if len(metadata) != 1 {
		t.Fatalf("expected 1 tool metadata, got %d", len(metadata))
	}

	tool := metadata[0]
	if tool.Name != "get_work_schedule" {
		t.Errorf("expected tool name 'get_work_schedule', got %q", tool.Name)
	}
	if tool.Description != "Returns the work calendar for a staff member." {
		t.Errorf("expected correct description, got %q", tool.Description)
	}

	if len(tool.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(tool.Parameters))
	}

	param := tool.Parameters[0]
	if param.Name != "staff_id" {
		t.Errorf("expected param name 'staff_id', got %q", param.Name)
	}
	if param.Type != "number" {
		t.Errorf("expected param type 'number', got %q", param.Type)
	}
	if !param.Required {
		t.Errorf("expected param to be required")
	}

	if tool.Execute == nil {
		t.Errorf("expected tool.Execute to be assigned")
	}

	// Verify that RegisterTools does not panic
	srv := mcp.NewMCPServer("test", "1.0")
	m.RegisterTools(srv)
}
