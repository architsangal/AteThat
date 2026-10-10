.PHONY: local-up local-run local-down local-status local-config contracts-check generate test-go

local-up:
	docker compose -f deploy/local/compose.yaml up -d --wait

local-config:
	go run scripts/local-config.go

local-run:
	@test -f deploy/local/.env || (echo "Run make local-config first"; exit 1)
	docker compose -f deploy/local/compose.yaml -f deploy/local/compose.app.yaml up -d --build --wait

local-down:
	docker compose -f deploy/local/compose.yaml -f deploy/local/compose.app.yaml down

local-status:
	docker compose -f deploy/local/compose.yaml -f deploy/local/compose.app.yaml ps

contracts-check:
	sh scripts/check-contracts.sh

generate:
	sh scripts/generate-proto.sh

test-go:
	cd services/core && go test ./...
	cd services/api && go test ./...
