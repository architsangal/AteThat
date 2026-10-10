package server

import (
	"context"
	"errors"
	"strings"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"github.com/architsangal/AteThat/services/core/internal/security"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const registerMessage = "If registration can be completed, a verification email has been sent."
const resetMessage = "If an eligible account exists, a secure sign-in or reset link has been sent."

func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.GenericMessage, error) {
	displayEmail := strings.TrimSpace(req.Email)
	email, ok := validEmail(displayEmail)
	if !ok || !validPassword(req.Password) || !validTimezone(req.Timezone) || len(strings.TrimSpace(req.DisplayName)) < 1 || len(req.DisplayName) > 80 {
		return nil, invalid("invalid registration fields")
	}
	hash, err := security.PasswordHash(req.Password)
	if err != nil {
		return nil, internal()
	}
	raw, err := security.RandomToken()
	if err != nil {
		return nil, internal()
	}
	now := time.Now().UTC()
	user := userDoc{ID: primitive.NewObjectID(), EmailNormalized: email, EmailDisplay: displayEmail,
		Status: "PENDING_EMAIL_VERIFICATION", SchemaVersion: 1, Version: 1, CreatedAt: now, UpdatedAt: now}
	identity := identityDoc{ID: primitive.NewObjectID(), UserID: user.ID, IdentityType: "LOCAL_PASSWORD",
		PasswordHash: hash, PasswordChangedAt: now, SchemaVersion: 1, CreatedAt: now, UpdatedAt: now}
	profile := profileDoc{ID: primitive.NewObjectID(), UserID: user.ID, DisplayName: strings.TrimSpace(req.DisplayName),
		Timezone: req.Timezone, SchemaVersion: 1, Version: 1, CreatedAt: now, UpdatedAt: now}
	preferences := preferencesDoc{ID: primitive.NewObjectID(), UserID: user.ID, DailyResetTime: "04:00", UnitSystem: "METRIC",
		DashboardNutrients: []string{"ENERGY", "PROTEIN", "CARBOHYDRATE", "FAT", "FIBER"},
		SchemaVersion:      1, Version: 1, CreatedAt: now, UpdatedAt: now}
	token := emailTokenDoc{ID: primitive.NewObjectID(), UserID: user.ID, Purpose: "EMAIL_VERIFICATION",
		TokenHash: security.TokenHash(raw), ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now}
	session, err := s.Client.StartSession()
	if err != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		for collection, doc := range map[string]interface{}{"users": user, "auth_identities": identity,
			"user_profiles": profile, "user_preferences": preferences, "email_tokens": token} {
			if _, err := s.DB.Collection(collection).InsertOne(sc, doc); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
	if mongo.IsDuplicateKeyError(err) {
		return &pb.GenericMessage{Message: registerMessage}, nil
	}
	if err != nil {
		return nil, internal()
	}
	// Failure leaves the account unverified; a later resend can recover it.
	s.sendEmailBestEffort(ctx, user.EmailDisplay, "Verify your AteThat email", "/verify-email", raw)
	return &pb.GenericMessage{Message: registerMessage}, nil
}

func (s *Server) VerifyEmail(ctx context.Context, req *pb.VerifyEmailRequest) (*pb.VerifyEmailResponse, error) {
	if req.Token == "" {
		return nil, invalid("verification token is required")
	}
	now := time.Now().UTC()
	session, err := s.Client.StartSession()
	if err != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		var token emailTokenDoc
		filter := bson.M{"token_hash": security.TokenHash(req.Token), "purpose": "EMAIL_VERIFICATION",
			"consumed_at": nil, "expires_at": bson.M{"$gt": now}}
		if err := s.DB.Collection("email_tokens").FindOne(sc, filter).Decode(&token); err != nil {
			return nil, err
		}
		result, err := s.DB.Collection("email_tokens").UpdateOne(sc, bson.M{"_id": token.ID, "consumed_at": nil}, bson.M{"$set": bson.M{"consumed_at": now}})
		if err != nil || result.ModifiedCount != 1 {
			return nil, errors.New("token already consumed")
		}
		result, err = s.DB.Collection("users").UpdateOne(sc, bson.M{"_id": token.UserID, "status": "PENDING_EMAIL_VERIFICATION"},
			bson.M{"$set": bson.M{"status": "ACTIVE", "email_verified_at": now, "updated_at": now}, "$inc": bson.M{"version": 1}})
		if err == nil && result.ModifiedCount != 1 {
			return nil, errors.New("account not pending verification")
		}
		return nil, err
	})
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "verification token expired or already used")
	}
	return &pb.VerifyEmailResponse{EmailVerified: true}, nil
}

