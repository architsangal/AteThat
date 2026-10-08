package server

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type userDoc struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	EmailNormalized string             `bson:"email_normalized"`
	EmailDisplay    string             `bson:"email_display"`
	EmailVerifiedAt *time.Time         `bson:"email_verified_at"`
	Status          string             `bson:"status"`
	SchemaVersion   int                `bson:"schema_version"`
	Version         int64              `bson:"version"`
	CreatedAt       time.Time          `bson:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at"`
}

type identityDoc struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	UserID            primitive.ObjectID `bson:"user_id"`
	IdentityType      string             `bson:"identity_type"`
	PasswordHash      string             `bson:"password_hash"`
	Provider          string             `bson:"provider,omitempty"`
	Subject           string             `bson:"subject,omitempty"`
	PasswordChangedAt time.Time          `bson:"password_changed_at"`
	SchemaVersion     int                `bson:"schema_version"`
	CreatedAt         time.Time          `bson:"created_at"`
	UpdatedAt         time.Time          `bson:"updated_at"`
}

type oauthStateDoc struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty"`
	StateHash    string              `bson:"state_hash"`
	Nonce        string              `bson:"nonce"`
	CodeVerifier string              `bson:"code_verifier"`
	ReturnTo     string              `bson:"return_to"`
	Operation    string              `bson:"operation"`
	UserID       *primitive.ObjectID `bson:"user_id,omitempty"`
	ExpiresAt    time.Time           `bson:"expires_at"`
	ConsumedAt   *time.Time          `bson:"consumed_at"`
	CreatedAt    time.Time           `bson:"created_at"`
}

type sessionDoc struct {
	ID               primitive.ObjectID `bson:"_id,omitempty"`
	UserID           primitive.ObjectID `bson:"user_id"`
	FamilyID         string             `bson:"family_id"`
	RefreshTokenHash string             `bson:"refresh_token_hash"`
	Generation       int                `bson:"generation"`
	Status           string             `bson:"status"`
	ExpiresAt        time.Time          `bson:"expires_at"`
	LastUsedAt       *time.Time         `bson:"last_used_at"`
	RevokedAt        *time.Time         `bson:"revoked_at"`
	RevokeReason     string             `bson:"revoke_reason,omitempty"`
	SchemaVersion    int                `bson:"schema_version"`
	CreatedAt        time.Time          `bson:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at"`
}

type emailTokenDoc struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	UserID     primitive.ObjectID `bson:"user_id"`
	Purpose    string             `bson:"purpose"`
	TokenHash  string             `bson:"token_hash"`
	ExpiresAt  time.Time          `bson:"expires_at"`
	ConsumedAt *time.Time         `bson:"consumed_at"`
	CreatedAt  time.Time          `bson:"created_at"`
}

type quantityDoc struct {
	Amount primitive.Decimal128 `bson:"amount"`
	Unit   string               `bson:"unit"`
}

type profileDoc struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	UserID        primitive.ObjectID `bson:"user_id"`
	DisplayName   string             `bson:"display_name"`
	DateOfBirth   string             `bson:"date_of_birth,omitempty"`
	Sex           string             `bson:"sex,omitempty"`
	Height        *quantityDoc       `bson:"height,omitempty"`
	Weight        *quantityDoc       `bson:"weight,omitempty"`
	ActivityLevel string             `bson:"activity_level,omitempty"`
	Goal          string             `bson:"goal,omitempty"`
	Timezone      string             `bson:"timezone"`
	SchemaVersion int                `bson:"schema_version"`
	Version       int64              `bson:"version"`
	CreatedAt     time.Time          `bson:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at"`
}

type preferencesDoc struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty"`
	UserID             primitive.ObjectID `bson:"user_id"`
	DailyResetTime     string             `bson:"daily_reset_time"`
	UnitSystem         string             `bson:"unit_system"`
	DashboardNutrients []string           `bson:"dashboard_nutrients"`
	SchemaVersion      int                `bson:"schema_version"`
	Version            int64              `bson:"version"`
	CreatedAt          time.Time          `bson:"created_at"`
	UpdatedAt          time.Time          `bson:"updated_at"`
}

func createIndexes(ctx context.Context, db *mongo.Database) error {
	models := map[string][]mongo.IndexModel{
		"users": {{Keys: bson.D{{Key: "email_normalized", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"auth_identities": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "identity_type", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "provider", Value: 1}, {Key: "subject", Value: 1}}, Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.M{"provider": "GOOGLE"})},
		},
		"oauth_states": {
			{Keys: bson.D{{Key: "state_hash", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		},
		"sessions": {
			{Keys: bson.D{{Key: "refresh_token_hash", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "family_id", Value: 1}}},
			{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		},
		"email_tokens": {
			{Keys: bson.D{{Key: "token_hash", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "purpose", Value: 1}, {Key: "created_at", Value: -1}}},
			{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		},
		"user_profiles":    {{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"user_preferences": {{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)}},
	}
	for collection, indexes := range models {
		if _, err := db.Collection(collection).Indexes().CreateMany(ctx, indexes); err != nil {
			return err
		}
	}
	return nil
}
