package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"log"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/architsangal/AteThat/services/api/internal/server"
)

func required(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("missing %s", name)
	}
	return value
}

func main() {
	public, err := base64.StdEncoding.DecodeString(required("API_VERIFY_PUBLIC_KEY_B64"))
	if err != nil || len(public) != ed25519.PublicKeySize {
		log.Fatal("invalid API_VERIFY_PUBLIC_KEY_B64")
	}
	secret, err := base64.StdEncoding.DecodeString(required("INTERNAL_SERVICE_SECRET_B64"))
	if err != nil || len(secret) < 32 {
		log.Fatal("invalid INTERNAL_SERVICE_SECRET_B64")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	api, err := server.New(ctx, server.Config{
		CoreAddress: required("CORE_GRPC_ADDR"), FrontendOrigin: required("FRONTEND_ORIGIN"),
		VerifyKey: ed25519.PublicKey(public), InternalSecret: secret, CookieSecure: os.Getenv("COOKIE_SECURE") != "false", TrustedProxyCIDRs: trustedProxyCIDRs(),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer api.Conn.Close()
	server := &http.Server{Addr: env("API_LISTEN_ADDR", ":8080"), Handler: api, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server stopped: %v", err)
		}
	}()
	log.Printf("API listening on %s", server.Addr)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
}

func trustedProxyCIDRs() []netip.Prefix {
	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXY_CIDRS"))
	if raw == "" {
		return nil
	}
	prefixes := make([]netip.Prefix, 0, len(strings.Split(raw, ",")))
	for _, item := range strings.Split(raw, ",") {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(item))
		if err != nil {
			log.Fatalf("invalid TRUSTED_PROXY_CIDRS entry %q", item)
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}

func env(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