func (s *Server) ResendVerification(ctx context.Context, req *pb.EmailRequest) (*pb.GenericMessage, error) {
	email, ok := validEmail(strings.TrimSpace(req.Email))
	if !ok {
		return nil, invalid("invalid email")
	}
	var user userDoc
	if err := s.DB.Collection("users").FindOne(ctx, bson.M{"email_normalized": email, "status": "PENDING_EMAIL_VERIFICATION"}).Decode(&user); err != nil {
		return &pb.GenericMessage{Message: registerMessage}, nil
	}
	raw, err := security.RandomToken()
	if err != nil {
		return nil, internal()
	}
	now := time.Now().UTC()
	// Supersede earlier challenges before issuing the replacement.
	_, err = s.DB.Collection("email_tokens").UpdateMany(ctx, bson.M{"user_id": user.ID, "purpose": "EMAIL_VERIFICATION", "consumed_at": nil},
		bson.M{"$set": bson.M{"consumed_at": now}})
	if err != nil {
		return nil, internal()
	}
	_, err = s.DB.Collection("email_tokens").InsertOne(ctx, emailTokenDoc{ID: primitive.NewObjectID(), UserID: user.ID,
		Purpose: "EMAIL_VERIFICATION", TokenHash: security.TokenHash(raw), ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now})
	if err != nil {
		return nil, internal()
	}
	s.sendEmailBestEffort(ctx, user.EmailDisplay, "Verify your AteThat email", "/verify-email", raw)
	return &pb.GenericMessage{Message: registerMessage}, nil
}

