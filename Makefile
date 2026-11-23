.PHONY: test vet fmt-check check build run

test:
	go test ./...

vet:
	go vet ./...

fmt-check:
	@test -z "$$(gofmt -l $$(find cmd internal -name '*.go'))" || (echo "Run gofmt" && exit 1)

check: fmt-check test vet
	python3 scripts/check_public.py

build:
	go build -trimpath -o bin/warden ./cmd/warden

run:
	go run ./cmd/warden -config config.yaml
