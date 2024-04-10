
ocean-prefilter: main.go
	go build -o ocean-prefilter main.go

TAG_VERSION?=latest
appimage: export TAG_NAME = ${TAG_VERSION}
appimage: ocean-prefilter
	cd packaging/appimages && \
	mkdir -p deploy && \
	rm -f deploy/ocean-prefilter* && \
	appimage-builder --recipe ocean-prefilter-aarch64.yml
	cp ./packaging/appimages/ocean-prefilter-*-aarch64.AppImage  ./packaging/appimages/deploy/
