package server

import (
	"context"
	"time"

	"github.com/architsangal/AteThat/services/core/internal/targets"
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

// accountDeletionDoc keeps the cancellation window separate from the user record so
// the removal worker has an explicit, auditable target and deadline.
type accountDeletionDoc struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	UserID      primitive.ObjectID `bson:"user_id"`
	Status      string             `bson:"status"`
	RequestedAt time.Time          `bson:"requested_at"`
	PurgeAfter  time.Time          `bson:"purge_after"`
	CancelledAt *time.Time         `bson:"cancelled_at,omitempty"`
	UpdatedAt   time.Time          `bson:"updated_at"`
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

type referenceValueDoc struct {
	Amount primitive.Decimal128 `bson:"amount"`
	Unit   string               `bson:"unit"`
}

type referenceSetDoc struct {
	ID               primitive.ObjectID           `bson:"_id,omitempty"`
	ReferenceKey     string                       `bson:"reference_key"`
	ReferenceVersion string                       `bson:"reference_version"`
	Status           string                       `bson:"status"`
	TargetBasis      string                       `bson:"target_basis"`
	Source           targets.SourceAttribution    `bson:"source_attribution"`
	RequiredKeys     []string                     `bson:"required_nutrients"`
	Values           map[string]referenceValueDoc `bson:"values"`
	CreatedAt        time.Time                    `bson:"created_at"`
}

type targetValueDoc struct {
	ReferenceValue   primitive.Decimal128 `bson:"reference_value"`
	ActiveValue      primitive.Decimal128 `bson:"active_value"`
	Unit             string               `bson:"unit"`
	IsManualOverride bool                 `bson:"is_manual_override"`
	OverrideReason   string               `bson:"override_reason,omitempty"`
}

type targetVersionDoc struct {
	ID                       primitive.ObjectID        `bson:"_id,omitempty"`
	UserID                   primitive.ObjectID        `bson:"user_id"`
	Status                   string                    `bson:"status"`
	Revision                 int64                     `bson:"revision"`
	ProfileVersion           int64                     `bson:"profile_version"`
	ReferenceSet             bson.M                    `bson:"reference_set"`
	Source                   targets.SourceAttribution `bson:"source_attribution"`
	CalculationVersion       string                    `bson:"calculation_version"`
	TargetBasis              string                    `bson:"target_basis"`
	EffectiveFromTrackingDay string                    `bson:"effective_from_tracking_day"`
	Targets                  map[string]targetValueDoc `bson:"targets"`
	SupersedesVersionID      *primitive.ObjectID       `bson:"supersedes_version_id,omitempty"`
	ReviewedAt               *time.Time                `bson:"reviewed_at,omitempty"`
	CreatedAt                time.Time                 `bson:"created_at"`
}

type foodNutrientDoc struct {
	Status string                `bson:"status"`
	Amount *primitive.Decimal128 `bson:"amount,omitempty"`
	Unit   string                `bson:"unit"`
}

type customFoodDoc struct {
	ID           primitive.ObjectID         `bson:"_id,omitempty"`
	UserID       primitive.ObjectID         `bson:"user_id"`
	Version      int64                      `bson:"version"`
	Name         string                     `bson:"name"`
	Description  string                     `bson:"description"`
	FoodState    string                     `bson:"food_state"`
	BaseQuantity quantityDoc                `bson:"base_quantity"`
	Nutrients    map[string]foodNutrientDoc `bson:"nutrients"`
	DeletedAt    *time.Time                 `bson:"deleted_at,omitempty"`
	CreatedAt    time.Time                  `bson:"created_at"`
	UpdatedAt    time.Time                  `bson:"updated_at"`
}

type mealComponentDoc struct {
	Position       int         `bson:"position"`
	FoodType       string      `bson:"food_type"`
	ProviderFoodID string      `bson:"provider_food_id,omitempty"`
	CustomFoodID   string      `bson:"custom_food_id,omitempty"`
	Quantity       quantityDoc `bson:"quantity"`
	FoodState      string      `bson:"food_state"`
}

type mealDoc struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	UserID     primitive.ObjectID `bson:"user_id"`
	Version    int64              `bson:"version"`
	Name       string             `bson:"name"`
	Components []mealComponentDoc `bson:"components"`
	DeletedAt  *time.Time         `bson:"deleted_at,omitempty"`
	CreatedAt  time.Time          `bson:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at"`
}

