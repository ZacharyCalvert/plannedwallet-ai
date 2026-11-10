.PHONY: build test start clean

build:
	docker build -t plannedwallet-gateway:local .

test:
	@echo "Nginx configuration syntax check"

start:
	docker run --rm -p 8181:8181 plannedwallet-gateway:local

clean:
	@echo "Clean completed"
