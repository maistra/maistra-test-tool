
.PHONY: all
.PHONY: build
.PHONY: check
.PHONY: lint
.PHONY: lint-go
.PHONY: test
.PHONY: Test%
.PHONY: test.presubmit
.PHONY: test.presubmit.cleanup
.PHONY: test.presubmit.install

FINDFILES=find . \( -path ./.git -o -path ./.github -o -path ./tmp \) -prune -o -type f

all: test

build:
	scripts/compiletests.sh

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

# this will install production operators
test.presubmit.install:
	scripts/ci/operator.sh create wait get_csv

# this will delete production operators
test.presubmit.cleanup:
	scripts/ci/operator.sh delete delete_csv delete_cni

# In an CI job, this will run tests on a remote OpenShift cluster
test.presubmit:
	$(MAKE) test.presubmit.install
	$(MAKE) test
	$(MAKE) test.presubmit.cleanup
