# ghstats — Go + templ + HTMX + Tailwind. Deployed as a container on AWS App Runner.

APP            := ghstats
BIN            := bin/$(APP)
TEMPL_VERSION  := v0.3.960

# --- AWS / deploy config (override on the command line or via env) ---
AWS_REGION            ?= us-east-1
AWS_ACCOUNT_ID        ?=
ECR_REPO              ?= $(APP)
IMAGE_TAG             ?= latest
ECR_REGISTRY          := $(AWS_ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com
IMAGE                 := $(ECR_REGISTRY)/$(ECR_REPO):$(IMAGE_TAG)
APPRUNNER_SERVICE_ARN ?=

.PHONY: all generate build run test vet fmt tidy css clean \
        docker-build docker-push deploy ecr-login help

all: build

## generate: render templ components to *_templ.go (pinned templ version)
generate:
	go run github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION) generate

## build: generate templ then compile a static binary
build: generate
	CGO_ENABLED=0 go build -trimpath -o $(BIN) ./cmd/server

## run: generate templ then run the server locally
run: generate
	go run ./cmd/server

## test: run the test suite
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## fmt: format Go and templ sources
fmt:
	go fmt ./...
	go run github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION) fmt .

## tidy: tidy module dependencies
tidy:
	go mod tidy

## css: (optional, production) build a self-hosted Tailwind stylesheet.
## The app uses the Tailwind Play CDN by default; switch to a built sheet by
## installing the Tailwind standalone CLI and wiring the output into pages.templ.
css:
	@echo "Install the Tailwind standalone CLI, then:"
	@echo "  tailwindcss -i internal/web/input.css -o internal/web/static/app.css --minify"

## clean: remove build artifacts and generated templ files
clean:
	rm -rf bin
	find . -name '*_templ.go' -delete

## docker-build: build the container image locally
docker-build:
	docker build -t $(APP):$(IMAGE_TAG) .

## ecr-login: authenticate Docker to the account's ECR registry
ecr-login:
	aws ecr get-login-password --region $(AWS_REGION) \
		| docker login --username AWS --password-stdin $(ECR_REGISTRY)

## docker-push: tag and push the image to ECR
docker-push: docker-build ecr-login
	docker tag $(APP):$(IMAGE_TAG) $(IMAGE)
	docker push $(IMAGE)

## deploy: push the image and trigger an App Runner deployment
deploy: docker-push
	aws apprunner start-deployment \
		--service-arn $(APPRUNNER_SERVICE_ARN) \
		--region $(AWS_REGION)

## help: list available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'
