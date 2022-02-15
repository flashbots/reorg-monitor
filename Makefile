.PHONY: all build test clean lint cover cover-html build-for-docker docker-image docker-push

GOPATH := $(if $(GOPATH),$(GOPATH),~/go)
GIT_VER := $(shell git describe --tags --always --dirty="-dev")
ECR_URI := 223847889945.dkr.ecr.us-east-2.amazonaws.com/reorg-monitor

all: clean build

build:
	go build -ldflags "-X main.version=${GIT_VER}" -v -o reorg-monitor cmd/reorg-monitor/main.go

clean:
	rm -rf reorg-monitor build/

test:
	go test ./...

lint:
	gofmt -d ./
	go vet ./...
	staticcheck ./...

cover:
	go test -coverprofile=/tmp/go-bid-receiver.cover.tmp ./...
	go tool cover -func /tmp/go-bid-receiver.cover.tmp
	unlink /tmp/go-bid-receiver.cover.tmp

cover-html:
	go test -coverprofile=/tmp/go-bid-receiver.cover.tmp ./...
	go tool cover -html=/tmp/go-bid-receiver.cover.tmp
	unlink /tmp/go-bid-receiver.cover.tmp

build-for-docker:
	CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=${GIT_VER}" -v -o reorg-monitor cmd/reorg-monitor/main.go

docker-image:
	DOCKER_BUILDKIT=1 docker build . -t reorg-monitor
	docker tag reorg-monitor:latest ${ECR_URI}:${GIT_VER}
	docker tag reorg-monitor:latest ${ECR_URI}:latest

docker-push:
	docker push ${ECR_URI}:${GIT_VER}
	docker push ${ECR_URI}:latest

k8s-deploy:
	@echo "Checking if Docker image ${ECR_URI}:${GIT_VER} exists..."
	@docker manifest inspect ${ECR_URI}:${GIT_VER} > /dev/null || (echo "Docker image not found" && exit 1)
	kubectl set image deploy/deployment-reorg-monitor app-reorg-monitor=${ECR_URI}:${GIT_VER}
	kubectl rollout status deploy/deployment-reorg-monitor