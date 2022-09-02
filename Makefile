.ONESHELL:
.SHELL := /usr/bin/bash
.PHONY: help docker-start-components test _set-env


# we will put our integration testing in this path
INTEGRATION_TEST_PATH?=./pkg/service

# set of env variables that you need for testing
ENV_LOCAL_TEST=\
  PG_PASSWORD=test \
  PG_DB=eth \
  PG_HOST=pgsql.local \
  PG_USER=test

BOLD=$(shell tput bold)
RED=$(shell tput setaf 1)
GREEN=$(shell tput setaf 2)
YELLOW=$(shell tput setaf 3)
RESET=$(shell tput sgr0)

target: help
	$(info ${HELP_MESSAGE})
	@exit 0

help:
	@echo "$(YELLOW)$(BOLD)[INFO] ...::: WELCOME :::...$(RESET)"
	head -n 5 ./LICENCE.md && tail -n 2 ./LICENCE.md && printf "\n\n"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

docker-start-components: ## Start docker compose
	docker compose up -d --remove-orphans

docker-start-postgres: ## Start docker compose postgres
	docker compose up -d --remove-orphans pgsql.local

docker-stop: ## Stop docker compose postgres
	docker compose down

docker-force-rebuild: ## rebuild without cache
	docker compose build --no-cache webapi.local && docker compose up --force-recreate

test-integration: ## Run go test with postgres
	$(ENV_LOCAL_TEST) \
	go test -tags=postgres $(INTEGRATION_TEST_PATH) -count=1 -run=$(INTEGRATION_TEST_SUITE_PATH)

test-integration-debug: ## Run go test verbose with postgres
	$(ENV_LOCAL_TEST) \
	go test -tags=postgres $(INTEGRATION_TEST_PATH) -count=1 -v -run=$(INTEGRATION_TEST_SUITE_PATH)

#############
#  Helpers  #
#############

_set-env: ## Confirm that all required Environment Variables have been set.
	@if [ -z $(PG_PASSWORD) ]; then \
		echo "$(BOLD)$(RED)PG_PASSWORD was not set$(RESET)"; \
		ERROR=1; \
	 fi
	@if [ ! -z $${ERROR} ] && [ $${ERROR} -eq 1 ]; then \
		echo "$(BOLD)Example usage: \`PG_PASSWORD=my_profile  make docker.start.components\`$(RESET)"; \
		exit 1; \
	 fi


define HELP_MESSAGE

	Environment variables to be aware of or to hardcode depending on your use case:

        PG_PASSWORD
		Default: not_defined
		Info: Environment variable to declare the default postgreSQL password

	$(GREEN)Common usage:$(RESET)

	$(BOLD)...::: Start the docker compose components for development :::...$(RESET)
	$(GREEN)~$$$(RESET) make docker-start-components

	$(BOLD)...::: Start the docker compose postgres for testing :::...$(RESET)
	$(GREEN)~$$$(RESET) make docker-start-postgres

endef
