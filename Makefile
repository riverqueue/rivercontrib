.DEFAULT_GOAL := help

# Looks at comments using ## on targets and uses them to produce a help output.
.PHONY: help
help: ALIGN=22
help: ## Print this message
	@awk -F '::? .*## ' -- "/^[^':]+::? .*## /"' { printf "'$$(tput bold)'%-$(ALIGN)s'$$(tput sgr0)' %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Each directory of a submodule in the Go workspace. Go commands provide no
# built-in way to run for all workspace submodules. Add a new submodule to the
# workspace with `go work use ./driver/new`.
submodules := $(shell go list -f '{{.Dir}}' -m)

# Definitions of following tasks look ugly, but they're done this way because to
# produce the best/most comprehensible output by far (e.g. compared to a shell
# loop).
.PHONY: lint
lint:: ## Run linter (golangci-lint) for all submodules
define lint-target
    lint:: ; cd $1 && golangci-lint run --fix --timeout 5m
endef
$(foreach mod,$(submodules),$(eval $(call lint-target,$(mod))))

.PHONY: test
test:: ## Run test suite for all submodules
define test-target
    test:: ; cd $1 && go test ./...
endef
$(foreach mod,$(submodules),$(eval $(call test-target,$(mod))))

.PHONY: test/race
test/race:: ## Run test suite for all submodules with race detector
define test-race-target
    test/race:: ; cd $1 && go test ./... -race
endef
$(foreach mod,$(submodules),$(eval $(call test-race-target,$(mod))))

.PHONY: tidy
tidy:: ## Run `go mod tidy` for all submodules
define tidy-target
    tidy:: ; cd $1 && go mod tidy
endef
$(foreach mod,$(submodules),$(eval $(call tidy-target,$(mod))))

.PHONY: update-mod-go
update-mod-go: ## Update `go`/`toolchain` directives in all submodules to match `go.work`
	go run github.com/riverqueue/river/rivershared/cmd/update-mod-go@latest ./go.work

.PHONY: update-mod-version
update-mod-version: ## Update River packages in all submodules to $VERSION
	PACKAGE_PREFIX="github.com/riverqueue/rivercontrib" go run github.com/riverqueue/river/rivershared/cmd/update-mod-version@latest ./go.work
