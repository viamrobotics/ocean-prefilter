#!/bin/bash

# check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Docker is not installed. Please install Docker and try again."
    exit 1
fi

# check the architecture
ARCH=$(uname -m)

# pull the Docker image and log in
DOCKER_IMAGE="ghcr.io/viamrobotics/ocean-prefilter:$ARCH"

docker pull $DOCKER_IMAGE

if [ $? -ne 0 ]; then
    echo "Failed to pull Docker image: $DOCKER_IMAGE"
    exit 1
fi

docker run --rm \
    -e ARCH_TAG=$ARCH \
    -v "$(pwd)":/workspace \
    -w /workspace \
    $DOCKER_IMAGE \
    /bin/bash -c "make ocean-prefilter && make ocean-prefilter-appimage && make module.tar.gz"

