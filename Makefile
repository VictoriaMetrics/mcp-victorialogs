download-docs:
	bash ./scripts/update-docs.sh

download-blog:
	bash ./scripts/update-blog.sh

update-docs: download-docs

update-blog: download-blog

update-resources: update-docs update-blog

test:
	bash ./scripts/test-all.sh

check:
	bash ./scripts/check-all.sh

lint:
	bash ./scripts/lint-all.sh

build: ui-build
	bash ./scripts/build-binaries.sh

ui-install:
	cd web && npm install

ui-build: ui-install
	cd web && npm run build && touch ./dist/.gitkeep

ui-dev:
	cd web && npm run dev

dev-playground: build
	VL_INSTANCE_ENTRYPOINT=https://play-vmlogs.victoriametrics.com \
	MCP_SERVER_MODE=http \
	MCP_LISTEN_ADDR=:8081 \
	./mcp-victorialogs

all: test check lint build
