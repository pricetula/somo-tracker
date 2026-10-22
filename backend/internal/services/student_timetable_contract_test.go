package services

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestStudentTimetableSlotJSONContract verifies the JSON response contract
// for GetStudentTimetable — all fields must be snake_case per frontend contract.
func TestStudentTimetableSlotJSONContract(t *testing.T) {
	slot := StudentTimetableSlot{
		DayOfWeek:    1,
		TimeSlotName: "08:00 - 09:00",
		StartTime:    "08:00",
		EndTime:      "09:00",
		SubjectName:  "Mathematics",
		TeacherName:  "Ada Lovelace",
		RoomName:     "101",
		ClassName:    "Grade 7-A",
		GradeName:    "Grade 7",
	}

	b, err := json.Marshal(slot)
	require.NoError(t, err)

	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &m))

	// Contract: all keys must be snake_case (frontend expects these)
	require.Contains(t, m, "day_of_week")
	require.Contains(t, m, "time_slot_name")
	require.Contains(t, m, "start_time")
	require.Contains(t, m, "end_time")
	require.Contains(t, m, "subject_name")
	require.Contains(t, m, "teacher_name")
	require.Contains(t, m, "room_name")
	require.Contains(t, m, "class_name")
	require.Contains(t, m, "grade_name")

	// Must NOT contain PascalCase keys (old bug)
	require.NotContains(t, m, "DayOfWeek")
	require.NotContains(t, m, "TimeSlotName")

	// Verify values preserved
	require.Equal(t, float64(1), m["day_of_week"])
	require.Equal(t, "Mathematics", m["subject_name"])
}