func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.SessionResult, error) {
	email, ok := validEmail(strings.TrimSpace(req.Email))
	if !ok || req.Password == "" {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	var user userDoc
	var identity identityDoc
	err := s.DB.Collection("users").FindOne(ctx, bson.M{"email_normalized": email}).Decode(&user)
	if err == nil {
		err = s.DB.Collection("auth_identities").FindOne(ctx, bson.M{"user_id": user.ID, "identity_type": "LOCAL_PASSWORD"}).Decode(&identity)
	}
	if err != nil || !security.VerifyPassword(identity.PasswordHash, req.Password) {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	if user.Status != "ACTIVE" {
		return nil, status.Error(codes.PermissionDenied, "email not verified or account unavailable")
	}
	return s.newSession(ctx, user)
}

func (s *Server) newSession(ctx context.Context, user userDoc) (*pb.SessionResult, error) {
	raw, err := security.RandomToken()
	if err != nil {
		return nil, internal()
	}
	family, err := security.RandomToken()
	if err != nil {
		return nil, internal()
	}
	now := time.Now().UTC()
	doc := sessionDoc{ID: primitive.NewObjectID(), UserID: user.ID, FamilyID: family, RefreshTokenHash: security.TokenHash(raw),
		Generation: 1, Status: "ACTIVE", ExpiresAt: now.Add(30 * 24 * time.Hour), SchemaVersion: 1, CreatedAt: now, UpdatedAt: now}
	if _, err := s.DB.Collection("sessions").InsertOne(ctx, doc); err != nil {
		return nil, internal()
	}
	return s.sessionResult(user, raw, family, now)
}

func (s *Server) sessionResult(user userDoc, refresh, family string, now time.Time) (*pb.SessionResult, error) {
	access, err := security.AccessToken(s.Cfg.SigningKey, user.ID.Hex(), family, now)
	if err != nil {
		return nil, internal()
	}
	return &pb.SessionResult{AccessToken: access, ExpiresInSeconds: 900, RefreshToken: refresh,
		User: &pb.User{Id: user.ID.Hex(), Email: user.EmailDisplay, EmailVerified: user.EmailVerifiedAt != nil, Status: user.Status}}, nil
}

func (s *Server) RefreshSession(ctx context.Context, req *pb.RefreshSessionRequest) (*pb.SessionResult, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.Unauthenticated, "session expired")
	}
	var old sessionDoc
	if err := s.DB.Collection("sessions").FindOne(ctx, bson.M{"refresh_token_hash": security.TokenHash(req.RefreshToken)}).Decode(&old); err != nil {
		return nil, status.Error(codes.Unauthenticated, "session expired")
	}
	if old.Status == "ROTATED" {
		now := time.Now().UTC()
		_, _ = s.DB.Collection("sessions").UpdateMany(ctx, bson.M{"user_id": old.UserID, "family_id": old.FamilyID, "status": "ACTIVE"},
			bson.M{"$set": bson.M{"status": "REVOKED", "revoked_at": now, "revoke_reason": "ROTATED_TOKEN_REUSE", "updated_at": now}})
	}
	if old.Status != "ACTIVE" || !old.ExpiresAt.After(time.Now()) {
		return nil, status.Error(codes.Unauthenticated, "session expired")
	}
	var user userDoc
	if err := s.DB.Collection("users").FindOne(ctx, bson.M{"_id": old.UserID, "status": "ACTIVE"}).Decode(&user); err != nil {
		return nil, status.Error(codes.Unauthenticated, "session expired")
	}
	raw, err := security.RandomToken()
	if err != nil {
		return nil, internal()
	}
	now := time.Now().UTC()
	newDoc := sessionDoc{ID: primitive.NewObjectID(), UserID: old.UserID, FamilyID: old.FamilyID,
		RefreshTokenHash: security.TokenHash(raw), Generation: old.Generation + 1, Status: "ACTIVE",
		ExpiresAt: old.ExpiresAt, SchemaVersion: 1, CreatedAt: now, UpdatedAt: now}
	session, err := s.Client.StartSession()
	if err != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		result, err := s.DB.Collection("sessions").UpdateOne(sc, bson.M{"_id": old.ID, "status": "ACTIVE"},
			bson.M{"$set": bson.M{"status": "ROTATED", "last_used_at": now, "updated_at": now}})
		if err != nil || result.ModifiedCount != 1 {
			return nil, errors.New("session already rotated")
		}
		_, err = s.DB.Collection("sessions").InsertOne(sc, newDoc)
		return nil, err
	})
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "session expired")
	}
	return s.sessionResult(user, raw, old.FamilyID, now)
}

func (s *Server) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.Empty, error) {
	if req.RefreshToken == "" {
		return &pb.Empty{}, nil
	}
	var session sessionDoc
	if err := s.DB.Collection("sessions").FindOne(ctx, bson.M{"refresh_token_hash": security.TokenHash(req.RefreshToken)}).Decode(&session); err != nil {
		return &pb.Empty{}, nil
	}
	now := time.Now().UTC()
	_, err := s.DB.Collection("sessions").UpdateMany(ctx, bson.M{"user_id": session.UserID, "family_id": session.FamilyID, "status": "ACTIVE"},
		bson.M{"$set": bson.M{"status": "REVOKED", "revoked_at": now, "revoke_reason": "LOGOUT", "updated_at": now}})
	if err != nil {
		return nil, internal()
	}
	return &pb.Empty{}, nil
}