type servingDoc struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         primitive.ObjectID `bson:"user_id"`
	SchemaVersion  int                `bson:"schema_version"`
	Version        int64              `bson:"version"`
	Label          string             `bson:"label"`
	FoodType       string             `bson:"food_type"`
	ProviderFoodID string             `bson:"provider_food_id,omitempty"`
	CustomFoodID   string             `bson:"custom_food_id,omitempty"`
	Quantity       quantityDoc        `bson:"quantity"`
	FoodState      string             `bson:"food_state"`
	DeletedAt      *time.Time         `bson:"deleted_at,omitempty"`
	CreatedAt      time.Time          `bson:"created_at"`
	UpdatedAt      time.Time          `bson:"updated_at"`
}

type foodFavouriteDoc struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	UserID         primitive.ObjectID `bson:"user_id"`
	FoodKey        string             `bson:"food_key"`
	FoodType       string             `bson:"food_type"`
	ProviderFoodID string             `bson:"provider_food_id,omitempty"`
	CustomFoodID   string             `bson:"custom_food_id,omitempty"`
	Name           string             `bson:"name"`
	CreatedAt      time.Time          `bson:"created_at"`
}

type foodSourceDoc struct {
	Type           string `bson:"type"`
	ProviderFoodID string `bson:"provider_food_id,omitempty"`
	CustomFoodID   string `bson:"custom_food_id,omitempty"`
	Name           string `bson:"name"`
}

type foodLogItemDoc struct {
	ID                    primitive.ObjectID         `bson:"_id,omitempty"`
	UserID                primitive.ObjectID         `bson:"user_id"`
	GroupID               primitive.ObjectID         `bson:"food_log_group_id"`
	Position              int                        `bson:"item_position"`
	FoodSource            foodSourceDoc              `bson:"food_source"`
	Quantity              quantityDoc                `bson:"quantity"`
	FoodState             string                     `bson:"food_state"`
	ProviderNutrition     map[string]foodNutrientDoc `bson:"provider_nutrition_snapshot"`
	UserOverrideNutrition map[string]foodNutrientDoc `bson:"user_override_nutrition"`
	FinalNutrition        map[string]foodNutrientDoc `bson:"final_nutrition_snapshot"`
	LoggedAt              time.Time                  `bson:"logged_at_utc"`
	TrackingDay           string                     `bson:"tracking_day"`
	ConfirmedAt           time.Time                  `bson:"confirmed_at"`
}

type foodLogGroupDoc struct {
	ID                     primitive.ObjectID         `bson:"_id,omitempty"`
	UserID                 primitive.ObjectID         `bson:"user_id"`
	RootGroupID            primitive.ObjectID         `bson:"root_group_id"`
	SupersedesGroupID      *primitive.ObjectID        `bson:"supersedes_group_id,omitempty"`
	Revision               int64                      `bson:"revision"`
	Status                 string                     `bson:"status"`
	IdempotencyKey         string                     `bson:"idempotency_key"`
	PreviewID              primitive.ObjectID         `bson:"preview_id"`
	PreviewSecretHash      string                     `bson:"preview_secret_hash"`
	SourceType             string                     `bson:"source_type"`
	SourceMealID           *primitive.ObjectID        `bson:"source_meal_id,omitempty"`
	SourceGroupID          *primitive.ObjectID        `bson:"source_group_id,omitempty"`
	CorrectionReason       string                     `bson:"correction_reason,omitempty"`
	LoggedAt               time.Time                  `bson:"logged_at_utc"`
	TrackingDay            string                     `bson:"tracking_day"`
	TimezoneAtLogging      string                     `bson:"timezone_at_logging"`
	DailyResetTimeAtLog    string                     `bson:"daily_reset_time_at_logging"`
	TrackingDayRuleVersion int                        `bson:"tracking_day_rule_version"`
	MealCategory           string                     `bson:"meal_category"`
	Label                  string                     `bson:"label"`
	ItemCount              int                        `bson:"item_count"`
	Aggregate              map[string]foodNutrientDoc `bson:"aggregate_final_nutrition_snapshot"`
	CalculationVersion     string                     `bson:"calculation_version"`
	ConfirmedAt            time.Time                  `bson:"confirmed_at"`
	VoidedAt               *time.Time                 `bson:"voided_at,omitempty"`
}

