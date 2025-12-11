# Makefile
.PHONY: all build clean test install deb rpm

VERSION := 1.0.0

all: build

build:
	@echo "Building binaries..."
	@mkdir -p build
	go build -ldflags "-X main.Version=$(VERSION)" -o build/hardware-collector-client ./cmd/client
	go build -ldflags "-X main.Version=$(VERSION)" -o build/hardware-collector-cli ./cmd/cli

test:
	go test -v ./...

clean:
	rm -rf build/

deb: build
	@echo "Building DEB package..."
	@bash scripts/build-deb.sh

rpm: build
	@echo "Building RPM package..."
	@bash scripts/build-rpm.sh

install: build
	install -m 0755 build/hardware-collector-client /usr/bin/
	install -m 0755 build/hardware-collector-cli /usr/bin/hardware-collector
	install -m 0644 configs/config.yaml /etc/hardware-collector/
	install -m 0644 systemd/hardware-collector.service /etc/systemd/system/
	systemctl daemon-reload