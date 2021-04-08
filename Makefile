# MIT License
#
# Copyright (c) 2021 Rubrik
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

PROVIDER_VERSION ?= 0.0.1
PROVIDER = terraform.rubrik.com/rubrik/polaris/$(PROVIDER_VERSION)

GOOS   = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

.PHONY: build test install all build_darwin_amd64 build_linux_amd64 build_windows_amd64 clean

# Build for host OS/ARCH
build:
	CGO_ENABLED=0 go build -o build/$(PROVIDER)/$(GOOS)_$(GOARCH)/ ./cmd/terraform-provider-polaris

install: build
	@mkdir -p ~/.terraform.d/plugins/
	cp -r build/*/ ~/.terraform.d/plugins/

test:
	CGO_ENABLED=0 go test -cover -v ./...

clean:
	-@rm -r ./build

# Build for all supported OS/ARCH pairs and create a zip file with the
# resulting binaries.
all: build_darwin_amd64 build_linux_amd64 build_windows_amd64
	cd build; zip -r terraform-provider-polaris.zip terraform.rubrik.com

# Build for specific OS/ARCH
build_darwin_amd64:
	CGO_ENABLED=0 GOOS="darwin" GOARCH="amd64" go build -o build/$(PROVIDER)/darwin_amd64/ ./cmd/terraform-provider-polaris
	@cd build; sha256sum $(PROVIDER)/darwin_amd64/terraform-provider-polaris >> terraform-provider-polaris.sha256

build_linux_amd64:
	CGO_ENABLED=0 GOOS="linux" GOARCH="amd64" go build -o build/$(PROVIDER)/linux_amd64/ ./cmd/terraform-provider-polaris
	@cd build; sha256sum $(PROVIDER)/linux_amd64/terraform-provider-polaris >> terraform-provider-polaris.sha256

build_windows_amd64:
	CGO_ENABLED=0 GOOS="windows" GOARCH="amd64" go build -o build/$(PROVIDER)/windows_amd64/ ./cmd/terraform-provider-polaris
	@cd build; sha256sum $(PROVIDER)/windows_amd64/terraform-provider-polaris.exe >> terraform-provider-polaris.sha256
