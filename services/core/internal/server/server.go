package server

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/architsangal/AteThat/services/core/internal/gen/atethat/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Config struct {
	MongoURI           string
	Database           string
	ListenAddr         string
	SigningKey         ed25519.PrivateKey
	InternalSecret     []byte
	FrontendURL        string
	SMTPHost           string
	SMTPPort           string
	SMTPFrom           string
	SMTPUsername       string
	SMTPPassword       string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string
}

type Server struct {
	pb.UnimplementedIdentityServiceServer
	pb.UnimplementedProfileServiceServer
	DB     *mongo.Database
	Client *mongo.Client
	Cfg    Config
	nonces sync.Map
}

func New(ctx context.Context, cfg Config) (*Server, error) {
	if len(cfg.SigningKey) != ed25519.PrivateKeySize || len(cfg.InternalSecret) < 32 {
		return nil, errors.New("signing key and internal credential must be configured")
	}
	if cfg.MongoURI == "" || cfg.Database == "" || cfg.FrontendURL == "" || cfg.SMTPHost == "" || cfg.SMTPFrom == "" {
		return nil, errors.New("database, frontend URL, and SMTP configuration are required")
	}
	if parsed, err := url.Parse(cfg.FrontendURL); err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("invalid frontend URL")
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	db := client.Database(cfg.Database)
	if err := createIndexes(ctx, db); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	return &Server{DB: db, Client: client, Cfg: cfg}, nil
}

func (s *Server) GRPC() *grpc.Server {
	server := grpc.NewServer(grpc.UnaryInterceptor(s.authenticateRPC))
	pb.RegisterIdentityServiceServer(server, s)
	pb.RegisterProfileServiceServer(server, s)
	return server
}

type principalKey struct{}

func Principal(ctx context.Context) (primitive.ObjectID, error) {
	value, ok := ctx.Value(principalKey{}).(primitive.ObjectID)
	if !ok || value.IsZero() {
		return primitive.NilObjectID, status.Error(codes.Unauthenticated, "missing principal")
	}
	return value, nil
}

func (s *Server) authenticateRPC(ctx context.Context, request interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing service credential")
	}
	get := func(key string) string {
		values := md.Get(key)
		if len(values) == 1 {
			return values[0]
		}
		return ""
	}
	ts, userID, family, nonce, sig := get("x-atethat-ts"), get("x-atethat-user"), get("x-atethat-session"), get("x-atethat-nonce"), get("x-atethat-sig")
	when, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || nonce == "" || sig == "" || time.Since(time.Unix(when, 0)) > 30*time.Second || time.Until(time.Unix(when, 0)) > 30*time.Second {
		return nil, status.Error(codes.Unauthenticated, "invalid service credential")
	}
	mac := hmac.New(sha256.New, s.Cfg.InternalSecret)
	_, _ = fmt.Fprintf(mac, "%s|%s|%s|%s|%s", info.FullMethod, ts, userID, family, nonce)
	want := mac.Sum(nil)
	got, err := hex.DecodeString(sig)
	if err != nil || !hmac.Equal(got, want) {
		return nil, status.Error(codes.Unauthenticated, "invalid service credential")
	}
	if _, loaded := s.nonces.LoadOrStore(nonce, time.Now().Add(time.Minute)); loaded {
		return nil, status.Error(codes.Unauthenticated, "replayed service credential")
	}
	// The replay window is small; remove nonce after its credential expires.
	time.AfterFunc(time.Minute, func() { s.nonces.Delete(nonce) })
	if userID != "" {
		if family == "" {
			return nil, status.Error(codes.Unauthenticated, "missing session")
		}
		id, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid principal")
		}
		filter := bson.M{"user_id": id, "family_id": family, "status": "ACTIVE", "expires_at": bson.M{"$gt": time.Now().UTC()}}
		if err := s.DB.Collection("sessions").FindOne(ctx, filter).Err(); err != nil {
			return nil, status.Error(codes.Unauthenticated, "session revoked or expired")
		}
		ctx = context.WithValue(ctx, principalKey{}, id)
	} else if family != "" {
		return nil, status.Error(codes.Unauthenticated, "invalid session context")
	}
	return handler(ctx, request)
}

func validEmail(raw string) (string, bool) {
	if len(raw) < 3 || len(raw) > 254 || strings.ContainsAny(raw, "\r\n") {
		return "", false
	}
	address, err := mail.ParseAddress(raw)
	if err != nil || address.Address != raw || strings.Contains(raw, " ") {
		return "", false
	}
	return strings.ToLower(raw), true
}

func validTimezone(zone string) bool {
	if len(zone) > 100 || zone == "" {
		return false
	}
	_, err := time.LoadLocation(zone)
	return err == nil
}

func validPassword(password string) bool { return len(password) >= 12 && len(password) <= 128 }

func (s *Server) sendEmail(ctx context.Context, to, subject, path, token string) error {
	base, err := url.Parse(s.Cfg.FrontendURL)
	if err != nil {
		return err
	}
	base.Path = path
	base.RawQuery = url.Values{"token": {token}}.Encode()
	message := "From: " + s.Cfg.SMTPFrom + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" +
		"Open this one-time AteThat link:\r\n" + base.String() + "\r\n\r\nIf you did not request this, you can ignore it.\r\n"
	address := net.JoinHostPort(s.Cfg.SMTPHost, s.Cfg.SMTPPort)
	conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline := time.Now().Add(8 * time.Second)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	_ = conn.SetDeadline(deadline)
	client, err := smtp.NewClient(conn, s.Cfg.SMTPHost)
	if err != nil {
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.Cfg.SMTPHost, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	} else if s.Cfg.SMTPUsername != "" {
		return errors.New("authenticated SMTP requires STARTTLS")
	}
	if s.Cfg.SMTPUsername != "" {
		if err := client.Auth(smtp.PlainAuth("", s.Cfg.SMTPUsername, s.Cfg.SMTPPassword, s.Cfg.SMTPHost)); err != nil {
			return err
		}
	}
	if err := client.Mail(s.Cfg.SMTPFrom); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write([]byte(message)); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func invalid(message string) error { return status.Error(codes.InvalidArgument, message) }
func internal() error              { return status.Error(codes.Internal, "internal service error") }
