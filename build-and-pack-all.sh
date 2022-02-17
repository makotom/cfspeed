#!/bin/bash

set -euo pipefail

GOOS_LIST=("linux" "windows" "darwin" "android")
GOARCH_LIST=("amd64" "arm64")
BUILD_NAME=${BUILD_NAME:-"git"}
BUILD_ANNOTATION="$(date --iso-8601=seconds)"
BUILD_NAME_VAR_PACKAGE="main"

if [ "${GOOS_LIST_OVERRIDE:-}" ]
then
    eval GOOS_LIST="${GOOS_LIST_OVERRIDE}"
fi
if [ "${GOARCH_LIST_OVERRIDE:-}" ]
then
    eval GOARCH_LIST="${GOARCH_LIST_OVERRIDE}"
fi

build_gc() {
    goos=$1
    goarch=$2

    GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 go build -ldflags "-X ${BUILD_NAME_VAR_PACKAGE}.BuildName=${BUILD_NAME} -X ${BUILD_NAME_VAR_PACKAGE}.BuildAnnotation=${BUILD_ANNOTATION}" -o "dist/${goos}/${goarch}/cfspeed" .

    if [ "${goos}" == "windows" ]
    then
        mv "dist/${goos}/${goarch}/cfspeed" "dist/${goos}/${goarch}/cfspeed.exe"
    fi
}

build_android() {
    goos=$1
    goarch=$2

    if [ ! -d android-ndk-r21e ]
    then
        # https://developer.android.com/ndk/downloads
        curl -JOL https://dl.google.com/android/repository/android-ndk-r23b-linux.zip
        echo 'f47ec4c4badd11e9f593a8450180884a927c330d android-ndk-r23b-linux.zip' | sha1sum -c
        unzip android-ndk-r23b-linux.zip
    fi

    if [ "${goarch}" == "arm64" ]
    then
        arch_clang="aarch64"
    elif [ "${goarch}" == "amd64" ]
    then
        arch_clang="x86_64"
    fi

    CC="$(pwd)/android-ndk-r23b/toolchains/llvm/prebuilt/linux-x86_64/bin/${arch_clang}-linux-android31-clang" \
    CXX="$(pwd)/android-ndk-r23b/toolchains/llvm/prebuilt/linux-x86_64/bin/${arch_clang}-linux-android31-clang++" \
    GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=1 go build -ldflags "-X ${BUILD_NAME_VAR_PACKAGE}.BuildName=${BUILD_NAME} -X ${BUILD_NAME_VAR_PACKAGE}.BuildAnnotation=${BUILD_ANNOTATION}" -o "dist/${goos}/${goarch}/cfspeed" .
}

package() {
    goos=$1
    goarch=$2

    pushd "dist/${goos}/${goarch}"
    if [ "${goos}" == "linux" ] || [ "${goos}" == "android" ]
    then
        tar -czf "../../cfspeed-${BUILD_NAME}-${goos}-${goarch}.tar.gz" .
    else
        zip -r "../../cfspeed-${BUILD_NAME}-${goos}-${goarch}.zip" .
    fi
    popd
}

for goos in "${GOOS_LIST[@]}"
do
    for goarch in "${GOARCH_LIST[@]}"
    do
        echo "${goos}/${goarch}"

        mkdir -p "dist/${goos}/${goarch}"

        if [ "${goos}" == "android" ]
        then
            build_android "${goos}" "${goarch}"
        else
            build_gc "${goos}" "${goarch}"
        fi

        package "${goos}" "${goarch}"
    done
done
