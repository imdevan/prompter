.PHONY: build dev-build cross-platform install test clean

build:
	go build -o bin/prompter ./cmd/prompter

dev-build:
	go build -gcflags "all=-N -l" -o bin/prompter ./cmd/prompter

cross-platform:
	./scripts/build.sh

install:
	install -m 0755 bin/prompter /usr/local/bin/prompter

test:
	go test ./...

clean:
	rm -rf bin
