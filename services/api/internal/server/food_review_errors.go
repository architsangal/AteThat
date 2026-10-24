package server

import (
	"encoding/json"
	"log"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Only known, static reasons cross the public boundary. Never forward arbitrary
// RPC text: it may contain provider URLs, credentials or user-entered content.
func foodReviewValidation(message string) (string, string) {
	switch message {
	case "invalid food quantity", "invalid recipe quantity":
		return "FOOD_AMOUNT", "Enter an amount greater than zero for every selected food, then review again."
	case "serving unit must match the food base unit":
		return "FOOD_UNIT", "The amount unit does not match this food. Remove and add the food again, or use Quick log with the correct unit."
	case "logged_at must include a timezone offset", "logged_at must be within the last 90 days":
		return "LOG_TIME", "Choose a log date and time within the last 90 days, not in the future. Open Change log date and time, or choose Use current time."
	case "invalid meal category":
		return "MEAL_CATEGORY", "Choose Breakfast, Lunch, Dinner, Snack or Other, then review again."
	case "invalid nutrient", "invalid nutrient amount", "invalid unknown nutrient", "override unit must match the provider nutrient":
		return "NUTRITION_EDIT", "Check the nutrition values under More options → Edit nutrition. Use zero or a positive number, or clear an edit to use the food’s original value."
	case "invalid food state":
		return "FOOD_STATE", "This food’s preparation value could not be read. Remove and add it again, or use Quick log."
	case "invalid USDA food reference", "invalid custom food reference", "invalid food reference type", "invalid USDA food ID", "invalid manual food", "manual food requires a manual reference":
		return "FOOD_REFERENCE", "One selected food could not be read. Remove and add it again, or use Quick log."
	case "food group requires 1–50 items and a short label":
		return "MEAL_SIZE", "Select 1–50 foods and keep the meal name within 160 characters."
	default:
		return "UNCLASSIFIED", "We couldn’t review this meal. Nothing has been logged. Try again, or use Quick log if this food keeps failing."
	}
}

func (s *Server) foodReviewProblem(w http.ResponseWriter, requestID string, err error) {
	reason, detail := foodReviewValidation(status.Convert(err).Message())
	sourceFailure := status.Code(err) == codes.FailedPrecondition && status.Convert(err).Message() == "SOURCE_NUTRITION_INVALID"
	if sourceFailure {
		reason = "SOURCE_NUTRITION"
	}
	entry, _ := json.Marshal(map[string]string{"event": "food_review_failed", "request_id": requestID, "operation": "preview", "rpc_code": status.Code(err).String(), "reason": reason})
	log.Print(string(entry))
	if sourceFailure {
		s.problem(w, requestID, http.StatusBadGateway, "FOOD_SOURCE_INVALID", "Nutrition data for this food could not be processed. Try another food entry or use Quick log. Nothing has been logged.")
		return
	}
	if status.Code(err) == codes.InvalidArgument {
		s.problem(w, requestID, http.StatusUnprocessableEntity, "VALIDATION_FAILED", detail)
		return
	}
	s.rpcProblem(w, requestID, err)
}
