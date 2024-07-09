#!/bin/bash
#This script invoked via a make target by the Dockerfile
#which builds a cli wrapper container that contains all release images

#Keeping it similar to ROSA official releases which only publish amd64 to mirror
#This list can be modified as needed if additional os or arch support is needed
archs=(amd64)
oses=(darwin windows linux)

REL_VER=$(git describe --tags --abbrev=0 | sed "s/v//")
mkdir -p releases

build_release() {
for os in ${oses[@]}
do
  for arch in ${archs[@]}
  do
    if [[ $os == "windows" ]]; then
        extension=".exe"
    fi
    GOOS=${os} GOARCH=${arch} go build -o /tmp/ocm_${os}_${arch} ./cmd/ocm
    mv /tmp/ocm_${os}_${arch} ocm_v${REL_VER}${extension}
    zip releases/ocm_${REL_VER}_${os}_${arch}.zip ocm_v${REL_VER}${extension}
    rm ocm_v${REL_VER}${extension}
  done
done
cd releases && sha256sum *zip > ocm_${REL_VER}_SHA256SUMS
}

build_release
