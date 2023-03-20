SHELL := /bin/bash

.DEFAULT_GOAL := all
.PHONY: all
all: ## build pipeline
all: mod inst gen build spell lint test

.PHONY: ci
ci: ## CI build pipeline
ci: all diff

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: clean
clean: ## remove files created during build pipeline
	$(call print-target)
	rm -rf dist
	rm -f coverage.*
	rm -f '"$(shell go env GOCACHE)/../golangci-lint"'
	go clean -i -cache -testcache -modcache -fuzzcache -x
	rm -rf contrib/completion manpage $(OAS_FILE)

.PHONY: mod
mod: ## go mod tidy
	$(call print-target)
	go mod tidy
	cd tools && go mod tidy

.PHONY: inst
inst: ## go install tools
	$(call print-target)
	cd tools && go install $(shell cd tools && go list -f '{{ join .Imports " " }}' -tags=tools)

.PHONY: gen
gen: ## go generate
	$(call print-target)
	go generate ./...

.PHONY: build
build: ## goreleaser build
build:
	$(call print-target)
	GPG_FINGERPRINT=${GPG_FINGERPRINT} goreleaser build --rm-dist --single-target --snapshot

.PHONY: spell
spell: ## misspell
	$(call print-target)
	misspell -error -locale=US -w **.md

.PHONY: lint
lint: ## golangci-lint
	$(call print-target)
	golangci-lint run --fix

.PHONY: test
test: ## go test
	$(call print-target)
	go test -race -covermode=atomic -coverprofile=dist/coverage.out -coverpkg=./... ./...
	go tool cover -html=dist/coverage.out -o dist/coverage.html

.PHONY: diff
diff: ## git diff
	$(call print-target)
	git diff --exit-code
	RES=$$(git status --porcelain) ; if [ -n "$$RES" ]; then echo $$RES && exit 1 ; fi

.PHONY: manpage
manpage:
	mkdir -p contrib/manpage
	go run manpage/main.go

.PHONY: completions
completions:
	mkdir -p contrib/completion/bash \
		contrib/completion/powershell \
		contrib/completion/zsh
	go run completion/main.go bash ; mv bash_completion contrib/completion/bash/apono
	go run completion/main.go powershell ; mv powershell_completion contrib/completion/powershell/apono
	go run completion/main.go zsh ; mv zsh_completion contrib/completion/zsh/_apono

define print-target
    @printf "Executing target: \033[36m$@\033[0m\n"
endef
