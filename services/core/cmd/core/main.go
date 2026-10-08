package main

import (
	"context"
	"encoding/base64"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/architsangal/AteThat/services/core/internal/security"
	"github.com/architsangal/AteThat/services/core/internal/server"
)

func required(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("missing %s", name)
	}
	return value
}

func main() {
	key, err := security.SigningKey(required("CORE_SIGNING_SEED_B64"))
	if err != nil {
		log.Fatal(err)
	}
	secret, err := base64.StdEncoding.DecodeString(required("INTERNAL_SERVICE_SECRET_B64"))
	if err != nil || len(secret) < 32 {
		log.Fatal("invalid INTERNAL_SERVICE_SECRET_B64")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cfg := server.Config{
		MongoURI: required("MONGODB_URI"), Database: required("MONGODB_DATABASE"),
		ListenAddr: env("CORE_LISTEN_ADDR", ":9090"), SigningKey: key, InternalSecret: secret,
		FrontendURL: required("FRONTEND_ORIGIN"), SMTPHost: required("SMTP_HOST"),
		SMTPPort: env("SMTP_PORT", "1025"), SMTPFrom: required("SMTP_FROM"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"), SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"), GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURI: os.Getenv("GOOGLE_REDIRECT_URI"),
		USDAAPIKey:        env("USDA_API_KEY", "DEMO_KEY"), USDAAPIBaseURL: env("USDA_API_BASE_URL", "https://api.nal.usda.gov/fdc/v1"),
	}
	core, err := server.New(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer core.Client.Disconnect(context.Background())
	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := core.GRPC()
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			log.Printf("gRPC server stopped: %v", err)
		}
	}()
	log.Printf("Core listening on %s", cfg.ListenAddr)
	retentionCtx, stopRetention := context.WithCancel(context.Background())
	go core.RunRetentionLoop(retentionCtx)
	go core.RunAccountDeletionPurgeLoop(retentionCtx)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	stopRetention()
	grpcServer.GracefulStop()
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
