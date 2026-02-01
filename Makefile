.PHONY: build dev-build cross-platform install test test-verbose clean

build:
	GOCACHE=$(CURDIR)/.gocache go build -o bin/prompter ./cmd/prompter	

build-run:
	GOCACHE=$(CURDIR)/.gocache go build -o bin/prompter ./cmd/prompter	&& ./bin/prompter

dev-build:
	GOCACHE=$(CURDIR)/.gocache go build -gcflags "all=-N -l" -o bin/prompter ./cmd/prompter

cross-platform:
	./scripts/build.sh

install:
	install -m 0755 bin/prompter /usr/local/bin/prompter

test:
	GOCACHE=$(CURDIR)/.gocache go test ./...

test-verbose:
	GOCACHE=$(CURDIR)/.gocache go test -v ./...

clean:
	rm -rf bin
