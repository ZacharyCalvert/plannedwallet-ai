.PHONY: start stop clean build build-api build-frontend build-gateway test refresh local

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
