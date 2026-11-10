.PHONY: start stop build-api

DATA_DIR := data

start:
	mkdir -p $(DATA_DIR)
	docker compose up -d

stop:
	docker compose down

build-api:
	docker build -t plannedwallet:local ./backend-api
