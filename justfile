set shell := ["zsh", "-cu"]

build:
	go build -o bin/prompter ./cmd/prompter

build-run:
	go build -o bin/prompter ./cmd/prompter && ./bin/prompter

watch:
	@rg --files | entr -r sh -c 'sleep 0.5; go build -o bin/prompter ./cmd/prompter'

dev-build:
	go build -gcflags "all=-N -l" -o bin/prompter ./cmd/prompter

cross-platform:
	./scripts/build.sh

build-aur version:
	@if git remote get-url origin >/dev/null 2>&1; then \
		git tag {{version}}; \
		git push origin {{version}}; \
	else \
		echo "Skipping tag push: no 'origin' remote configured."; \
		git tag {{version}}; \
	fi
	@tarball_url="https://github.com/imdevan/prompter/archive/refs/tags/{{version}}.tar.gz"; \
	sha256="$$(curl -L -s "$${tarball_url}" | sha256sum | awk '{print $$1}')"; \
	VERSION={{version}} AUR_SOURCE_SHA256="$${sha256}" ./scripts/build_aur.sh

publish-aur aur_dir:
	./scripts/aur_publish.sh {{aur_dir}}

install:
	install -m 0755 bin/prompter /usr/local/bin/prompter

test:
	go test ./...

test-verbose:
	go test -v ./...

clean:
	rm -rf bin
