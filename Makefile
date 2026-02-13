.PHONY: build-app build-lambda-app dev generate-keys run-app test

.EXPORT_ALL_VARIABLES:
GIT_SHA = $(shell git rev-parse --short HEAD)

build-app:
	@echo "Building app..."
	@docker build --build-arg component=app --build-arg GIT_SHA=$(GIT_SHA) -t app .

build-lambda-app:
	@echo "Building lambda app..."
	@docker build --build-arg component=lambda_app --build-arg GIT_SHA=$(GIT_SHA) --provenance=false --sbom=false -t lambda-app .

dev:
	@echo "Starting development server..."
	@go run cmd/app/main.go

generate-keys:
	@echo "Generating keys..."
	@openssl genrsa -out ./keys/private.pem 2048
	@openssl rsa -in ./keys/private.pem -pubout > ./keys/public.pem

run:
	@echo "Starting server..."
	@docker run -p 3000:3000 app

test:
	@echo "Running tests..."
	@go test -cover ./...
