
ocean-prefilter: main.go
	go build -o ocean-prefilter main.go

TAG_VERSION?=latest
ocean-prefilter-appimage: export TAG_NAME = ${TAG_VERSION}
ocean-prefilter-appimage: ocean-prefilter
	cd packaging/appimages && \
	mkdir -p deploy && \
	rm -f deploy/ocean-prefilter* && \
	appimage-builder --recipe ocean-prefilter-aarch64.yml
	cp ./packaging/appimages/ocean-prefilter-*-aarch64.AppImage  ./packaging/appimages/deploy/
	cp ./packaging/appimages/deploy/ocean-prefilter-${TAG_VERSION}-aarch64.AppImage ocean-prefilter-appimage
	chmod a+x ocean-prefilter-appimage

module: ocean-prefilter-appimage
	tar czf module.tar.gz ocean-prefilter-appimage

clean:
	rm -rf ocean-prefilter ocean-prefilter-appimage packaging/appimages/deploy/ocean-prefilter*
