APP_NAME := docker-pihole-customdns
VERSION := 0.0.1

ifeq ($(shell command -v podman 2> /dev/null),)
    DOCKER_CMD=docker
else
    DOCKER_CMD=podman
endif

build_dir := build
darwin_amd64 := $(build_dir)/darwin_amd64
darwin_arm64 := $(build_dir)/darwin_arm64
linux_amd64 := $(build_dir)/linux_amd64
linux_arm := $(build_dir)/linux_arm
linux_arm64 := $(build_dir)/linux_arm64
windows_amd64 := $(build_dir)/windows_amd64

.PHONY: clean
clean:
	@rm -rf $(build_dir)

.PHONY: darwin_amd64
darwin_amd64: build_dir
	@echo "Building Darwin/amd64 binary..."
	@GOOS=darwin GOARCH=amd64 go build -o $(darwin_amd64)/$(APP_NAME) -ldflags="-s -w -X main.version=$(VERSION)" .

.PHONY: darwin_arm64
darwin_arm64: build_dir
	@echo "Building Darwin/arm64 binary..."
	@GOOS=darwin GOARCH=arm64 go build -o $(darwin_arm64)/$(APP_NAME) -ldflags="-s -w -X main.version=$(VERSION)" .

.PHONY: linux_amd64
linux_amd64: build_dir
	@echo "Building Linux/amd64 binary..."
	@GOOS=linux GOARCH=amd64 go build -o $(linux_amd64)/$(APP_NAME) -ldflags="-s -w -X main.version=$(VERSION)" .

.PHONY: linux_arm
linux_arm: build_dir
	@echo "Building Linux/arm binary..."
	@GOOS=linux GOARCH=arm go build -o $(linux_arm)/$(APP_NAME) -ldflags="-s -w -X main.version=$(VERSION)" .

.PHONY: linux_arm64
linux_arm64: build_dir
	@echo "Building Linux/arm64 binary..."
	@GOOS=linux GOARCH=arm64 go build -o $(linux_arm64)/$(APP_NAME) -ldflags="-s -w -X main.version=$(VERSION)" .

.PHONY: windows_amd64
windows_amd64: build_dir
	@echo "Building Windows/amd64 binary..."
	@GOOS=windows GOARCH=amd64 go build -o $(windows_amd64)/$(APP_NAME).exe -ldflags="-s -w -X main.version=$(VERSION)" .

.PHONY: build_dir
build_dir:
	@mkdir -p $(build_dir)
	@mkdir -p $(darwin_amd64) $(darwin_arm64) $(linux_amd64) $(linux_arm) $(linux_arm64) $(windows_amd64)

.PHONY: build_all
build-all: darwin_amd64 darwin_arm64 linux_amd64 linux_arm linux_arm64 windows_amd64

.PHONY: docker_build
docker_build:
	@$(DOCKER_CMD) build -t $(APP_NAME):$(VERSION) -f ./Docker/Dockerfile .
#-v /var/run/docker.sock:/var/run/docker.sock:ro

.PHONY: docker_run
docker_run:
	@$(DOCKER_CMD) run -d --name $(APP_NAME) --restart=unless-stopped  -e DPC_PIHOLE_URL=$(DPC_PIHOLE_URL) -e DPC_DEFAULT_TARGET_IP=$(DPC_DEFAULT_TARGET_IP) -e DPC_PIHOLE_API_TOKEN=$(DPC_PIHOLE_API_TOKEN) $(APP_NAME):$(VERSION)

.PHONY: update_modules
update_modules:
	@go get -u
	@go mod tidy
