package server

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/architsangal/AteThat/services/core/internal/foods"
	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const usdaCacheTTL = 24 * time.Hour // finite; below the project-wide 15-day cap

type usdaCacheDoc struct {
	CacheKey  string    `bson:"cache_key"`
	Payload   []byte    `bson:"payload"`
	ExpiresAt time.Time `bson:"expires_at"`
}

func (s *Server) usdaJSON(ctx context.Context, cacheKey, endpoint string) ([]byte, error) {
	var cached usdaCacheDoc
	if err := s.DB.Collection("usda_food_cache").FindOne(ctx, bson.M{"cache_key": cacheKey,
		"expires_at": bson.M{"$gt": time.Now().UTC()}}).Decode(&cached); err == nil {
		return cached.Payload, nil
	} else if err != mongo.ErrNoDocuments {
		return nil, status.Error(codes.Unavailable, "food cache unavailable")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid food provider request")
	}
	client := &http.Client{Timeout: 6 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "USDA food provider unavailable")
	}
	defer response.Body.Close()
	if err := usdaHTTPError(response.StatusCode, s.Cfg.USDAAPIKey == "DEMO_KEY"); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 3<<20))
	if err != nil || len(data) == 0 || !json.Valid(data) {
		return nil, status.Error(codes.Unavailable, "USDA food provider returned invalid data")
	}
	_, _ = s.DB.Collection("usda_food_cache").UpdateOne(ctx, bson.M{"cache_key": cacheKey},
		bson.M{"$set": usdaCacheDoc{CacheKey: cacheKey, Payload: data, ExpiresAt: time.Now().UTC().Add(usdaCacheTTL)}},
		options.Update().SetUpsert(true))
	return data, nil
}

func usdaFoodPB(food foods.Food, state string) *pb.FoodResult {
	nutrients := make(map[string]*pb.FoodNutrient, len(food.Nutrients))
	for key, value := range food.Nutrients {
		nutrients[key] = &pb.FoodNutrient{Status: value.Status, Amount: value.Amount, Unit: value.Unit}
	}
	return &pb.FoodResult{ProviderFoodId: food.ID, Name: food.Name, FoodState: state,
		BaseAmount: "100", BaseUnit: "g", Nutrients: nutrients, DataType: food.DataType,
		SourceUrl: "https://fdc.nal.usda.gov/food-details/" + food.ID + "/nutrients"}
}

type foodCursor struct {
	Version int    `json:"v"`
	Query   string `json:"q"`
	Page    int    `json:"p"`
}

