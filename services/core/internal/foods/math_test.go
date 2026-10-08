package foods

import "testing"

func TestScaleAndSum(t *testing.T) {
	first, err := Scale("130", "200", "100")
	if err != nil || first != "260" {
		t.Fatalf("scale: %s %v", first, err)
	}
	second, err := Scale("2.69", "150", "100")
	if err != nil || second != "4.035" {
		t.Fatalf("scale: %s %v", second, err)
	}
	total, err := Sum([]string{first, second})
	if err != nil || total != "264.035" {
		t.Fatalf("sum: %s %v", total, err)
	}
}

func TestRejectNonDecimalNutrition(t *testing.T) {
	for _, raw := range []string{"1/2", "1e3", "+2", "-1", ".5", "1.", "NaN", "1..2"} {
		if _, err := Scale(raw, "1", "1"); err == nil {
			t.Errorf("scale accepted %q", raw)
		}
		if _, err := Sum([]string{raw}); err == nil {
			t.Errorf("sum accepted %q", raw)
		}
	}
}
