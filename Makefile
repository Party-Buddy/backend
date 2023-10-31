.PHONY: build run test

# build project
build:
	go build party-buddy

# build and run project
run:
	go build party-buddy && ./party-buddy

# run tests recursively with data race detection
test:
	go test -race ./...

# TODO
# add docker-compose
