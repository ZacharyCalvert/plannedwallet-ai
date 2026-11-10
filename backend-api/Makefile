.PHONY: spec stop-spec stop

spec: stop-spec
	docker run --rm -d --name pw_spec -p 8080:8080 -v "$(shell pwd)":/tmp -e SWAGGER_JSON=/tmp/schema.yml docker.swagger.io/swaggerapi/swagger-ui

stop-spec:
	@docker stop pw_spec >/dev/null 2>&1 || true

stop: stop-spec
