.PHONY: build run test

# build project
build:
	go build -v party-buddy

# build and run project
run: build
	./party-buddy

# run tests recursively with data race detection
test:
	go test -race ./...

# TODO
# add docker-compose
