EXT=
ifeq (${GOOS},windows)
    EXT=.exe
endif

.PHONY: all
all: vendor clean build install

.PHONY: vendor clean build install

build: 
	go build -o ./build/gweb .

vendor: glide.lock glide.yaml
	glide install 

clean:
	rm -rf ./build

install: build
	install ./build/gweb ${GOPATH}/bin/gweb${EXT}