type foodPreviewDoc struct {
	ID                     primitive.ObjectID         `bson:"_id,omitempty"`
	UserID                 primitive.ObjectID         `bson:"user_id"`
	SecretHash             string                     `bson:"preview_secret_hash"`
	Status                 string                     `bson:"status"`
	SourceType             string                     `bson:"source_type"`
	SourceMealID           *primitive.ObjectID        `bson:"source_meal_id,omitempty"`
	SourceGroupID          *primitive.ObjectID        `bson:"source_group_id,omitempty"`
	CorrectionReason       string                     `bson:"correction_reason,omitempty"`
	LoggedAt               time.Time                  `bson:"logged_at_utc"`
	TrackingDay            string                     `bson:"tracking_day"`
	TimezoneAtLogging      string                     `bson:"timezone_at_logging"`
	DailyResetTimeAtLog    string                     `bson:"daily_reset_time_at_logging"`
	TrackingDayRuleVersion int                        `bson:"tracking_day_rule_version"`
	MealCategory           string                     `bson:"meal_category"`
	Label                  string                     `bson:"label"`
	Items                  []foodLogItemDoc           `bson:"items"`
	Aggregate              map[string]foodNutrientDoc `bson:"aggregate_final_nutrition_snapshot"`
	CalculationVersion     string                     `bson:"calculation_version"`
	ExpiresAt              time.Time                  `bson:"expires_at"`
	ConsumedAt             *time.Time                 `bson:"consumed_at,omitempty"`
	CreatedAt              time.Time                  `bson:"created_at"`
}

type dailyNutritionSummaryDoc struct {
	ID                  primitive.ObjectID         `bson:"_id,omitempty"`
	UserID              primitive.ObjectID         `bson:"user_id"`
	TrackingDay         string                     `bson:"tracking_day"`
	TimezoneAtSummary   string                     `bson:"timezone_at_summary"`
	DailyResetTime      string                     `bson:"daily_reset_time_at_summary"`
	Aggregate           map[string]foodNutrientDoc `bson:"aggregate_final_nutrition_snapshot"`
	ConfirmedGroupCount int                        `bson:"confirmed_group_count"`
	ConfirmedItemCount  int                        `bson:"confirmed_item_count"`
	EarliestLoggedAtUTC time.Time                  `bson:"earliest_logged_at_utc,omitempty"`
	LatestLoggedAtUTC   time.Time                  `bson:"latest_logged_at_utc,omitempty"`
	SchemaVersion       int                        `bson:"schema_version"`
	DetailCompactedAt   time.Time                  `bson:"detail_compacted_at"`
	SourceDetailThrough time.Time                  `bson:"source_detail_through_utc"`
	CreatedAt           time.Time                  `bson:"created_at"`
}

type healthItemDoc struct {
	ID                     primitive.ObjectID `bson:"_id,omitempty"`
	UserID                 primitive.ObjectID `bson:"user_id"`
	Type                   string             `bson:"type"`
	Name                   string             `bson:"name"`
	DosageAmountText       string             `bson:"dosage_amount_text"`
	DosageUnitText         string             `bson:"dosage_unit_text"`
	Instructions           string             `bson:"instructions"`
	Notes                  string             `bson:"notes"`
	Active                 bool               `bson:"active"`
	StartDate              string             `bson:"start_date"`
	EndDate                string             `bson:"end_date,omitempty"`
	CurrentScheduleVersion int64              `bson:"current_schedule_version"`
	Version                int64              `bson:"version"`
	CreatedAt              time.Time          `bson:"created_at"`
	UpdatedAt              time.Time          `bson:"updated_at"`
}

