EXT=
ifeq (${GOOS},windows)
    EXT=.exe
endif

.PHONY: all
all: clean build install

.PHONY: clean build install

build: 
	go build -o ./build/gweb .

clean:
	rm -rf ./build

install: build
	install ./build/gweb ${GOPATH}/bin/gweb${EXT}

