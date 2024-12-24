ifndef GOPATH
	GOPATH := $(shell go env GOPATH)
endif
ifndef GOBIN # derive value from gopath (default to first entry, similar to 'go get')
	GOBIN := $(shell go env GOPATH | sed 's/:.*//')/bin
endif

tools = $(addprefix $(GOBIN)/, golangci-lint goimports gci)
deps = $(addprefix $(GOBIN)/, wire)

ifneq (,$(wildcard ./.local_runner.env))
    include ./.local_runner.env
    export
endif

.DEFAULT_GOAL := all

###############################################################################
#
# Initialization
#
###############################################################################

init: ## Install git hooks to perform pre-commit checks
	git config core.hooksPath .githooks
	git config commit.template .gitmessage

dep: $(deps) ## Install the deps required to generate code and build Harness
	@echo "Installing dependencies"
	@go mod download

tools: $(tools) ## Install tools required for the build
	@echo "Installed tools"

###############################################################################
# Code Generation
#
# Some code generation can be slow, so we only run it if
# the source file has changed.
###############################################################################

generate: wire
	@echo "Generated Code"

wire: cli/wire_gen.go

force-wire: ## Force wire code generation
	@sh ./scripts/wire/runner.sh

cli/wire_gen.go: cli/wire.go
	@sh ./scripts/wire/runner.sh

###############################################################################
#
# Runner Build and testing rules
#
###############################################################################

build: ## Build the all-in-one Harness binary
	@echo "Building Harness Runner"
	go build -o ./runner


test:  ## Run the go tests
	@echo "Running tests"
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out


###############################################################################
#
# Code Formatting and linting
#
###############################################################################

format: tools # Format go code and error if any changes are made
	@echo "Formatting ..."
	@goimports -w .
	@gci write --skip-generated --custom-order -s standard -s "prefix(github.com/harness/gitness)" -s default -s blank -s dot .
	@echo "Formatting complete"

lint: tools # lint the golang code
	@echo "Linting $(1)"
	@golangci-lint run --timeout=3m --verbose

###############################################################################
# Install Tools and deps
#
# These targets specify the full path to where the tool is installed
# If the tool already exists it wont be re-installed.
###############################################################################

update-tools: delete-tools $(tools) ## Update the tools by deleting and re-installing

delete-tools: ## Delete the tools
	@rm $(tools) || true

# Install golangci-lint
$(GOBIN)/golangci-lint:
	@echo "🔘 Installing golangci-lint... (`date '+%H:%M:%S'`)"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) v1.56.2

# Install goimports to format code
$(GOBIN)/goimports:
	@echo "🔘 Installing goimports ... (`date '+%H:%M:%S'`)"
	@go install golang.org/x/tools/cmd/goimports

$(GOBIN)/gci:
	go install github.com/daixiang0/gci@v0.13.1	


help: ## show help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% 0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: delete-tools update-tools help format lint

###############################################################################
# Install Tools and deps
#
# These targets specify the full path to where the tool is installed
# If the tool already exists it wont be re-installed.
###############################################################################

# Install wire to generate dependency injection
$(GOBIN)/wire:
	go install github.com/google/wire/cmd/wire@latest
