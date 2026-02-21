.PHONY: lint fix
DEVBINDIR = $(CURDIR)/devbin

LINT_VERSION ?= v2.9.0
LINT_BIN := $(DEVBINDIR)/golangci-lint

lint: $(LINT_BIN)
	$(LINT_BIN) run ./...

fix: $(LINT_BIN)
	$(LINT_BIN) run --fix ./...

$(LINT_BIN):
	@echo "Installing golangci-lint $(LINT_VERSION)..."
	@mkdir -p $(DEVBINDIR)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
		sh -s -- -b $(DEVBINDIR) $(LINT_VERSION)

.PHONY: vuln
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
