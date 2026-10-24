package server

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestFoodReviewErrorsAreActionableAndSafe(t *testing.T) {
	for _, tc := range []struct {
		message  string
		code     codes.Code
		httpCode int
		want     string
	}{
		{"invalid food quantity", codes.InvalidArgument, 422, "greater than zero"},
		{"logged_at must be within the last 90 days", codes.InvalidArgument, 422, "Use current time"},
		{"invalid nutrient amount", codes.InvalidArgument, 422, "Edit nutrition"},
		{"SOURCE_NUTRITION_INVALID", codes.FailedPrecondition, 502, "Try another food entry or use Quick log"},
		{"secret-token user-health-record https://provider.invalid/?api_key=secret", codes.InvalidArgument, 422, "Nothing has been logged"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			var output bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&output)
			defer log.SetOutput(previous)
			w := httptest.NewRecorder()
			(&Server{}).foodReviewProblem(w, "review-request", status.Error(tc.code, tc.message))
			if w.Code != tc.httpCode || !strings.Contains(w.Body.String(), tc.want) {
				t.Fatal(w.Code, w.Body.String())
			}
			var body map[string]any
			if json.Unmarshal(w.Body.Bytes(), &body) != nil || body["request_id"] != "review-request" {
				t.Fatal(w.Body.String())
			}
			for _, text := range []string{w.Body.String(), output.String()} {
				if strings.Contains(text, "secret-token") || strings.Contains(text, "api_key") || strings.Contains(text, "user-health-record") {
					t.Fatal("sensitive RPC detail leaked")
				}
			}
			if !strings.Contains(output.String(), `"event":"food_review_failed"`) || !strings.Contains(output.String(), `"request_id":"review-request"`) {
				t.Fatal(output.String())
			}
		})
	}
}
