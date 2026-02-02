set shell := ["zsh", "-cu"]

build:
	go build -o bin/prompter ./cmd/prompter

build-run:
	go build -o bin/prompter ./cmd/prompter && ./bin/prompter

dev-build:
	go build -gcflags "all=-N -l" -o bin/prompter ./cmd/prompter

cross-platform:
	./scripts/build.sh

install:
	install -m 0755 bin/prompter /usr/local/bin/prompter

test:
	go test ./...

test-verbose:
	go test -v ./...

clean:
	rm -rf bin
