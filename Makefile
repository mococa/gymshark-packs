.PHONY: help dev dev-api templ test test-coverage build build-docker swagger clean \
        install-tools bootstrap tf-init tf-plan tf-apply tf-destroy deploy destroy fmt lint

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

install-tools: ## Install templ, swag, and staticcheck
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest

dev: ## Start local env via Docker Compose
	docker-compose up

dev-api: ## Run server locally (requires install-tools)
	@which templ > /dev/null || (echo "templ not found. Run: make install-tools" && exit 1)
	@which swag > /dev/null || (echo "swag not found. Run: make install-tools" && exit 1)
	templ generate && swag init -g cmd/server/main.go -o docs && go run ./cmd/server

templ: ## Generate templ files
	@which templ > /dev/null || (echo "templ not found. Run: make install-tools" && exit 1)
	templ generate

test: ## Run tests and linters
	@which templ > /dev/null || (echo "templ not found. Run: make install-tools" && exit 1)
	@which swag > /dev/null || (echo "swag not found. Run: make install-tools" && exit 1)
	@templ generate > /dev/null 2>&1
	@swag init -g cmd/server/main.go -o docs > /dev/null 2>&1
	go test -v -race -cover ./...
	go vet ./...
	@which staticcheck > /dev/null && staticcheck ./... || echo "staticcheck not installed, skipping (run: make install-tools)"

test-coverage: ## Generate HTML coverage report
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

build: ## Build server binary to bin/server
	@which templ > /dev/null || (echo "templ not found. Run: make install-tools" && exit 1)
	@which swag > /dev/null || (echo "swag not found. Run: make install-tools" && exit 1)
	templ generate
	swag init -g cmd/server/main.go -o docs
	go build -o bin/server ./cmd/server

build-docker: ## Build Docker image locally (dev target)
	docker build --target dev -t gymshark-challenge .

swagger: ## Generate Swagger docs
	@which swag > /dev/null || (echo "swag not found. Run: make install-tools" && exit 1)
	swag init -g cmd/server/main.go -o docs

clean: ## Remove build artifacts and generated files
	rm -rf bin docs/swagger.* internal/web/templates/*_templ.go coverage.out

## Bootstrap Terraform state backend (one-time).
## Pass DOMAIN_NAME to also request an ACM certificate:
##   make bootstrap DOMAIN_NAME=gymshark-challenge.moureau.dev
bootstrap: ## Bootstrap Terraform state backend (one-time)
	cd terraform/bootstrap && terraform init && terraform apply $(if $(DOMAIN_NAME),-var="domain_name=$(DOMAIN_NAME)",)
	@echo ""
	@echo "Bootstrap complete. Set these as GitHub repository secrets:"
	@echo "  TF_STATE_BUCKET      = (see state_bucket output above)"
	@echo "  TF_STATE_LOCK_TABLE  = (see lock_table output above)"
	$(if $(DOMAIN_NAME),@echo "  ACM_CERTIFICATE_ARN  = (see certificate_arn output above)"
	@echo ""
	@echo "ACTION REQUIRED: Add the certificate_validation_cname DNS record to your DNS"
	@echo "provider, then wait for the certificate status to become ISSUED before deploying.",)

tf-init: ## Init Terraform backend (requires TF_STATE_BUCKET and TF_STATE_LOCK_TABLE)
	cd terraform/environments/prod && terraform init \
		-backend-config="bucket=$${TF_STATE_BUCKET}" \
		-backend-config="key=prod/terraform.tfstate" \
		-backend-config="region=us-east-1" \
		-backend-config="dynamodb_table=$${TF_STATE_LOCK_TABLE}" \
		-backend-config="encrypt=true"

tf-plan: ## Terraform plan
	cd terraform/environments/prod && terraform plan

tf-apply: ## Terraform apply
	cd terraform/environments/prod && terraform apply

tf-destroy: ## Terraform destroy prod infrastructure
	cd terraform/environments/prod && terraform destroy

deploy: ## Deploy (handled by GitHub Actions on push to main)
	@echo "Push to main to trigger GitHub Actions deployment"

## Complete teardown of all AWS infrastructure.
## Usage: make destroy [TF_STATE_BUCKET=x] [TF_STATE_LOCK_TABLE=y] [DOMAIN_NAME=z] [ACM_CERTIFICATE_ARN=w]
destroy: ## Destroy all AWS infrastructure (prompts for confirmation)
	@bash -c '\
		set -e; \
		[ -n "$$TF_STATE_BUCKET" ] || { read -p "TF_STATE_BUCKET: " TF_STATE_BUCKET; }; \
		[ -n "$$TF_STATE_LOCK_TABLE" ] || { read -p "TF_STATE_LOCK_TABLE: " TF_STATE_LOCK_TABLE; }; \
		echo "WARNING: This will destroy all infrastructure"; \
		read -p "Type yes to continue: " confirm && [ "$$confirm" = "yes" ] || exit 1; \
		echo "Initialising prod backend..."; \
		cd terraform/environments/prod && terraform init \
			-backend-config="bucket=$$TF_STATE_BUCKET" \
			-backend-config="key=prod/terraform.tfstate" \
			-backend-config="region=us-east-1" \
			-backend-config="dynamodb_table=$$TF_STATE_LOCK_TABLE" \
			-backend-config="encrypt=true"; \
		DOMAIN_ARGS=""; \
		[ -n "$(DOMAIN_NAME)" ] && DOMAIN_ARGS="$$DOMAIN_ARGS -var=domain_name=$(DOMAIN_NAME)"; \
		[ -n "$(ACM_CERTIFICATE_ARN)" ] && DOMAIN_ARGS="$$DOMAIN_ARGS -var=certificate_arn=$(ACM_CERTIFICATE_ARN)"; \
		echo "Destroying application infrastructure..."; \
		terraform destroy -auto-approve $$DOMAIN_ARGS \
			-var="bootstrap_image_uri=public.ecr.aws/lambda/provided:al2023"; \
		echo "Emptying state bucket..."; \
		aws s3 rm s3://$$TF_STATE_BUCKET --recursive; \
		echo "Initialising bootstrap..."; \
		cd ../../../terraform/bootstrap && terraform init; \
		BOOTSTRAP_ARGS=""; \
		[ -n "$(DOMAIN_NAME)" ] && BOOTSTRAP_ARGS="-var=domain_name=$(DOMAIN_NAME)"; \
		echo "Destroying bootstrap infrastructure..."; \
		terraform destroy -auto-approve $$BOOTSTRAP_ARGS; \
		echo "Destroy complete"; \
	'

fmt: ## Format Go and Terraform files
	go fmt ./...
	cd terraform && terraform fmt -recursive

lint: ## Run Go vet, staticcheck, and Terraform fmt check
	go vet ./...
	@which staticcheck > /dev/null && staticcheck ./... || echo "staticcheck not installed"
	cd terraform && terraform fmt -check -recursive
