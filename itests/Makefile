.PHONY: test install clean help

TARGET_SERVER ?= http://localhost:8080
VENV ?= .venv
PYTHON := $(VENV)/bin/python
PIP := $(VENV)/bin/pip
PYTEST := $(VENV)/bin/pytest

help:
	@echo "Integration Test Runner"
	@echo "Usage:"
	@echo "  make install                      - Create virtualenv and install dependencies"
	@echo "  make test                         - Run tests (default TARGET_SERVER=http://localhost:8080)"
	@echo "  make test TARGET_SERVER=<url>     - Run tests against specific server"
	@echo "  make clean                        - Remove virtualenv and cache files"

$(PYTEST):
	python3 -m venv $(VENV)
	$(PIP) install --upgrade pip
	$(PIP) install -r requirements.txt

install: $(PYTEST)

test: install
	TARGET_SERVER=$(TARGET_SERVER) $(PYTEST) -v

clean:
	rm -rf $(VENV) __pycache__ .pytest_cache
