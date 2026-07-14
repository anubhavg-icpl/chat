################################################################################
# Build & release helpers
################################################################################

DOCKER_IMAGE_TAG_GO_RELEASER := goreleaser/goreleaser:v2.15.4
# Docker builds cannot load a macOS USB PKCS#11 token. Options: (1) SKIP_CODE_SIGN=1 and skip;
# (2) SIGN_HTTP_URL=http://host.docker.internal:8765 plus `make sign-server` on the host to
# sign via HTTP; (3) run release-sign with host goreleaser (no Docker).
SKIP_CODE_SIGN ?= 1
# When set (e.g. http://host.docker.internal:8765), GoReleaser in Docker calls the host
# sign_server to run PKCS#11 signing on the same bind-mounted dist/ tree.
SIGN_HTTP_URL ?=
SIGN_SERVER_TOKEN ?=
GORELEASER ?= goreleaser
GOLANGCI_LINT ?= golangci-lint

DOCKER_RUN_GO_RELEASER := @docker run \
	--env CGO_ENABLED=0 \
	--env GITHUB_TOKEN=$(GITHUB_TOKEN) \
	--env SKIP_CODE_SIGN=$(SKIP_CODE_SIGN) \
	--env SIGN_HTTP_URL=$(SIGN_HTTP_URL) \
	--env SIGN_SERVER_TOKEN=$(SIGN_SERVER_TOKEN) \
	--rm \
	--volume `pwd`:/go/src/open-oscar-server \
	--workdir /go/src/open-oscar-server \
	$(DOCKER_IMAGE_TAG_GO_RELEASER)
OSCAR_HOST ?= ras.dev

.PHONY: config-basic config-ssl config
config-basic: ## Generate basic config file template
	go run ./cmd/config_generator unix config/settings.env basic

config-ssl: ## Generate SSL config file template
	go run ./cmd/config_generator unix config/ssl/settings.env ssl

config: config-basic config-ssl ## Generate all config file templates from Config struct

.PHONY: lint
lint: ## Run formatting and static analysis checks
	@fmt_output="$$(gofmt -s -l .)"; \
	if [ -n "$$fmt_output" ]; then \
		echo "The following files need formatting:"; \
		echo "$$fmt_output"; \
		exit 1; \
	fi
	$(GOLANGCI_LINT) run ./...
	go vet ./...

.PHONY: release
release: ## Run a clean, full GoReleaser run (publish + validate)
	$(DOCKER_RUN_GO_RELEASER) --clean

.PHONY: release-dry-run
release-dry-run: ## GoReleaser dry-run (skips validate & publish)
	$(DOCKER_RUN_GO_RELEASER) --clean --skip=validate --skip=publish

SIGN_SERVER_PORT ?= 8765

.PHONY: sign-server
sign-server: ## Local HTTP signer for Windows PE (run before Docker release if using SIGN_HTTP_URL)
	go run ./cmd/sign_server

.PHONY: sign-server-stop
sign-server-stop: ## Stop whatever is listening on SIGN_SERVER_PORT (usually a leftover sign_server)
	-@kill $$(lsof -t -iTCP:$(SIGN_SERVER_PORT) -sTCP:LISTEN) 2>/dev/null || true

# Default URL for GoReleaser-in-Docker → host signing (Docker Desktop Mac/Win).
# On Linux Docker, use host.docker.internal:8765 only if you add
# --add-host=host.docker.internal:host-gateway to the docker run (or set SIGN_DOCKER_URL).
SIGN_DOCKER_URL ?= http://host.docker.internal:8765

.PHONY: release-dry-run-sign-docker
release-dry-run-sign-docker: ## Dry-run in Docker; Windows Authenticode via host sign_server (run `make sign-server` first)
	@$(MAKE) release-dry-run SIGN_HTTP_URL=$(SIGN_DOCKER_URL)

.PHONY: release-sign-docker
release-sign-docker: ## Full release in Docker; Windows Authenticode via host sign_server (run `make sign-server` first)
	@$(MAKE) release SIGN_HTTP_URL=$(SIGN_DOCKER_URL)

.PHONY: release-dry-run-nosign
release-dry-run-nosign: ## GoReleaser dry-run on host without Windows Authenticode
	SKIP_CODE_SIGN=1 $(GORELEASER) --clean --skip=validate --skip=publish

