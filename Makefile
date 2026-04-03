.PHONY: help lint test test-integration test-e2e mutation security budgets ci pre-commit analyzers clean

help:
	@echo "Available targets:"
	@echo "  lint              Run golangci-lint + custom analyzers"
	@echo "  test              Run unit tests with coverage gate"
	@echo "  test-integration  Run integration tests"
	@echo "  test-e2e          Run E2E tests (requires CI_E2E=true)"
	@echo "  mutation          Run mutation tests"
	@echo "  security          Run gosec + govulncheck"
	@echo "  budgets           Run file size + file count checks"
	@echo "  ci                Run full CI pipeline (all stages)"
	@echo "  pre-commit        Run pre-commit checks"
	@echo "  analyzers         Build custom Go analyzers"
	@echo "  clean             Remove generated artifacts"

lint:
	@./scripts/ci-lint.sh

test:
	@./scripts/ci-test.sh

test-integration:
	@./scripts/ci-integration.sh

test-e2e:
	@CI_E2E=true ./scripts/ci-e2e.sh

mutation:
	@./scripts/ci-mutation.sh

security:
	@./scripts/ci-security.sh

budgets:
	@./scripts/ci-budgets.sh

ci:
	@./scripts/ci.sh

pre-commit:
	@./scripts/pre-commit.sh

analyzers:
	@cd tools/analyzers && go build -o bin/analyzers .

clean:
	@rm -f coverage.out
	@rm -f tools/analyzers/bin/analyzers
	@echo "Cleaned"
