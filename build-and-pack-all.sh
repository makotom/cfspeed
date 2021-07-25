#!/bin/bash

set -euo pipefail

OS_LIST=("linux" "windows" "darwin")
ARCH_LIST=("amd64" "arm64")
BUILD_NAME=${BUILD_NAME:-"git"}
BUILD_ANNOTATION="$(date --iso-8601=seconds)"

for os in "${OS_LIST[@]}"
do
    if [ "${os}" == "windows" ]
    then
        ext=".exe"
    else
        ext=""
    fi

    for arch in "${ARCH_LIST[@]}"
    do
        if [ "${os}" == "windows" ] && [ "${arch}" == "arm64" ]
        then
            continue
        fi

        echo "${os}/${arch}"
        mkdir -p "dist/${os}/${arch}"
        GOOS=$os GOARCH=$arch go build -ldflags "-X main.BuildName=${BUILD_NAME} -X main.BuildAnnotation=${BUILD_ANNOTATION}" -o "dist/${os}/${arch}/cfspeed${ext}" .

        pushd "dist/${os}/${arch}"
        if [ "${os}" == "linux" ]
        then
            tar -czf "../../cfspeed-${BUILD_NAME}-${os}-${arch}.tar.gz" "cfspeed${ext}"
        else
            zip -r "../../cfspeed-${BUILD_NAME}-${os}-${arch}.zip" "cfspeed${ext}"
        fi
        popd
    done
done
