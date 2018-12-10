NAME          := asg-check-consul
VERSION       := $(shell git describe --tags --abbrev=1)
LDFLAGS       := -linkmode external -extldflags -static -X 'main.version=$(VERSION)'

.PHONY: setup
setup:
	go get -u -v github.com/golang/dep/cmd/dep
	go get -u -v github.com/mitchellh/gox
	go get -u -v github.com/tcnksm/ghr

.PHONY: build
build:
	dep ensure -v
	mkdir -p bin
	gox -osarch "linux/amd64" -ldflags "$(LDFLAGS)" -output bin/$(NAME)_{{.OS}}_{{.Arch}}
	strip bin/*

.PHONY: publish
publish:
	ghr -replace $(VERSION) bin
