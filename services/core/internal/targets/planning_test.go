package targets

import "testing"

func TestMacroCaloriesExact(t *testing.T) {
	for _, tc := range []struct{ p, c, f, want string }{{"160", "400", "100", "3140"}, {"0", "0", "0", "0"}, {"0.0000000000000001", "0", "0", "0.0000000000000004"}, {"50.123456789", "275", "78", "2002.493827156"}} {
		got, err := MacroCalories(map[string]string{"PROTEIN": tc.p, "CARBOHYDRATE": tc.c, "FAT": tc.f})
		if err != nil || got != tc.want {
			t.Fatalf("got %q %v; want %s", got, err, tc.want)
		}
	}
	for _, value := range []string{"", "-1", "NaN", "1e3", "01"} {
		if _, err := MacroCalories(map[string]string{"PROTEIN": value, "CARBOHYDRATE": "1", "FAT": "1"}); err == nil {
			t.Fatalf("accepted %q", value)
		}
	}
	if ValidSelectedAmount("ENERGY", "0") || !ValidSelectedAmount("FIBER", "0") {
		t.Fatal("incorrect zero policy")
	}
}
