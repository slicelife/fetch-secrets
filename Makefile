.PHONY: test clean artifact lint generate

test:
	@go test ./...

clean:
	@go clean -i -cache

artifact:
	@go build

lint:
	@docker run -t --rm -v ./:/app -w /app golangci/golangci-lint golangci-lint run -v --timeout=300s

generate:
	@go install go.uber.org/mock/mockgen@latest
	@go generate ./...