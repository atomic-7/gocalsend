OUTPUT=./build
BINARY_NAME=gocalsend

all: clean build

clean:
	go clean
	rm -rf ${OUTPUT}
build:
	mkdir -p ${OUTPUT}
	go build -o ${OUTPUT}

run: build
	${OUTPUT}/${BINARY_NAME}
release: clean
	mkdir -p ${OUTPUT}/release
	go build -ldflaggs="-s -w -X" -o ${OUTPUT}/release/${BINARY_NAME}
