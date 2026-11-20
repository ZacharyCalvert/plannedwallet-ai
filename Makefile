.PHONY: start stop clean build build-api build-frontend build-gateway test itest refresh local pdfs

export DATA_DIR ?= data

refresh: stop clean start

start: stop
	mkdir -p $(DATA_DIR)
	docker compose up -d --build

local: stop
	mkdir -p local-data
	DATA_DIR=local-data docker compose up -d --build

clean: stop
	rm -rf ./$(DATA_DIR) || sleep 2 && docker compose down --remove-orphans && rm -rf ./$(DATA_DIR) && echo "Cleaned directory"

stop:
	docker compose down --remove-orphans

build: build-api build-frontend build-gateway

build-api:
	docker build -t plannedwallet:local ./backend-api

build-frontend:
	docker build -t plannedwallet-frontend:local ./frontend-ui

build-gateway:
	docker build -t plannedwallet-gateway:local ./nginx-gateway

test:
	make -C backend-api test
	make -C frontend-ui test

itest: refresh
	@status=0; \
	ready=0; \
	for i in $$(seq 1 60); do \
		if curl -sf -H "X-Client-ID: make-itest" http://localhost:8181/api/health >/dev/null 2>&1 \
			&& curl -sf http://localhost:8181 | grep -q "Planned Wallet"; then \
			echo "Local gateway ready at http://localhost:8181"; \
			ready=1; \
			break; \
		fi; \
		echo "Waiting for local gateway ($$i/60)..."; \
		sleep 2; \
	done; \
	if [ "$$ready" -ne 1 ]; then \
		echo "Error: Local gateway failed to become ready within timeout"; \
		docker compose ps; \
		docker compose logs; \
		make clean; \
		exit 1; \
	fi; \
	make -C itests test TARGET_SERVER=http://localhost:8181 || status=$$?; \
	if [ "$$status" -ne 0 ]; then \
		docker compose logs || true; \
	fi; \
	make clean || true; \
	exit $$status

pdfs:
	@if [ ! -x itests/.venv/bin/python ]; then $(MAKE) -C itests install; fi
	itests/.venv/bin/python scripts/pdf-generation/generate_pdfs.py
	@echo "Updated PW_TERMS SHA-256:"
	@shasum -a 256 frontend-ui/public/terms-of-use.pdf
