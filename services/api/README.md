# Public API service (planned)

This is an independently buildable Go module. It owns the browser-facing `/api/v1` REST transport, bearer-token verification, request validation, CORS, refresh cookies, and translation to Core's private gRPC contract. It must not import `services/core/internal` or access MongoDB, USDA, or SMTP directly.

The first implemented contract is in `contracts/openapi/v1.yaml`. Phase 1 rate limiting, explicit Google account linking, account deletion, and production deployment hardening are not complete yet. Run `go test ./...` from this directory.