func (s *Server) RequestPasswordReset(ctx context.Context, req *pb.EmailRequest) (*pb.GenericMessage, error) {
	email, ok := validEmail(strings.TrimSpace(req.Email))
	if !ok {
		return nil, invalid("invalid email")
	}
	var user userDoc
	if err := s.DB.Collection("users").FindOne(ctx, bson.M{"email_normalized": email, "status": "ACTIVE"}).Decode(&user); err != nil {
		return &pb.GenericMessage{Message: resetMessage}, nil
	}
	// A Google-only account has no password to reset. Send a one-time email
	// sign-in link instead, while preserving the same public response so this
	// endpoint cannot be used to discover an account's sign-in methods.
	var localIdentity identityDoc
	err := s.DB.Collection("auth_identities").FindOne(ctx, bson.M{"user_id": user.ID, "identity_type": "LOCAL_PASSWORD"}).Decode(&localIdentity)
	if err == mongo.ErrNoDocuments {
		if err := s.requestEmailSignIn(ctx, user); err != nil {
			return nil, err
		}
		return &pb.GenericMessage{Message: resetMessage}, nil
	}
	if err != nil {
		return &pb.GenericMessage{Message: resetMessage}, nil
	}
	raw, err := security.RandomToken()
	if err != nil {
		return nil, internal()
	}
	now := time.Now().UTC()
	_, err = s.DB.Collection("email_tokens").InsertOne(ctx, emailTokenDoc{ID: primitive.NewObjectID(), UserID: user.ID,
		Purpose: "PASSWORD_RESET", TokenHash: security.TokenHash(raw), ExpiresAt: now.Add(15 * time.Minute), CreatedAt: now})
	if err != nil {
		return nil, internal()
	}
	s.sendEmailBestEffort(ctx, user.EmailDisplay, "Reset your AteThat password", "/reset-password", raw)
	return &pb.GenericMessage{Message: resetMessage}, nil
}

func (s *Server) requestEmailSignIn(ctx context.Context, user userDoc) error {
	raw, err := security.RandomToken()
	if err != nil {
		return internal()
	}
	now := time.Now().UTC()
	_, err = s.DB.Collection("email_tokens").UpdateMany(ctx, bson.M{"user_id": user.ID, "purpose": "EMAIL_SIGN_IN", "consumed_at": nil},
		bson.M{"$set": bson.M{"consumed_at": now}})
	if err != nil {
		return internal()
	}
	_, err = s.DB.Collection("email_tokens").InsertOne(ctx, emailTokenDoc{ID: primitive.NewObjectID(), UserID: user.ID,
		Purpose: "EMAIL_SIGN_IN", TokenHash: security.TokenHash(raw), ExpiresAt: now.Add(15 * time.Minute), CreatedAt: now})
	if err != nil {
		return internal()
	}
	s.sendEmailBestEffort(ctx, user.EmailDisplay, "Set an AteThat password", "/email-login", raw)
	return nil
}

