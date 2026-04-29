.PHONY: help dev test build clean deploy teardown install-tools

## Commands help
help:
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

## Required development tools installation
install-tools:
	@echo "Installing Go tools..."
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "Tools installed successfully"

## Start local env
dev:
	docker-compose up

## Start server locally
dev-api:
	@which templ > /dev/null || (echo "templ not found. Run: make install-tools" && exit 1)
	@which swag > /dev/null || (echo "swag not found. Run: make install-tools" && exit 1)
	templ generate && swag init -g cmd/server/main.go -o docs && go run ./cmd/server

## Generate templ templates
templ:
	@which templ > /dev/null || (echo "templ not found. Run: make install-tools" && exit 1)
	templ generate

## Run all tests
test:
	@which templ > /dev/null || (echo "templ not found. Run: make install-tools" && exit 1)
	@which swag > /dev/null || (echo "swag not found. Run: make install-tools" && exit 1)
	@echo "Generating code..."
	@templ generate > /dev/null 2>&1
	@swag init -g cmd/server/main.go -o docs > /dev/null 2>&1
	@echo "Running Go tests..."
	go test -v -race -cover ./...
	@echo "Running linters..."
	go vet ./...
	@which staticcheck > /dev/null && staticcheck ./... || echo "staticcheck not installed, skipping (run: make install-tools)"

## Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## Build binary
build:
	@which templ > /dev/null || (echo "templ not found. Run: make install-tools" && exit 1)
	@which swag > /dev/null || (echo "swag not found. Run: make install-tools" && exit 1)
	@echo "Generating templates..."
	templ generate
	@echo "Generating API docs..."
	swag init -g cmd/server/main.go -o docs
	@echo "Building binary..."
	go build -o bin/server ./cmd/server
	@echo "Build complete"

## Build Docker image locally
build-docker:
	docker build --target dev -t gymshark-challenge .

## Generate docs
swagger:
	@which swag > /dev/null || (echo "swag not found. Run: make install-tools" && exit 1)
	swag init -g cmd/server/main.go -o docs

## Clean artifacts
clean:
	rm -rf bin
	rm -rf docs/swagger.*
	rm -rf internal/web/templates/*_templ.go
	rm -rf coverage.out

## Bootstrap Terraform (run this only once to create the state on AWS)
bootstrap:
	@echo "Bootstrapping Terraform state backend..."
	cd terraform/bootstrap && terraform init && terraform apply
	@echo "Bootstrap complete. Update backend config in prod/main.tf if needed"

tf-init:
	@echo "Note: Set TF_STATE_BUCKET and TF_STATE_LOCK_TABLE environment variables"
	cd terraform/environments/prod && terraform init \
		-backend-config="bucket=$${TF_STATE_BUCKET}" \
		-backend-config="key=prod/terraform.tfstate" \
		-backend-config="region=us-east-1" \
		-backend-config="dynamodb_table=$${TF_STATE_LOCK_TABLE}" \
		-backend-config="encrypt=true"

tf-plan:
	cd terraform/environments/prod && terraform plan

tf-apply:
	cd terraform/environments/prod && terraform apply

tf-destroy:
	cd terraform/environments/prod && terraform destroy

## Full deployment
deploy:
	@echo "This is handled by GitHub Actions on push to main"
	@echo "To deploy manually:"
	@echo "  1. make tf-apply"
	@echo "  2. Build and push Docker image to ECR"
	@echo "  3. Lambda updates automatically from ECR"

## Complete teardown (destroys everything about this on AWS)
destroy:
	@echo "WARNING: This will destroy all infrastructure"
	@read -p "Type 'yes' to continue: " confirm && [ "$$confirm" = "yes" ] || exit 1
	@test -n "$${TF_STATE_BUCKET}" || (echo "Error: TF_STATE_BUCKET not set" && exit 1)
	@echo "Destroying application infrastructure..."
	cd terraform/environments/prod && terraform destroy -auto-approve
	@echo "Emptying state bucket..."
	aws s3 rm s3://$${TF_STATE_BUCKET} --recursive
	@echo "Destroying bootstrap infrastructure..."
	cd terraform/bootstrap && terraform destroy -auto-approve
	@echo "Destroy complete"

fmt:
	go fmt ./...
	cd terraform && terraform fmt -recursive

lint:
	go vet ./...
	@which staticcheck > /dev/null && staticcheck ./... || echo "staticcheck not installed"
	cd terraform && terraform fmt -check -recursive
