
.PHONY: all
.PHONY: build
.PHONY: check
.PHONY: lint
.PHONY: lint-go
.PHONY: test
.PHONY: test-cleanup
.PHONY: Test%
.PHONY: image
.PHONY: push

FINDFILES=find . \( -path ./.git -o -path ./.github -o -path ./tmp \) -prune -o -type f

CONTAINER_IMAGE ?= quay.io/maistra/maistra-test-tool:latest

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

test-cleanup:
	kubectl delete ns istio-system bookinfo foo bar mesh-external legacy --ignore-not-found

image:
	podman build -t ${CONTAINER_IMAGE} .

push: image
	podman push ${CONTAINER_IMAGE}

clean:
	rm -rf tests/result-*

test-groups:
# Display all the test groups available in the test suite
	@echo "Available test groups:"
	@echo ""
	@awk '/^	[a-zA-Z0-9_]+[ \t]+TestGroup =/ {gsub(/^[ \t]+/, "", $$1); gsub("\"", "", $$3); print $$1}' pkg/util/test/test.go
	@echo ""
	@echo "Test group count:" `awk '/^	[a-zA-Z0-9_]+[ \t]+TestGroup =/ {gsub(/^[ \t]+/, "", $$1); gsub("\"", "", $$3); print $$1}' pkg/util/test/test.go | wc -l`
	@echo "To run all tests in a group, use 'TEST_GROUP=<group-name> make test'"

test-groups-%:
# Display all the tests in the specified test group
	@echo "Available tests in group '$*':"
	@echo ""
	@find pkg/tests -name "*_test.go" -exec grep -El 'Groups\(.*$*' {} \;
	@echo ""
	@echo "Test package count in Test Group '$*':" `find pkg/tests -name "*_test.go" -exec grep -El 'Groups\(.*$*' {} \; | wc -l`
	@echo "To run all tests in group '$*', use 'TEST_GROUP='$*' make test'"

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Available targets:"
	@echo "  all               - build and run all tests"
	@echo "  build             - build the test binary"
	@echo "  check             - run all pre-commit checks"
	@echo "  lint              - run all linters"
	@echo "  lint-go           - run the Go linter"
	@echo "  test              - run all tests"
	@echo "  test-cleanup      - delete all test resources"
	@echo "  Test<test-name>   - run the specified test"
	@echo "  image             - build the container image"
	@echo "  push              - push the container image to the registry"
	@echo "  clean             - remove all generated files"
	@echo "  test-groups       - list all test groups"
	@echo "  test-groups-<group-name>"
	@echo "                    - list all tests in the specified group"
	@echo "  help              - print this help message"
