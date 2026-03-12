set -e
set -o pipefail

GO_BIN_DIR="${GOBIN:-$(go env GOPATH)/bin}"
export PATH="$GO_BIN_DIR:$PATH"

# check licenses
which wwhrd || go install github.com/frapposelli/wwhrd@latest
wwhrd check -f .wwhrd.yml

# check for vulnerabilities
which govulncheck || go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