func (s *Server) SearchUSDA(ctx context.Context, req *pb.SearchUSDARequest) (*pb.FoodSearchPage, error) {
	userID, authErr := Principal(ctx)
	if authErr != nil {
		return nil, authErr
	}
	source := strings.ToUpper(req.Source)
	if source == "STATUS" {
		return s.foodSearchStats(ctx, userID.Hex(), &pb.FoodSearchPage{})
	}
	if source != "" && source != "USDA" && source != "LOCAL" {
		return nil, invalid("unsupported search source")
	}
	query := strings.Join(strings.Fields(strings.TrimSpace(req.Query)), " ")
	if len(query) < 2 || len(query) > 100 || req.Limit == 0 || req.Limit > 50 {
		return nil, invalid("query must be 2–100 characters and limit must be 1–50")
	}
	state := strings.ToUpper(req.FoodState)
	if state != "" && state != "RAW" && state != "COOKED" && state != "PREPARED" && state != "UNKNOWN" {
		return nil, invalid("unsupported food state")
	}
	if source == "LOCAL" {
		return s.searchCatalogue(ctx, userID.Hex(), query, state, req)
	}
	queryHash := sha256.Sum256([]byte(strings.ToLower(query) + "|" + state + "|" + strconv.FormatUint(uint64(req.Limit), 10)))
	queryKey := hex.EncodeToString(queryHash[:])[:24]
	page := 1
	if req.Cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(req.Cursor)
		if err != nil {
			return nil, invalid("invalid food cursor")
		}
		var cursor foodCursor
		if len(decoded) > 256 || json.Unmarshal(decoded, &cursor) != nil || cursor.Version != 1 || cursor.Query != queryKey || cursor.Page < 2 || cursor.Page > 1000 {
			return nil, invalid("invalid food cursor")
		}
		page = cursor.Page
	}
	usageID := userID.Hex() + ":" + time.Now().UTC().Format("2006-01-02")
	// The primary key and conditional increment serialize simultaneous requests.
	var reservation searchUsage
	err := s.DB.Collection("food_search_usage").FindOneAndUpdate(ctx,
		bson.M{"_id": usageID, "usda": bson.M{"$lt": 10}},
		bson.M{"$inc": bson.M{"usda": 1}, "$setOnInsert": bson.M{"expires_at": time.Now().UTC().Add(48 * time.Hour)}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&reservation)
	if mongo.IsDuplicateKeyError(err) || err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.ResourceExhausted, "Your 10 daily USDA searches have been used. Search AteThat foods or try again after midnight UTC.")
	}
	if err != nil {
		return nil, status.Error(codes.Unavailable, "Search allowance unavailable")
	}
	success := false
	defer func() {
		if !success {
			refundCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, _ = s.DB.Collection("food_search_usage").UpdateOne(refundCtx, bson.M{"_id": usageID}, bson.M{"$inc": bson.M{"usda": -1}})
		}
	}()
	search := query
	if state == "RAW" || state == "COOKED" {
		search += " " + strings.ToLower(state)
	}
	params := url.Values{"api_key": {s.Cfg.USDAAPIKey}, "query": {search},
		"pageSize": {strconv.FormatUint(uint64(req.Limit), 10)}, "pageNumber": {strconv.Itoa(page)}}
	endpoint := strings.TrimSuffix(s.Cfg.USDAAPIBaseURL, "/") + "/foods/search?" + params.Encode()
	data, err := s.usdaJSON(ctx, fmt.Sprintf("search:%s:%d", queryKey, page), endpoint)
	if err != nil {
		return nil, err
	}
	items, total, err := foods.DecodeSearch(data)
	if err != nil {
		return nil, status.Error(codes.Unavailable, "USDA food provider returned invalid data")
	}
	result := &pb.FoodSearchPage{Items: make([]*pb.FoodResult, 0, len(items))}
	for _, item := range items {
		result.Items = append(result.Items, usdaFoodPB(item, stateOrUnknown(state)))
	}
	result.HasMore = int64(page)*int64(req.Limit) < total && len(items) > 0 && page < 1000
	if result.HasMore {
		encoded, _ := json.Marshal(foodCursor{Version: 1, Query: queryKey, Page: page + 1})
		result.NextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	for _, item := range items {
		_, err := s.Client.Database(s.Cfg.Database+"_foods").Collection("foods").UpdateOne(ctx, bson.M{"_id": item.ID}, bson.M{"$set": bson.M{"food": usdaFoodPB(item, "UNKNOWN"), "name": item.Name, "fetched_at": time.Now().UTC()}}, options.Update().SetUpsert(true))
		if err != nil {
			return nil, status.Error(codes.Unavailable, "Food search is temporarily unavailable. Please try again later.")
		}
	}
	result.MatchedCount = uint64(total)
	result, err = s.foodSearchStats(ctx, userID.Hex(), result)
	if err != nil {
		return nil, err
	}
	success = true
	return result, nil
}

func stateOrUnknown(state string) string {
	if state == "" {
		return "UNKNOWN"
	}
	return state
}

func (s *Server) GetUSDAFood(ctx context.Context, req *pb.GetUSDAFoodRequest) (*pb.FoodResult, error) {
	if _, err := Principal(ctx); err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(req.ProviderFoodId, 10, 64)
	if err != nil || id <= 0 {
		return nil, invalid("invalid USDA food ID")
	}
	params := url.Values{"api_key": {s.Cfg.USDAAPIKey}}
	endpoint := strings.TrimSuffix(s.Cfg.USDAAPIBaseURL, "/") + "/food/" + strconv.FormatInt(id, 10) + "?" + params.Encode()
	data, err := s.usdaJSON(ctx, "detail:"+strconv.FormatInt(id, 10), endpoint)
	if err != nil {
		return nil, err
	}
	food, err := foods.DecodeDetail(data)
	if err != nil || food.ID != strconv.FormatInt(id, 10) {
		return nil, status.Error(codes.Unavailable, "USDA food provider returned invalid data")
	}
	return usdaFoodPB(food, "UNKNOWN"), nil
}

