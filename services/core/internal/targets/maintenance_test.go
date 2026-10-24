package targets

import "testing"

func TestLowActivityMaintenance(t *testing.T) {
	for _, tc := range []struct{ sex, want string }{{"MALE", "3008.17"}, {"FEMALE", "2575.2"}} {
		value, equation, err := LowActivityMaintenance(30, tc.sex, "180", "100")
		if err != nil || value != tc.want || equation == "" {
			t.Fatal(value, equation, err)
		}
	}
	for _, tc := range []struct {
		age       uint32
		sex, h, w string
	}{{18, "MALE", "175", "70"}, {30, "DECLINED", "175", "70"}, {30, "MALE", "0", "70"}, {30, "MALE", "175", "-1"}} {
		if _, _, err := LowActivityMaintenance(tc.age, tc.sex, tc.h, tc.w); err == nil {
			t.Fatal(tc)
		}
	}
}
