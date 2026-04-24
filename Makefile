GOCACHE_DIR := /tmp/go-build-cache

.PHONY: run test build fmt

run:
	GOCACHE=$(GOCACHE_DIR) go run ./cmd/chat-trace-viewer

test:
	GOCACHE=$(GOCACHE_DIR) go test ./...

build:
	GOCACHE=$(GOCACHE_DIR) go build ./...

fmt:
	gofmt -w cmd/chat-trace-viewer/main.go internal/config/config.go internal/config/config_test.go internal/config/envfile.go internal/domain/models.go internal/httpapi/server.go internal/logger/logger.go internal/normalizer/normalizer.go internal/normalizer/normalizer_test.go internal/parser/raw.go internal/parser/subject.go internal/parser/subject_test.go internal/service/chat_trace.go internal/service/chat_trace_test.go internal/timeline/builder.go internal/victorialogs/client.go internal/victorialogs/client_test.go
