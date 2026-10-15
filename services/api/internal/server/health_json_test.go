package server

import (
	"encoding/json"
	pb "github.com/architsangal/AteThat/services/api/internal/gen/atethat/v1"
	"testing"
)

func TestPausedScheduleJSONUsesEmptyArrays(t *testing.T) {
	data, err := json.Marshal(healthItemJSON(&pb.HealthItem{Schedule: &pb.HealthSchedule{Frequency: "PAUSED"}}))
	if err != nil {
		t.Fatal(err)
	}
	var item struct {
		Schedule struct {
			Recurrence map[string]interface{} `json:"recurrence"`
		} `json:"schedule"`
	}
	if err := json.Unmarshal(data, &item); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"weekdays", "local_times", "month_days"} {
		values, ok := item.Schedule.Recurrence[key].([]interface{})
		if !ok || len(values) != 0 {
			t.Fatalf("%s must be [], got %s", key, data)
		}
	}
}
