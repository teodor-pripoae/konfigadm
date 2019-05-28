all: test docs integration


.PHONY: linux
linux:
	GOOS=linux go build -o dist/konfigadm -ldflags '-X main.version=built-$(shell date +%Y%m%d%M%H%M%S)' .

.PHONY: test
test:
	go test -v ./pkg/... ./cmd/... -race -coverprofile=coverage.txt -covermode=atomic

.PHONY: integration
integration: linux
		./scripts/e2e.sh $(test)

.PHONY: docs
docs:
	git remote add docs "https://$(GH_TOKEN)@github.com/moshloop/konfigadm.git"
	git fetch docs && git fetch docs gh-pages:gh-pages
	mkdocs gh-deploy -v --remote-name docs -m "Deployed {sha} with MkDocs version: {version} [ci skip]"
