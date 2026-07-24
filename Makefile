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

# These commands omit `./cmd` because updating River might cause
# self-referencing problems to other River Pro modules from `./cmd`. We should
# switch to a Go workspace to avoid all this nastiness.
.PHONY: update-river
update-river:: ## Update version of River dependencies to latest
define update-river-target
update-river:: ; cd $1 && \
	go get -u github.com/riverqueue/river@latest \
		github.com/riverqueue/river/riverdriver@latest \
		github.com/riverqueue/river/riverdriver/riverdatabasesql@latest \
		github.com/riverqueue/river/riverdriver/riverpgxv5@latest \
		github.com/riverqueue/river/riverdriver/riversqlite@latest \
		github.com/riverqueue/river/rivershared@latest \
		github.com/riverqueue/river/rivertype@latest && \
	go mod tidy
endef
$(foreach mod,$(submodules),$(eval $(call update-river-target,$(mod))))

# Use this like:
#
#     RIVER_BRANCH=brandur-pilot-init make update-river-to-branch
#
# Which updates all River dependencies in all submodules to the given branch.
# Omit `RIVER_BRANCH` to update to `master`.
.PHONY: update-river-to-branch
update-river-to-branch:: ## Update version of River dependencies to a specific branch

# Set the branch to the value of RIVER_MASTER or default to 'master'
RIVER_BRANCH ?= $(or $(RIVER_MASTER),master)

define update-river-to-branch-target
update-river-to-branch:: ; cd $1 && \
	go get -u github.com/riverqueue/river@$(RIVER_BRANCH) \
		github.com/riverqueue/river/riverdriver@$(RIVER_BRANCH) \
		github.com/riverqueue/river/riverdriver/riverdatabasesql@$(RIVER_BRANCH) \
		github.com/riverqueue/river/riverdriver/riverpgxv5@$(RIVER_BRANCH) \
		github.com/riverqueue/river/riverdriver/riversqlite@$(RIVER_BRANCH) \
		github.com/riverqueue/river/rivershared@$(RIVER_BRANCH) \
		github.com/riverqueue/river/rivertype@$(RIVER_BRANCH) && \
	go mod tidy
endef
$(foreach mod,$(submodules),$(eval $(call update-river-to-branch-target,$(mod))))