type searchUsage struct {
	USDA  int64 `bson:"usda"`
	Local int64 `bson:"local"`
}

func (s *Server) foodSearchStats(ctx context.Context, user string, result *pb.FoodSearchPage) (*pb.FoodSearchPage, error) {
	now := time.Now().UTC()
	var usage searchUsage
	err := s.DB.Collection("food_search_usage").FindOne(ctx, bson.M{"_id": user + ":" + now.Format("2006-01-02")}).Decode(&usage)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, internal()
	}
	count, err := s.Client.Database(s.Cfg.Database+"_foods").Collection("foods").CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, internal()
	}
	result.UsdaLimit = 10
	result.UsdaRemaining = uint32(max(int64(0), 10-usage.USDA))
	result.LocalSearchesToday = uint64(max(int64(0), usage.Local))
	result.CatalogueCount = uint64(count)
	result.ResetsAt = time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	return result, nil
}
func (s *Server) searchCatalogue(ctx context.Context, user, query, state string, req *pb.SearchUSDARequest) (*pb.FoodSearchPage, error) {
	filter := bson.M{"name": bson.M{"$regex": regexp.QuoteMeta(query), "$options": "i"}}
	if state != "" && state != "UNKNOWN" {
		filter["$and"] = bson.A{bson.M{"name": bson.M{"$regex": regexp.QuoteMeta(strings.ToLower(state)), "$options": "i"}}}
	}
	offset := int64(0)
	if req.Cursor != "" {
		n, err := strconv.ParseInt(req.Cursor, 10, 64)
		if err != nil || n < 0 || n > 100000 {
			return nil, invalid("invalid local cursor")
		}
		offset = n
	}
	total, err := s.Client.Database(s.Cfg.Database+"_foods").Collection("foods").CountDocuments(ctx, filter)
	if err != nil {
		return nil, internal()
	}
	cursor, err := s.Client.Database(s.Cfg.Database+"_foods").Collection("foods").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "name", Value: 1}, {Key: "_id", Value: 1}}).SetSkip(offset).SetLimit(int64(req.Limit)))
	if err != nil {
		return nil, internal()
	}
	defer cursor.Close(ctx)
	result := &pb.FoodSearchPage{Items: []*pb.FoodResult{}, MatchedCount: uint64(total)}
	for cursor.Next(ctx) {
		var doc struct {
			Food *pb.FoodResult `bson:"food"`
		}
		if err := cursor.Decode(&doc); err != nil {
			return nil, internal()
		}
		result.Items = append(result.Items, doc.Food)
	}
	if cursor.Err() != nil {
		return nil, internal()
	}
	result.HasMore = offset+int64(len(result.Items)) < total
	if result.HasMore {
		result.NextCursor = strconv.FormatInt(offset+int64(len(result.Items)), 10)
	}
	_, err = s.DB.Collection("food_search_usage").UpdateOne(ctx, bson.M{"_id": user + ":" + time.Now().UTC().Format("2006-01-02")}, bson.M{"$inc": bson.M{"local": 1}, "$setOnInsert": bson.M{"usda": 0, "expires_at": time.Now().UTC().Add(48 * time.Hour)}}, options.Update().SetUpsert(true))
	if err != nil {
		return nil, internal()
	}
	return s.foodSearchStats(ctx, user, result)
}

// Keep the provider's shared/IP limit distinct from the account's daily allowance.
// Only stable classifications cross RPC; never include the request URL or API key.
func usdaHTTPError(code int, demo bool) error {
	switch code {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return status.Error(codes.NotFound, "USDA food not found")
	case http.StatusTooManyRequests:
		if demo {
			return status.Error(codes.Unavailable, "USDA_DEMO_RATE_LIMITED")
		}
		return status.Error(codes.Unavailable, "USDA_PROVIDER_RATE_LIMITED")
	case http.StatusUnauthorized, http.StatusForbidden:
		return status.Error(codes.Unavailable, "USDA_PROVIDER_AUTH_FAILED")
	default:
		return status.Error(codes.Unavailable, "USDA food provider unavailable")
	}
}
