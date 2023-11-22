.PHONY: build run test docker docker-build

DOCKER ?= docker
DOCKER_COMPOSE ?= $(DOCKER) compose

COMPOSE_FILE := ./compose.yaml
CONTAINER_DIR := ./container
POSTGRES_PASSWD := $(CONTAINER_DIR)/postgres-passwd.txt

# build project
build:
	go build -v party-buddy

# build and run project
run: build
	./party-buddy

# run tests recursively with data race detection
test:
	go test -race ./...

# create a dir for container data
$(CONTAINER_DIR):
	mkdir $@

# create a default postgres-passwd secret file
$(POSTGRES_PASSWD): | $(CONTAINER_DIR)
	echo zxcvbnM1 > $@

# rebuild Docker images
docker-build:
	$(DOCKER_COMPOSE) -f ./compose.yaml build

# start Docker containers
docker: $(POSTGRES_PASSWD)
	$(DOCKER_COMPOSE) -f ./compose.yaml up
