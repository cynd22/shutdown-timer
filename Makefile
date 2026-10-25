VERSION ?= dev
GO ?= go

.PHONY: build test package clean

build:
	$(GO) build -trimpath -ldflags '-s -w -X main.version=$(VERSION)' -o shutdown-timer .

test:
	$(GO) test -race -tags ci ./...
	$(GO) vet -tags ci ./...

package: build
	mkdir -p dist
	tar -czf dist/ShutdownTimer.tar.gz shutdown-timer icon.png install.sh README.txt LICENSE
	cd dist && sha256sum ShutdownTimer.tar.gz > SHA256SUMS

clean:
	rm -f shutdown-timer
	rm -rf dist
