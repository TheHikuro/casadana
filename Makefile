.DEFAULT_GOAL := help

COMPOSE_FILE := .docker/docker-compose.yml
ENV_FILE     := .env.dev

# --env-file is required so compose can interpolate ${POSTGRES_USER} etc. in the
# postgres service definition (that's separate from api's own env_file: entry).
# --profile dev is required or the profile-gated services (api/postgres/web-dev)
# won't be started at all.
COMPOSE := docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) --profile dev

FILE     ?= .env.dev
ENC_FILE ?= .env.encrypted

.PHONY: help encrypt decrypt backend backend-down backend-logs front dev

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'

encrypt: ## Encrypt FILE (default .env.dev) into ENC_FILE (default .env.encrypted) via sops
	@sops -e --input-type dotenv --output-type dotenv $(FILE) > $(ENC_FILE).tmp || \
		(rm -f $(ENC_FILE).tmp; echo "ERROR: sops encrypt failed - check .sops.yaml and your age key" >&2; exit 1)
	@mv $(ENC_FILE).tmp $(ENC_FILE)
	@echo "Encrypted $(FILE) -> $(ENC_FILE)"

decrypt: ## Decrypt ENC_FILE (default .env.encrypted) into FILE (default .env.dev) via sops
	@sops -d --input-type dotenv --output-type dotenv $(ENC_FILE) > $(FILE).tmp || \
		(rm -f $(FILE).tmp; echo "ERROR: sops decrypt failed - no age key available (set SOPS_AGE_KEY or check ~/.config/sops/age/keys.txt)" >&2; exit 1)
	@mv $(FILE).tmp $(FILE)
	@echo "Decrypted $(ENC_FILE) -> $(FILE)"

backend: ## Start dockerized backend only (api + postgres), detached
	$(COMPOSE) up -d api postgres

backend-down: ## Stop the dockerized backend containers
	$(COMPOSE) down

backend-logs: ## Follow logs for the dockerized backend containers
	$(COMPOSE) logs -f api postgres

front: ## Run the frontend natively with Bun (fast HMR, not containerized)
	bun install && bun --cwd apps/web dev

dev: ## Reminder: backend and front are both blocking, run them in separate terminals
	@echo "Run these in two separate terminals:"
	@echo "  make backend   # dockerized api + postgres"
	@echo "  make front     # bun dev server for the frontend"
