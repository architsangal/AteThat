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
	if response.StatusCode == http.StatusNotFound {
		return nil, status.Error(codes.NotFound, "USDA food not found")
	}
	if response.StatusCode != http.StatusOK {
		return nil, status.Error(codes.Unavailable, "USDA food provider unavailable")
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
	if _, err := Principal(ctx); err != nil {
		return nil, err
	}
	query := strings.Join(strings.Fields(strings.TrimSpace(req.Query)), " ")
	if len(query) < 2 || len(query) > 100 || req.Limit == 0 || req.Limit > 50 {
		return nil, invalid("query must be 2–100 characters and limit must be 1–50")
	}
	state := strings.ToUpper(req.FoodState)
	if state != "" && state != "RAW" && state != "COOKED" && state != "PREPARED" && state != "UNKNOWN" {
		return nil, invalid("unsupported food state")
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
