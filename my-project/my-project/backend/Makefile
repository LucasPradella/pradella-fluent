.PHONY: lint test test-integration coverage sqlc run

lint:
	golangci-lint run
	go vet ./...
	govulncheck ./...

test:
	go test ./...

test-integration:
	go test -tags=integration ./...

# Coverage gate: >=80% on internal/domain and internal/usecase (constitution II).
# Contract tests in adapter/http also exercise the usecases, so coverage is
# measured across the whole suite but attributed to domain/usecase only.
coverage:
	go test ./... -coverprofile=cover.out -coverpkg=./internal/domain/...,./internal/usecase/...
	@go tool cover -func=cover.out | tail -1
	@total=$$(go tool cover -func=cover.out | tail -1 | awk '{gsub(/%/,"",$$3); print $$3}'); \
	awk -v t="$$total" 'BEGIN { if (t+0 < 80.0) { printf "coverage %.1f%% is below the 80%% gate\n", t; exit 1 } else { printf "coverage %.1f%% meets the 80%% gate\n", t } }'

sqlc:
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate

run:
	go run ./cmd/api -migrate -seed
