.PHONY: install build test start clean docker-build

install:
	npm ci || npm install

build: install
	npm run build

test:
	npm test

start:
	npm run dev

docker-build:
	docker build -t plannedwallet-frontend:local .

clean:
	rm -rf .next out node_modules
