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

VERSION ?= 0.0.3

GOOS   = $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)

.PHONY: build test all build_darwin_amd64 build_linux_amd64 build_windows_amd64 install

build:
	CGO_ENABLED=0 go build ./cmd/terraform-provider-polaris

test:
	CGO_ENABLED=0 go test -cover ./...

install: build
	@mkdir -p ~/.terraform.d/plugins/terraform.rubrik.com/rubrik/polaris/$(VERSION)/$(GOOS)_$(GOARCH)
	cp terraform-provider-polaris ~/.terraform.d/plugins/terraform.rubrik.com/rubrik/polaris/$(VERSION)/$(GOOS)_$(GOARCH)

all: build_darwin_amd64 build_linux_amd64 build_windows_amd64

build_darwin_amd64:
	CGO_ENABLED=0 GOOS="darwin" GOARCH="amd64" go build -o ./build/darwin_amd64/ ./cmd/terraform-provider-polaris

build_linux_amd64:
	CGO_ENABLED=0 GOOS="linux" GOARCH="amd64" go build -o ./build/linux_amd64/ ./cmd/terraform-provider-polaris

build_windows_amd64:
	CGO_ENABLED=0 GOOS="windows" GOARCH="amd64" go build -o ./build/windows_amd64/ ./cmd/terraform-provider-polaris
