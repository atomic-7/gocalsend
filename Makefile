OUTPUT=build
BINARY_NAME=gocalsend

# Windoof is such a shitshow
SEP:=/
RMDIR=rm -rf
MKDIR=mkdir -p
DIR=${OUTPUT}
ifeq ($(OS), Windows_NT)
	SEP=$(strip \ )
	# define rmdir dynamically so DIR can be updated
	RMDIR:=if exist ${DIR} rmdir
	MKDIR:=mkdir
	WINOPTS=/s /q 2>nul
	EXT=.exe
endif

all: clean release

setup:
	${MKDIR} ${OUTPUT}
clean:
	go clean
	${RMDIR} ${OUTPUT} ${WINOPTS}
	$(eval DIR=cert)
	${RMDIR} cert ${WINOPTS}
build: clean setup
	go build -o ${OUTPUT}${SEP}${BINARY_NAME}${EXT} .${SEP}cmd${SEP}gocalsend

run: build
	${OUTPUT}${SEP}${BINARY_NAME}

# todo use -X flag to put latest tag into the binary
release: clean 
	${MKDIR} ${OUTPUT}${SEP}release
	go build -ldflags="-s -w" -o ${OUTPUT}${SEP}release${SEP}${BINARY_NAME}${EXT} .${SEP}cmd${SEP}${BINARY_NAME}
	go build -ldflags="-s -w" -o ${OUTPUT}${SEP}release${SEP}gclsnd${EXT} .${SEP}cmd${SEP}gclsnd

# Utilities
# todo: make a build that uses no external dependencies
#gclsnd: build
debug: build multicaster dummyep uploader tui

multicaster: setup
	go build -o ${OUTPUT}${SEP}multicaster${EXT} .${SEP}cmd${SEP}multicaster

dummyep: setup
	go build -o ${OUTPUT}${SEP}dummyep${EXT} .${SEP}cmd${SEP}dummyEndpoint

uploader: setup
	go build -o ${OUTPUT}${SEP}uploader${EXT} .${SEP}cmd${SEP}uploader

tui: setup
	go build -o ${OUTPUT}${SEP}gocalsend${EXT} .${SEP}cmd${SEP}gocalsend

cli: setup
	go build -o ${OUTPUT}${SEP}gclsnd${EXT} .${SEP}cmd${SEP}gclsnd
