
.PHONY: all
.PHONY: build
.PHONY: check
.PHONY: lint
.PHONY: lint-go
.PHONY: test
.PHONY: Test%

FINDFILES=find . \( -path ./.git -o -path ./.github -o -path ./tmp \) -prune -o -type f

all: test

build:
	go build ./...

# perform all the pre-commit checks
check: build lint

lint: lint-go

lint-go:
	@${FINDFILES} -name '*.go' -print0 | ${XARGS} scripts/lint_go.sh

# You can use this target in two ways:
#     make test                       # runs all tests
#     make test TestFaultInjection    # runs the specified test
test:
	scripts/runtests.sh $(filter-out $@,$(MAKECMDGOALS))

# this prevents errors like "No rule to make target 'TestFaultInjection'" when you run "make test TestFaultInjection"
Test%:
	@:
