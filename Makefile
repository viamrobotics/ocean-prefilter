
ocean-prefilter: main.go
	go build -o ocean-prefilter main.go

TAG_VERSION?=latest

ifeq (${ARCH_TAG},arm64)
	    ARCH_NAME = aarch64
	else ifeq (${ARCH_TAG},amd64)
	    ARCH_NAME = x86_64
	else
	    ARCH_NAME = none
	endif

ocean-prefilter-appimage: export TAG_NAME = ${TAG_VERSION}
ocean-prefilter-appimage: ocean-prefilter
	cd packaging/appimages && \
	mkdir -p deploy && \
	rm -f deploy/ocean-prefilter* && \
	appimage-builder --recipe ocean-prefilter-${ARCH_NAME}.yml
	cp ./packaging/appimages/ocean-prefilter-*-${ARCH_NAME}.AppImage  ./packaging/appimages/deploy/
	cp ./packaging/appimages/deploy/ocean-prefilter-${TAG_VERSION}-${ARCH_NAME}.AppImage ocean-prefilter-appimage
	chmod a+x ocean-prefilter-appimage

module: ocean-prefilter-appimage
	tar czf module.tar.gz ocean-prefilter-appimage

clean:
	rm -rf module.tar.gz ocean-prefilter ocean-prefilter-appimage packaging/appimages/deploy/ocean-prefilter*

# Docker stuff
BUILD_CMD = docker buildx build --pull $(BUILD_PUSH) --force-rm --no-cache --build-arg MAIN_TAG=$(MAIN_TAG) --build-arg BASE_TAG=$(BUILD_TAG) --platform linux/$(BUILD_TAG) -f $(BUILD_FILE) -t '$(MAIN_TAG):$(BUILD_TAG)' .
BUILD_PUSH = --load
BUILD_FILE = ./etc/Dockerfile.debian.bookworm

docker: docker-build docker-upload

docker-build: docker-arm64 docker-amd64

docker-upload: docker-upload-arm64 docker-upload-amd64

docker-arm64: MAIN_TAG = ghcr.io/viam-labs/ocean-prefilter
docker-arm64: BUILD_TAG = arm64
docker-arm64:
	$(BUILD_CMD)

docker-amd64: MAIN_TAG = ghcr.io/viam-labs/ocean-prefilter
docker-amd64: BUILD_TAG = amd64
docker-amd64:
	$(BUILD_CMD)

docker-upload-arm64:
	docker push 'ghcr.io/viam-labs/ocean-prefilter:arm64'

docker-upload-amd64:
	docker push 'ghcr.io/viam-labs/ocean-prefilter:amd64'

# CI targets that automatically push, avoid for local test-first-then-push workflows
docker-arm64-ci: MAIN_TAG = ghcr.io/viam-labs/ocean-prefilter
docker-arm64-ci: BUILD_TAG = arm64
docker-arm64-ci: BUILD_PUSH = --push
docker-arm64-ci:
	$(BUILD_CMD)

# CI targets that automatically push, avoid for local test-first-then-push workflows
docker-amd64-ci: MAIN_TAG = ghcr.io/viam-labs/ocean-prefilter
docker-amd64-ci: BUILD_TAG = amd64
docker-amd64-ci: BUILD_PUSH = --push
docker-amd64-ci:
	$(BUILD_CMD)

