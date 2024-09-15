OUTPUT=./build
BINARY_NAME=gocalsend

all: clean build

setup:
	mkdir -p ${OUTPUT}
clean:
	go clean
	rm -rf ${OUTPUT}
build: setup
	go build -o ${OUTPUT}

run: build
	${OUTPUT}/${BINARY_NAME}
release: clean
	mkdir -p ${OUTPUT}/release
	go build -ldflaggs="-s -w -X" -o ${OUTPUT}/release/${BINARY_NAME}

# Utilities
multicaster: setup
	go build -o ${OUTPUT}/multicaster ./cmd/multicaster

dummyep: setup
	go build -o ${OUTPUT}/dummyep ./cmd/dummyEndpoint
