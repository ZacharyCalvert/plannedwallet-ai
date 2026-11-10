.PHONY: start stop

DATA_DIR := data

start:
	mkdir -p $(DATA_DIR)
	docker compose up -d

stop:
	docker compose down
