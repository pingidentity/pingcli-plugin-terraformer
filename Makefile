TEST?=$$(go list ./...)
PKG_NAME=pingcli-terraformer
BINARY=pingcli-terraformer
VERSION=0.1.0

GOBIN?=$$(go env GOPATH)/bin

default: install

build:
	@echo "==> Building..."
	go mod tidy
	go build -v -o $(BINARY) .

install: build
	@echo "==> Installing..."
	@cp $(BINARY) $(GOBIN)/$(BINARY)
	@echo "Installed $(BINARY) to $(GOBIN)/$(BINARY)"

test: build
	@echo "==> Running unit tests..."
	go test $(TEST) -v $(TESTARGS) -timeout=5m

testacc: build
	@echo "==> Running acceptance tests..."
	go test -tags acceptance $(TEST) -v $(TESTARGS) -timeout 120m

testcoverage: build
	@echo "==> Running tests with coverage..."
	go test -tags acceptance -coverprofile=coverage.out $(TEST) $(TESTARGS) -timeout=120m
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

vet:
	@echo "==> Running go vet..."
	@go vet ./... ; if [ $$? -ne 0 ]; then \
		echo ""; \
		echo "Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
		exit 1; \
	fi

depscheck:
	@echo "==> Checking source code with go mod tidy..."
	@go mod tidy
	@git diff --exit-code -- go.mod go.sum || \
		(echo; echo "Unexpected difference in go.mod/go.sum files. Run 'go mod tidy' command or revert any go.mod/go.sum changes and commit."; exit 1)

lint: golangcilint

golangcilint:
	@echo "==> Checking source code with golangci-lint..."
	@golangci-lint run ./...

fmt:
	@echo "==> Formatting Go code..."
	@go fmt ./...

clean:
	@echo "==> Cleaning build artifacts..."
	rm -f $(BINARY)
	rm -f coverage.out coverage.html
	go clean -testcache

devcheck: build vet fmt lint test testacc

devchecknotest: build vet fmt lint test

.PHONY: build install test testacc testcoverage vet depscheck lint golangcilint fmt clean devcheck devchecknotest
