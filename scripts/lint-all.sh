set -e
set -o pipefail

GO_BIN_DIR="${GOBIN:-$(go env GOPATH)/bin}"
export PATH="$GO_BIN_DIR:$PATH"

# check source code by linter
gofmt -l -w -s ./cmd
go vet ./cmd/...
which golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$GO_BIN_DIR" v2.4.0
GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-/tmp/golangci-lint}" golangci-lint run ./cmd/mcp-victorialogs
