package server

import (
	"testing"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAdultRangesBoundariesAndNoInventedPointLimits(t *testing.T) {
	values := map[string]targetValueDoc{}
	for k, v := range map[string]string{"ENERGY": "2000", "PROTEIN": "50", "CARBOHYDRATE": "325", "FAT": "60", "FIBER": "0"} {
		n, _ := decimalTargetValue(v)
		values[k] = targetValueDoc{ActiveValue: n, Unit: "g"}
	}
	if warnings := applyTargetRanges(values, nil, 26); len(warnings) != 0 {
		t.Fatal(warnings)
	}
	if values["ENERGY"].ReferenceRange != nil || values["FIBER"].ReferenceRange != nil {
		t.Fatal("point limit fabricated")
	}
	value := values["PROTEIN"]
	value.ActiveValue, _ = decimalTargetValue("175.000000001")
	values["PROTEIN"] = value
	warnings := applyTargetRanges(values, nil, 26)
	if len(warnings) != 1 || warnings[0].Nutrient != "PROTEIN" || warnings[0].ReferenceRange.Minimum != "50" || warnings[0].ReferenceRange.Maximum != "175" {
		t.Fatal(warnings)
	}
	if warnings := applyTargetRanges(values, nil, 18); len(warnings) != 0 || values["PROTEIN"].ReferenceRange != nil {
		t.Fatal("adult ranges applied to minor")
	}
}
func TestCustomRangeAcknowledgementsAreSignedExactAndDurable(t *testing.T) {
	s, owner, other := phaseTwoServer(t)
	s.Cfg.InternalSecret = []byte("fixture-only-internal-secret-32-bytes")
	user, _ := Principal(owner)
	if _, e := s.DB.Collection("user_profiles").UpdateOne(owner, bson.M{"user_id": user}, bson.M{"$set": bson.M{"version": int64(1), "date_of_birth": "2000-01-01"}}); e != nil {
		t.Fatal(e)
	}
	changes := map[string]*pb.TargetOverride{}
	for k, v := range map[string]string{"ENERGY": "2000", "PROTEIN": "75", "CARBOHYDRATE": "250", "FAT": "60", "FIBER": "28.123456789"} {
		changes[k] = &pb.TargetOverride{ActiveValue: v}
	}
	initial, e := s.SaveCustom(owner, &pb.SaveCustomTargetsRequest{ProfileVersion: 1, Changes: changes})
	if e != nil {
		t.Fatal(e)
	}
	request := &pb.SaveCustomTargetsRequest{ExpectedActiveId: initial.Id, ExpectedRevision: initial.Revision, ProfileVersion: 1, Changes: map[string]*pb.TargetOverride{"PROTEIN": {ActiveValue: "500"}}}
	review, e := s.ReviewCustom(owner, request)
	if e != nil || len(review.Warnings) != 1 || review.AcknowledgementToken == "" {
		t.Fatal(review, e)
	}
	if _, e := s.SaveCustom(owner, request); status.Code(e) != codes.FailedPrecondition {
		t.Fatal("unreviewed save", e)
	}
	request.AcknowledgementToken = "forged"
	if _, e := s.SaveCustom(owner, request); status.Code(e) != codes.FailedPrecondition {
		t.Fatal("forged confirmation", e)
	}
	request.AcknowledgementToken = review.AcknowledgementToken
	request.Changes["PROTEIN"].ActiveValue = "501"
	if _, e := s.SaveCustom(owner, request); status.Code(e) != codes.FailedPrecondition {
		t.Fatal("changed value confirmation", e)
	}
	request.Changes["PROTEIN"].ActiveValue = "500"
	if _, e := s.SaveCustom(other, request); e == nil {
		t.Fatal("foreign acknowledgement accepted")
	}
	accepted, e := s.SaveCustom(owner, request)
	if e != nil {
		t.Fatal(e)
	}
	p := accepted.Targets["PROTEIN"]
	if !p.OutsideReference || p.AcknowledgedAt == "" || p.ReferenceRange.EnergyBasis != "2000" || accepted.Targets["FIBER"].ActiveValue != "28.123456789" {
		t.Fatal(accepted)
	}
	next := &pb.SaveCustomTargetsRequest{ExpectedActiveId: accepted.Id, ExpectedRevision: accepted.Revision, ProfileVersion: 1, Changes: map[string]*pb.TargetOverride{"FIBER": {ActiveValue: "29"}}}
	kept, e := s.SaveCustom(owner, next)
	if e != nil || kept.Targets["PROTEIN"].AcknowledgedAt != p.AcknowledgedAt {
		t.Fatal("unchanged acknowledgement lost", kept, e)
	}
	next.ExpectedActiveId, next.ExpectedRevision = kept.Id, kept.Revision
	next.Changes = map[string]*pb.TargetOverride{"ENERGY": {ActiveValue: "2500"}}
	if _, e := s.SaveCustom(owner, next); status.Code(e) != codes.FailedPrecondition {
		t.Fatal("new reference basis bypassed acknowledgement", e)
	}
	next.AcknowledgementToken = review.AcknowledgementToken
	if _, e := s.SaveCustom(owner, next); status.Code(e) != codes.FailedPrecondition {
		t.Fatal("replayed token accepted for new basis", e)
	}
	newer, e := s.ReviewCustom(owner, next)
	if e != nil || len(newer.Warnings) == 0 {
		t.Fatal(newer, e)
	}
	next.AcknowledgementToken = s.signTargetReview("irrelevant", time.Now().Add(-time.Minute).Unix())
	if _, e := s.SaveCustom(owner, next); status.Code(e) != codes.FailedPrecondition {
		t.Fatal("expired token accepted", e)
	}
	next.AcknowledgementToken = newer.AcknowledgementToken
	if _, e := s.DB.Collection("user_profiles").UpdateOne(owner, bson.M{"user_id": user}, bson.M{"$set": bson.M{"version": int64(2)}}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.SaveCustom(owner, next); status.Code(e) != codes.Aborted {
		t.Fatal("stale profile accepted", e)
	}
}
func TestAutomaticProposalIsIdempotentExplicitAndProfileBound(t *testing.T) {
	s, owner, other := phaseTwoServer(t)
	user, _ := Principal(owner)
	if _, e := s.DB.Collection("user_profiles").UpdateOne(owner, bson.M{"user_id": user}, bson.M{"$set": bson.M{"version": int64(1), "date_of_birth": "2000-01-01", "sex": "MALE"}}); e != nil {
		t.Fatal(e)
	}
	first, e := s.ProposeCalculated(owner, &pb.Empty{})
	if e != nil || first.Status != "PROPOSED" {
		t.Fatal(first, e)
	}
	again, e := s.ProposeCalculated(owner, &pb.Empty{})
	if e != nil || again.Id != first.Id {
		t.Fatal("not idempotent", again, e)
	}
	selection, e := s.Current(owner, &pb.Empty{})
	if e != nil || selection.Active != nil {
		t.Fatal("proposal activated without consent", selection, e)
	}
	day, e := s.GetCurrentTrackingDay(owner, &pb.Empty{})
	if e != nil {
		t.Fatal(e)
	}
	request := &pb.AcceptTargetRequest{VersionId: first.Id, EffectiveFromTrackingDay: day.TrackingDay}
	if _, e := s.Accept(owner, request); status.Code(e) != codes.InvalidArgument {
		t.Fatal("missing scope acknowledgement", e)
	}
	request.AdultScopeConfirmed = true
	if _, e := s.Accept(other, request); e == nil {
		t.Fatal("foreign proposal accepted")
	}
	if _, e := s.DB.Collection("user_profiles").UpdateOne(owner, bson.M{"user_id": user}, bson.M{"$set": bson.M{"version": int64(2)}}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Accept(owner, request); status.Code(e) != codes.Aborted {
		t.Fatal("stale proposal accepted", e)
	}
	fresh, e := s.ProposeCalculated(owner, &pb.Empty{})
	if e != nil || fresh.Id == first.Id {
		t.Fatal(fresh, e)
	}
	if _, e := s.Reject(owner, &pb.TargetVersionRequest{VersionId: fresh.Id}); e != nil {
		t.Fatal(e)
	}
	declined, e := s.ProposeCalculated(owner, &pb.Empty{})
	if e != nil || declined.Status != "REJECTED" || declined.Id != fresh.Id {
		t.Fatal("declined proposal recreated", declined, e)
	}
	if _, e := s.DB.Collection("user_profiles").UpdateOne(owner, bson.M{"user_id": user}, bson.M{"$set": bson.M{"version": int64(3)}}); e != nil {
		t.Fatal(e)
	}
	fresh, e = s.ProposeCalculated(owner, &pb.Empty{})
	if e != nil {
		t.Fatal(e)
	}
	request.VersionId = fresh.Id
	accepted, e := s.Accept(owner, request)
	if e != nil || accepted.Status != "ACTIVE" || accepted.InputSnapshot.WeightKg != "70.4375" {
		t.Fatal(accepted, e)
	}
	if _, e := s.Accept(owner, request); e == nil {
		t.Fatal("proposal accepted twice")
	}
	if _, e := s.ProposeCalculated(owner, &pb.Empty{}); status.Code(e) != codes.FailedPrecondition {
		t.Fatal("initial proposal replaced accepted targets", e)
	}
}
