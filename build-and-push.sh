#!/bin/bash

set -o nounset
set -o errexit
set -o pipefail

read -r tag < VERSION

docker buildx build --platform linux/amd64,linux/arm64 -t "sutoj/letrvu:${tag}" . --push