type healthScheduleDoc struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	UserID            primitive.ObjectID `bson:"user_id"`
	HealthItemID      primitive.ObjectID `bson:"health_item_id"`
	ScheduleVersion   int64              `bson:"schedule_version"`
	Timezone          string             `bson:"timezone"`
	Frequency         string             `bson:"frequency"`
	Interval          int                `bson:"interval"`
	Weekdays          []string           `bson:"weekdays"`
	LocalTimes        []string           `bson:"local_times"`
	EffectiveFromDate string             `bson:"effective_from_date"`
	EffectiveToDate   string             `bson:"effective_to_date,omitempty"`
	CreatedAt         time.Time          `bson:"created_at"`
}

type healthEventDoc struct {
	ID              primitive.ObjectID `bson:"_id,omitempty"`
	UserID          primitive.ObjectID `bson:"user_id"`
	HealthItemID    primitive.ObjectID `bson:"health_item_id"`
	ScheduleVersion int64              `bson:"schedule_version"`
	OccurrenceID    string             `bson:"occurrence_id"`
	ScheduledAt     time.Time          `bson:"scheduled_at_utc"`
	LocalDate       string             `bson:"scheduled_local_date"`
	LocalTime       string             `bson:"scheduled_local_time"`
	Timezone        string             `bson:"timezone"`
	Status          string             `bson:"status"`
	CompletedAt     *time.Time         `bson:"completed_at_utc,omitempty"`
	RescheduledTo   *time.Time         `bson:"rescheduled_to_utc,omitempty"`
	Notes           string             `bson:"notes"`
	CreatedAt       time.Time          `bson:"created_at"`
	UpdatedAt       time.Time          `bson:"updated_at"`
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
		"account_deletions": {
			{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "status", Value: 1}, {Key: "purge_after", Value: 1}}},
		},
		"user_profiles":            {{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"user_preferences":         {{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"nutrition_reference_sets": {{Keys: bson.D{{Key: "reference_key", Value: 1}, {Key: "reference_version", Value: 1}}, Options: options.Index().SetUnique(true)}},
		"nutrition_target_versions": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "revision", Value: -1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "status", Value: 1}, {Key: "effective_from_tracking_day", Value: -1}}},
			{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetName("user_id_1").SetUnique(true).SetPartialFilterExpression(bson.M{"status": "ACTIVE"})},
			{Keys: bson.D{{Key: "user_id", Value: 1}}, Options: options.Index().SetName("target_proposed_user").SetUnique(true).SetPartialFilterExpression(bson.M{"status": "PROPOSED"})},
		},
		"usda_food_cache": {
			{Keys: bson.D{{Key: "cache_key", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		},
		"custom_foods": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "_id", Value: -1}}},
		},
		"meals": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "_id", Value: -1}}},
		},
		"serving_definitions": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "_id", Value: -1}}},
		},
		"food_favourites": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "food_key", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "_id", Value: -1}}},
		},
		"food_log_previews": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
			{Keys: bson.D{{Key: "expires_at", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		},
		"food_log_groups": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "idempotency_key", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "tracking_day", Value: 1}, {Key: "status", Value: 1}, {Key: "logged_at_utc", Value: -1}, {Key: "_id", Value: -1}}},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "root_group_id", Value: 1}, {Key: "revision", Value: -1}}, Options: options.Index().SetUnique(true)},
		},
		"food_logs": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "food_log_group_id", Value: 1}, {Key: "item_position", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "tracking_day", Value: 1}, {Key: "logged_at_utc", Value: -1}, {Key: "_id", Value: -1}}},
		},
		"daily_nutrition_summaries": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "tracking_day", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "tracking_day", Value: -1}, {Key: "_id", Value: -1}}},
		},
		"health_items": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "active", Value: 1}, {Key: "start_date", Value: 1}, {Key: "_id", Value: 1}}},
		},
		"health_item_schedule_versions": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "health_item_id", Value: 1}, {Key: "schedule_version", Value: 1}}, Options: options.Index().SetUnique(true)},
		},
		"health_item_events": {
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "occurrence_id", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "scheduled_at_utc", Value: -1}}},
		},
	}
	for collection, indexes := range models {
		if _, err := db.Collection(collection).Indexes().CreateMany(ctx, indexes); err != nil {
			return err
		}
	}
	return nil
}