func (s *Server) CompleteEmailSignIn(ctx context.Context, req *pb.EmailTokenRequest) (*pb.SessionResult, error) {
	if req.Token == "" || !validPassword(req.NewPassword) {
		return nil, invalid("email sign-in token and a 12-character password are required")
	}
	passwordHash, err := security.PasswordHash(req.NewPassword)
	if err != nil {
		return nil, internal()
	}
	now := time.Now().UTC()
	session, err := s.Client.StartSession()
	if err != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	var result *pb.SessionResult
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		var token emailTokenDoc
		filter := bson.M{"token_hash": security.TokenHash(req.Token), "purpose": "EMAIL_SIGN_IN", "consumed_at": nil, "expires_at": bson.M{"$gt": now}}
		if err := s.DB.Collection("email_tokens").FindOne(sc, filter).Decode(&token); err != nil {
			return nil, err
		}
		updated, err := s.DB.Collection("email_tokens").UpdateOne(sc, bson.M{"_id": token.ID, "consumed_at": nil}, bson.M{"$set": bson.M{"consumed_at": now}})
		if err != nil || updated.ModifiedCount != 1 {
			return nil, errors.New("token already consumed")
		}
		var user userDoc
		if err := s.DB.Collection("users").FindOne(sc, bson.M{"_id": token.UserID, "status": "ACTIVE"}).Decode(&user); err != nil {
			return nil, err
		}
		var localIdentity identityDoc
		err = s.DB.Collection("auth_identities").FindOne(sc, bson.M{"user_id": user.ID, "identity_type": "LOCAL_PASSWORD"}).Decode(&localIdentity)
		if err == nil {
			return nil, errors.New("local password already exists")
		}
		if err != mongo.ErrNoDocuments {
			return nil, err
		}
		identity := identityDoc{ID: primitive.NewObjectID(), UserID: user.ID, IdentityType: "LOCAL_PASSWORD",
			PasswordHash: passwordHash, PasswordChangedAt: now, SchemaVersion: 1, CreatedAt: now, UpdatedAt: now}
		if _, err := s.DB.Collection("auth_identities").InsertOne(sc, identity); err != nil {
			return nil, err
		}
		raw, err := security.RandomToken()
		if err != nil {
			return nil, err
		}
		family, err := security.RandomToken()
		if err != nil {
			return nil, err
		}
		doc := sessionDoc{ID: primitive.NewObjectID(), UserID: user.ID, FamilyID: family, RefreshTokenHash: security.TokenHash(raw),
			Generation: 1, Status: "ACTIVE", ExpiresAt: now.Add(30 * 24 * time.Hour), SchemaVersion: 1, CreatedAt: now, UpdatedAt: now}
		if _, err := s.DB.Collection("sessions").InsertOne(sc, doc); err != nil {
			return nil, err
		}
		result, err = s.sessionResult(user, raw, family, now)
		return nil, err
	})
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "email sign-in link expired or already used")
	}
	return result, nil
}

func (s *Server) ResetPassword(ctx context.Context, req *pb.ResetPasswordRequest) (*pb.Empty, error) {
	if req.Token == "" || !validPassword(req.NewPassword) {
		return nil, invalid("invalid reset fields")
	}
	hash, err := security.PasswordHash(req.NewPassword)
	if err != nil {
		return nil, internal()
	}
	now := time.Now().UTC()
	session, err := s.Client.StartSession()
	if err != nil {
		return nil, internal()
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(sc mongo.SessionContext) (interface{}, error) {
		var token emailTokenDoc
		filter := bson.M{"token_hash": security.TokenHash(req.Token), "purpose": "PASSWORD_RESET", "consumed_at": nil, "expires_at": bson.M{"$gt": now}}
		if err := s.DB.Collection("email_tokens").FindOne(sc, filter).Decode(&token); err != nil {
			return nil, err
		}
		result, err := s.DB.Collection("email_tokens").UpdateOne(sc, bson.M{"_id": token.ID, "consumed_at": nil}, bson.M{"$set": bson.M{"consumed_at": now}})
		if err != nil || result.ModifiedCount != 1 {
			return nil, errors.New("token already consumed")
		}
		result, err = s.DB.Collection("auth_identities").UpdateOne(sc, bson.M{"user_id": token.UserID, "identity_type": "LOCAL_PASSWORD"},
			bson.M{"$set": bson.M{"password_hash": hash, "password_changed_at": now, "updated_at": now}})
		if err != nil || result.MatchedCount != 1 {
			return nil, errors.New("local identity missing")
		}
		_, err = s.DB.Collection("sessions").UpdateMany(sc, bson.M{"user_id": token.UserID, "status": "ACTIVE"},
			bson.M{"$set": bson.M{"status": "REVOKED", "revoked_at": now, "revoke_reason": "PASSWORD_RESET", "updated_at": now}})
		return nil, err
	})
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "reset token expired or already used")
	}
	return &pb.Empty{}, nil
}
