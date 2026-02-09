.PHONY: all build clean test rpc-test docs help

# --- Configuration ---
BIN_NAME := cc
MAIN_PACKAGE := ./cmd/cc
OUTPUT_DIR := bin

# --- Colors ---
RED := $(shell tput -Txterm setaf 1)
GREEN := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET := $(shell tput -Txterm sgr0)

# --- Targets ---
all: help

## build: Build the CC-Mono executable
build:
	@echo "$(GREEN)=== Building CC-Mono ===$(RESET)"
	@mkdir -p $(OUTPUT_DIR)
	@cd $(MAIN_PACKAGE) && go mod tidy && cd - && go build -o $(OUTPUT_DIR)/$(BIN_NAME) $(MAIN_PACKAGE)
	@echo "$(GREEN)=== Build complete! Executable created at: $(OUTPUT_DIR)/$(BIN_NAME)$(RESET)"

## clean: Remove the build binary
clean:
	@echo "$(RED)=== Cleaning up ===$(RESET)"
	@rm -f $(OUTPUT_DIR)/$(BIN_NAME)
	@echo "$(GREEN)=== Clean complete! ===$(RESET)"

## test: Run all tests
test:
	@echo "$(YELLOW)=== Running tests ===$(RESET)"
	@cd cmd/cc && go test -v .
	@cd internal/tui && go test -v .
	@cd pkg/agent && go test -v .
	@cd pkg/ai && go test -v .
	@cd pkg/codingagent && go test -v .
	@cd pkg/rpc && go test -v .
	@cd pkg/shared && go test -v .

## fmt: Format source code with go fmt
fmt:
	@echo "$(YELLOW)=== Formating code ===$(RESET)"
	@go fmt ./...
	@echo "$(GREEN)=== Code formatted successfully ===$(RESET)"

## rpc-test: Test the RPC functionality
rpc-test: build
	@echo "$(YELLOW)=== Testing RPC interface ===$(RESET)"
	@echo "$(GREEN)=== RPC server is available at ./bin/cc chat --mode rpc ===$(RESET)"

## docs: Generate documentation (placeholder)
docs:
	@echo "$(YELLOW)=== Documentation ===$(RESET)"
	@echo "$(GREEN)- Main documentation in README.md$(RESET)"
	@echo "$(GREEN)- RPC mode documentation in docs/RPC_MODE.md$(RESET)"
	@echo "$(GREEN)- Web UI documentation in web-ui/README.md$(RESET)"

## help: Show this help message
help:
	@echo "$(YELLOW) CC-Mono - AI Coding Assistant$(RESET)"
	@echo ""
	@echo "$(GREEN)Usage:$(RESET)"
	@echo "  make <target>"
	@echo ""
	@echo "$(GREEN)Targets:$(RESET)"
	@awk '/^## / { sub(/^## /, "", $$0); print; }' $(MAKEFILE_LIST) | sort -d | \
		while IFS= read -r line; do \
			key=$$(echo $$line | awk -F: '{print $$1}'); \
			desc=$$(echo $$line | sed 's/^[^:]*://'); \
			printf "  $(GREEN)%-20s$(RESET) %s\n" "$$key" "$$desc"; \
		done

## install: Build and install to GOPATH
install:
	@echo "$(GREEN)=== Installing CC-Mono ===$(RESET)"
	@cd $(MAIN_PACKAGE) && go install
	@echo "$(GREEN)=== Installation complete! ===$(RESET)"
