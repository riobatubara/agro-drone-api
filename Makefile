.PHONY: clean all init generate generate_mocks test test_api

all: build/main

build/main: cmd/main.go generated generate_mocks
	@echo "Building application binary..."
	go build -o $@ ./cmd/...

clean:
	@echo "Cleaning up generated assets..."
	rm -rf generated
	rm -f repository/*.mock.gen.go
	rm -f coverage.out

init: clean generate
	go mod tidy
	go mod vendor

test: generate_mocks
	@echo "Executing unit tests with coverage tracking..."
	go clean -testcache
	go test -short -coverprofile=coverage.out -v ./handler/... ./repository/...

test_api:
	@echo "Executing contract and API integration tests..."
	go clean -testcache
	go test -v ./tests/...

generate: generated generate_mocks

generated: api.yml
	@echo "Generating OpenAPI v3 Echo engine stubs..."
	mkdir -p generated
	oapi-codegen --package generated -generate types,server,spec $< > generated/api.gen.go

# Find targeted source interface files across repository paths
INTERFACES_GO_FILES := $(shell find repository -name "interfaces.go")
INTERFACES_GEN_GO_FILES := $(INTERFACES_GO_FILES:%.go=%.mock.gen.go)

generate_mocks: $(INTERFACES_GEN_GO_FILES)

$(INTERFACES_GEN_GO_FILES): %.mock.gen.go: %.go
	@echo "Generating mock blueprint $@ for source interface $<"
	mockgen -source=$< -destination=$@ -package=repository
