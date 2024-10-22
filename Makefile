OUTPUT=./build
BINARY_NAME=gocalsend

all: clean build uploader multicaster dummyep

setup:
	mkdir -p ${OUTPUT}
clean:
	go clean
	rm -rf ${OUTPUT}
	rm -rf ./cert
build: setup
	go build -o ${OUTPUT}/${BINARY_NAME} ./cmd/gocalsend

run: build
	${OUTPUT}/${BINARY_NAME}
release: clean
	mkdir -p ${OUTPUT}/release
	go build -ldflaggs="-s -w -X" -o ${OUTPUT}/release/${BINARY_NAME}

# Utilities
gclsnd: build

multicaster: setup
	go build -o ${OUTPUT}/multicaster ./cmd/multicaster

dummyep: setup
	go build -o ${OUTPUT}/dummyep ./cmd/dummyEndpoint

uploader: setup
	go build -o ${OUTPUT}/uploader ./cmd/uploader
