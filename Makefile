# Makefile
all: help

## help: to show help
.PHONY: help
help: Makefile
	@echo
	@echo "-- GO DB STORE --"
	@echo "Choose a make command to run"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo


## test: run tests
.PHONY: test
test:
	go test ./...

## lint: run golang linter
.PHONY: lint
lint:
	@command -v golangci-lint > /dev/null || { \
		echo "golangci-lint não está instalado."; \
		echo "Needs golangci-lint installed"; \
		echo "curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b \(go env GOPATH)/bin v2.1.6"; \
		exit 1; \
	}
	golangci-lint run

## tag: create tag release
.PHONY: tag
tag:
	@read -p "Enter version (ex: v1.0.0): " version; \
	git tag $$version; \
	git push origin $$version
