SPACE_VERSION = v0.0.8
LINUX_PLATFORM = x86_64-linux
LINUX_ARM_PLATFORM = arm64-linux
MAC_PLATFORM = x86_64-darwin
MAC_ARM_PLATFORM = arm64-darwin
WINDOWS_PLATFORM = x86_64-windows

LDFLAGS := -X github.com/deta/pc-cli/cmd.spaceVersion=$(SPACE_VERSION) $(LDFLAGS)

.PHONY: build clean

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS) -X github.com/deta/pc-cli/cmd.platform=$(LINUX_PLATFORM)" -o build/space
	cd build && zip -FSr space-$(LINUX_PLATFORM).zip space

build-linux-arm:
	GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS) -X github.com/deta/pc-cli/cmd.platform=$(LINUX_ARM_PLATFORM)" -o build/space
	cd build && zip -FSr space-$(LINUX_ARM_PLATFORM).zip space

build-win:
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS) -X github.com/deta/pc-cli/cmd.platform=$(WINDOWS_PLATFORM)" -o build/space.exe
	cd build && zip -FSr space-$(WINDOWS_PLATFORM).zip space.exe

build-mac:
	GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS) -X github.com/deta/pc-cli/cmd.platform=$(MAC_PLATFORM)" -o build/space
	cd build && zip -FSr space-$(MAC_PLATFORM).zip space

build-mac-arm:
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS) -X github.com/deta/pc-cli/cmd.platform=$(MAC_ARM_PLATFORM)" -o build/space
	cd build && zip -FSr space-$(MAC_ARM_PLATFORM).zip space

build: build-linux build-win build-mac build-mac-arm build-linux-arm

notarize-mac: build-mac
	gon ./.x86_64.hcl

notarize-mac-arm: build-mac-arm
	gon ./.arm64.hcl

clean:
	rm -rf out build/space build/space.exe build/*.zip

