.PHONY: build-app build-lambda-app dev generate-keys run-app test

build-app:
	@echo "Building app..."
	@docker build --build-arg component=app -t app .

build-lambda-app:
	@echo "Building lambda app..."
	@docker build --build-arg component=lambda_app -t lambda-app .

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
