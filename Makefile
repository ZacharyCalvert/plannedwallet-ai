.PHONY: start stop build build-api build-frontend build-gateway test

DATA_DIR := data

start:
	mkdir -p $(DATA_DIR)
	docker compose up -d

clean:
	rm -rf ./$(DATA_DIR)

stop:
	docker compose down

build: build-api build-frontend build-gateway

build-api:
	docker build -t plannedwallet:local ./backend-api

build-frontend:
	docker build -t plannedwallet-frontend:local ./frontend-ui

build-gateway:
	docker build -t plannedwallet-gateway:local ./nginx

test:
	make -C backend-api test
	make -C frontend-ui test
