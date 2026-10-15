package server

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"github.com/architsangal/AteThat/services/core/internal/security"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const accountDeletionGracePeriod = 30 * 24 * time.Hour

// RequestAccountDeletion requires the current local password. Google-only accounts
// use a fresh Google authorization and reach cancelGoogleDeletion instead.
func (s *Server) RequestAccountDeletion(ctx context.Context, req *pb.RequestAccountDeletionRequest) (*pb.AccountDeletionStatus, error) {
	userID, err := Principal(ctx)
	if err != nil {
		return nil, err
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "current password is required; Google-only accounts must use Google re-authentication")
	}
	var identity identityDoc
	if err := s.DB.Collection("auth_identities").FindOne(ctx, bson.M{"user_id": userID, "identity_type": "LOCAL_PASSWORD"}).Decode(&identity); err != nil || !security.VerifyPassword(identity.PasswordHash, req.Password) {
		return nil, status.Error(codes.Unauthenticated, "current password was not accepted")
	}
	return s.requestDeletionForUser(ctx, userID)
}

func (s *Server) requestDeletionForUser(ctx context.Context, userID primitive.ObjectID) (*pb.AccountDeletionStatus, error) {
	now := time.Now().UTC()
	purgeAfter := now.Add(accountDeletionGracePeriod)
	session, err := s.Client.StartSession()
	if err != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		var user userDoc
		if err := s.DB.Collection("users").FindOne(sc, bson.M{"_id": userID, "status": "ACTIVE"}).Decode(&user); err != nil {
			return nil, err
		}
		if _, err := s.DB.Collection("account_deletions").UpdateOne(sc, bson.M{"user_id": userID}, bson.M{"$set": accountDeletionDoc{
			UserID: userID, Status: "PENDING", RequestedAt: now, PurgeAfter: purgeAfter, UpdatedAt: now,
		}}, options.Update().SetUpsert(true)); err != nil {
			return nil, err
		}
		if _, err := s.DB.Collection("users").UpdateOne(sc, bson.M{"_id": userID, "status": "ACTIVE"}, bson.M{"$set": bson.M{"status": "DELETION_PENDING", "updated_at": now}}); err != nil {
			return nil, err
		}
		_, err := s.DB.Collection("sessions").UpdateMany(sc, bson.M{"user_id": userID, "status": "ACTIVE"}, bson.M{"$set": bson.M{"status": "REVOKED", "revoked_at": now, "revoke_reason": "ACCOUNT_DELETION_REQUESTED", "updated_at": now}})
		return nil, err
	})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Error(codes.FailedPrecondition, "account is not available for deletion")
		}
		return nil, internal()
	}
	return &pb.AccountDeletionStatus{Status: "PENDING", RequestedAt: now.Format(time.RFC3339Nano), PurgeAfter: purgeAfter.Format(time.RFC3339Nano)}, nil
}

func (s *Server) CancelAccountDeletion(ctx context.Context, req *pb.CancelAccountDeletionRequest) (*pb.SessionResult, error) {
	email, ok := validEmail(strings.TrimSpace(req.Email))
	if !ok || req.Password == "" {
		return nil, status.Error(codes.Unauthenticated, "invalid cancellation credentials")
	}
	var user userDoc
	if err := s.DB.Collection("users").FindOne(ctx, bson.M{"email_normalized": email, "status": "DELETION_PENDING"}).Decode(&user); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid cancellation credentials")
	}
	var identity identityDoc
	if err := s.DB.Collection("auth_identities").FindOne(ctx, bson.M{"user_id": user.ID, "identity_type": "LOCAL_PASSWORD"}).Decode(&identity); err != nil || !security.VerifyPassword(identity.PasswordHash, req.Password) {
		return nil, status.Error(codes.Unauthenticated, "invalid cancellation credentials")
	}
	if err := s.cancelDeletionForUser(ctx, user.ID); err != nil {
		return nil, err
	}
	user.Status = "ACTIVE"
	return s.newSession(ctx, user)
}

func (s *Server) cancelDeletionForUser(ctx context.Context, userID primitive.ObjectID) error {
	now := time.Now().UTC()
	session, err := s.Client.StartSession()
	if err != nil {
		return internal()
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		result, err := s.DB.Collection("users").UpdateOne(sc, bson.M{"_id": userID, "status": "DELETION_PENDING"}, bson.M{"$set": bson.M{"status": "ACTIVE", "updated_at": now}})
		if err != nil || result.ModifiedCount != 1 {
			return nil, errors.New("account is not pending deletion")
		}
		result, err = s.DB.Collection("account_deletions").UpdateOne(sc, bson.M{"user_id": userID, "status": "PENDING"}, bson.M{"$set": bson.M{"status": "CANCELLED", "cancelled_at": now, "updated_at": now}})
		if err != nil || result.ModifiedCount != 1 {
			return nil, errors.New("deletion request is not pending")
		}
		return nil, nil
	})
	if err != nil {
		return status.Error(codes.FailedPrecondition, "account deletion is no longer pending")
	}
	return nil
}

// RunAccountDeletionPurge permanently removes user-owned data whose explicit
// 30-day cancellation period has elapsed. Reference and shared provider caches
// are deliberately excluded.
func (s *Server) RunAccountDeletionPurge(ctx context.Context) error {
	now := time.Now().UTC()
	cursor, err := s.DB.Collection("account_deletions").Find(ctx, bson.M{"status": "PENDING", "purge_after": bson.M{"$lte": now}})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var deletion accountDeletionDoc
		if err := cursor.Decode(&deletion); err != nil {
			return err
		}
		if err := s.purgeUser(ctx, deletion.UserID); err != nil {
			return err
		}
	}
	return cursor.Err()
}

func (s *Server) RunAccountDeletionPurgeLoop(ctx context.Context) {
	for {
		if err := s.RunAccountDeletionPurge(ctx); err != nil {
			log.Printf("account deletion purge failed; retrying in one minute: %v", err)
			if !waitForPurge(ctx, time.Minute) {
				return
			}
			continue
		}
		if !waitForPurge(ctx, time.Hour) {
			return
		}
	}
}

func waitForPurge(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (s *Server) purgeUser(ctx context.Context, userID primitive.ObjectID) error {
	session, err := s.Client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		collections := []string{"wellness_records", "wellness_settings", "auth_identities", "oauth_states", "sessions", "email_tokens", "user_profiles", "user_preferences", "nutrition_target_versions", "custom_foods", "meals", "serving_definitions", "food_favourites", "food_log_previews", "food_log_groups", "food_logs", "daily_nutrition_summaries", "health_items", "health_item_schedule_versions", "health_item_events"}
		for _, collection := range collections {
			if _, err := s.DB.Collection(collection).DeleteMany(sc, bson.M{"user_id": userID}); err != nil {
				return nil, err
			}
		}
		if _, err := s.DB.Collection("account_deletions").DeleteOne(sc, bson.M{"user_id": userID, "status": "PENDING"}); err != nil {
			return nil, err
		}
		_, err := s.DB.Collection("users").DeleteOne(sc, bson.M{"_id": userID, "status": "DELETION_PENDING"})
		return nil, err
	})
	return err
}