.PHONY: release-nosign
release-nosign: ## Full GoReleaser on host without Windows Authenticode
	SKIP_CODE_SIGN=1 $(GORELEASER) --clean

.PHONY: release-dry-run-sign
release-dry-run-sign: ## GoReleaser dry-run on host with Windows signing (needs $(GORELEASER), PKCS#11 env)
	SKIP_CODE_SIGN=0 $(GORELEASER) --clean --skip=validate --skip=publish

.PHONY: release-sign
release-sign: ## Full GoReleaser on host with Windows signing (needs $(GORELEASER), PKCS#11 env)
	SKIP_CODE_SIGN=0 $(GORELEASER) --clean

.PHONY: docker-image-ras
docker-image-ras: ## Build Open OSCAR Server image from local Dockerfile
	docker compose build open-oscar-server

.PHONY: docker-image-stunnel
docker-image-stunnel: ## Build stunnel image (OpenSSL 1.0.2u) from local Dockerfile
	docker compose build stunnel

.PHONY: docker-image-certgen
docker-image-certgen: ## Build certgen helper image from local Dockerfile
	docker compose build cert-gen

.PHONY: docker-images
docker-images: ## Build all images from local Dockerfiles (no registry pulls)
	docker compose build

.PHONY: docker-image-console
docker-image-console: ## Build operator console image
	docker compose build web

.PHONY: docker-run
docker-run: ## Build from repo and run Open OSCAR Server + stunnel (foreground)
	OSCAR_HOST=$(OSCAR_HOST) docker compose up --build open-oscar-server stunnel cert-gen

.PHONY: docker-run-bg
docker-run-bg: ## Build from repo and run full stack in background (incl. console)
	OSCAR_HOST=$(OSCAR_HOST) docker compose up -d --build

.PHONY: docker-run-stop
docker-run-stop: ## Stop Open OSCAR Server docker-compose services
	OSCAR_HOST=$(OSCAR_HOST) docker compose down

.PHONY: run
run: # run the server with plain socket config
	./scripts/run_dev.sh ./config/settings.env

.PHONY: run-ssl
run-ssl: # run the server with ssl socket config
	./scripts/run_dev.sh ./config/ssl/settings.env

.PHONY: run-stunnel
run-stunnel: # run stunnel for SSL termination
	./scripts/run_stunnel.sh ./certs/server.pem

################################################################################
# SSL Helpers
################################################################################

.PHONY: docker-cert
docker-cert: clean-certs ## Create SSL certificates for server
	mkdir -p certs/
	OSCAR_HOST=$(OSCAR_HOST) docker compose run --no-TTY --rm --build cert-gen

.PHONY: docker-nss
docker-nss: ## Create NSS certificate database for AIM 6.x clients
	OSCAR_HOST=$(OSCAR_HOST) docker compose --profile nss run --no-TTY --rm --build nss-gen

.PHONY: clean-certs
clean-certs: ## Remove all generated certificates & NSS DB
	rm -rf certs/*

################################################################################
# Web API Tools
################################################################################

.PHONY: webapi-keygen
webapi-keygen: ## Build the Web API key generator tool
	go build -o webapi_keygen ./cmd/webapi_keygen

.PHONY: webapi-keygen-install
webapi-keygen-install: ## Install the Web API key generator tool system-wide
	go install ./cmd/webapi_keygen

################################################################################
# Client Tools
################################################################################

.PHONY: tocbot
tocbot: ## Build the TOC protocol bot binary (connects to the TOC server on :9898)
	go build -o tocbot ./cmd/tocbot

.PHONY: admin
admin: ## Build the management API admin CLI
	go build -o oscar-admin ./cmd/admin

.PHONY: clients
clients: tocbot admin ## Build all client tools (TOC bot + admin CLI)

.PHONY: aibot
aibot: ## Build the LLM AIM bot (OpenAI-compatible; needs OPENAI_API_KEY to run)
	go build -o aibot ./cmd/aibot

.PHONY: discord-bridge
discord-bridge: ## Build the Discord<->AIM bridge (needs DISCORD_TOKEN to run)
	go build -o discord-bridge ./cmd/discord-bridge

.PHONY: irc-bridge
irc-bridge: ## Build the IRC<->AIM bridge (stdlib IRC)
	go build -o irc-bridge ./cmd/irc-bridge

.PHONY: bridges
bridges: discord-bridge irc-bridge ## Build all bridges
