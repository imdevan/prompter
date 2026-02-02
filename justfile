set shell := ["zsh", "-cu"]

build:
	GOCACHE={{justfile_directory()}}/.gocache go build -o bin/prompter ./cmd/prompter

build-run:
	GOCACHE={{justfile_directory()}}/.gocache go build -o bin/prompter ./cmd/prompter && ./bin/prompter

dev-build:
	GOCACHE={{justfile_directory()}}/.gocache go build -gcflags "all=-N -l" -o bin/prompter ./cmd/prompter

cross-platform:
	./scripts/build.sh

install:
	install -m 0755 bin/prompter /usr/local/bin/prompter

test:
	GOCACHE={{justfile_directory()}}/.gocache go test ./...

test-verbose:
	GOCACHE={{justfile_directory()}}/.gocache go test -v ./...

clean:
	rm -rf bin
