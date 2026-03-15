#!/usr/bin/env bash
set -euo pipefail

: "${SDL2_INCLUDE:=}"
: "${SDL2_LIB:=/build/sdl2}"
: "${OUT:=/build/src/dist/hmbrg}"
: "${MMF_SYSROOT:=}"

cd /build/src

if [ -d "/build/sdl2/lib" ]; then SDL2_LIB="/build/sdl2/lib"; fi

# Prefer standard SDL2 include dir, but add multiarch include for SDL_config.
if [ -n "${SDL2_INCLUDE}" ] && [ ! -f "${SDL2_INCLUDE}/SDL.h" ]; then
  SDL2_INCLUDE=""
fi
if [ -z "${SDL2_INCLUDE}" ]; then
  for candidate in \
    "/usr/include/SDL2" \
    "/usr/include/arm-linux-gnueabihf/SDL2" \
    "/usr/include"
  do
    if [ -f "${candidate}/SDL.h" ]; then
      SDL2_INCLUDE="${candidate}"
      break
    fi
  done
fi

SYSROOT_FLAG=""
SYSROOT_LIBS=""
RPATH_LINK=""
if [ -n "${MMF_SYSROOT}" ] && [ -d "${MMF_SYSROOT}/lib" ]; then
  SYSROOT_FLAG="--sysroot=${MMF_SYSROOT}"
  SYSROOT_LIBS="-L${MMF_SYSROOT}/lib -L${MMF_SYSROOT}/usr/lib -Wl,--dynamic-linker=/lib/ld-linux-armhf.so.3"
  RPATH_LINK="-Wl,-rpath-link,${MMF_SYSROOT}/lib -Wl,-rpath-link,${MMF_SYSROOT}/usr/lib"
fi

mkdir -p /tmp/pkgconfig
cat >/tmp/pkgconfig/sdl2.pc <<EOF
prefix=/build/sdl2
exec_prefix=\${prefix}
libdir=${SDL2_LIB}
includedir=${SDL2_INCLUDE}

Name: sdl2
Description: Simple DirectMedia Layer
Version: 2.0.0
Libs: -L\${libdir} -lSDL2
Cflags: -I\${includedir} -I/usr/include/arm-linux-gnueabihf
EOF

export PKG_CONFIG_PATH=/tmp/pkgconfig
EXTRA_INCLUDE=""
if [ -d "/usr/include/arm-linux-gnueabihf/SDL2" ]; then EXTRA_INCLUDE="-I/usr/include/arm-linux-gnueabihf"; fi
export CGO_CFLAGS="${SYSROOT_FLAG} -I${SDL2_INCLUDE} ${EXTRA_INCLUDE} -I/usr/include"
export CGO_LDFLAGS="${SYSROOT_FLAG} ${SYSROOT_LIBS} -L${SDL2_LIB} -lSDL2 -lSDL2_EGL -lSDL2_GLESv2 -lSDL2_json-c -lSDL2_mixer -lSDL2_z -l:libmpg123.so.0 -Wl,-rpath-link,${SDL2_LIB} ${RPATH_LINK} -Wl,--allow-shlib-undefined"

if [ -n "${SYSROOT_FLAG}" ]; then
  export CC="arm-linux-gnueabihf-gcc ${SYSROOT_FLAG}"
  export CXX="arm-linux-gnueabihf-g++ ${SYSROOT_FLAG}"
fi

# Use module cache inside container for repeat builds
export GOMODCACHE=/build/gomodcache
export GOCACHE=/build/gocache

/usr/local/go/bin/go build -tags mmf -o "${OUT}" ./src

echo "Built ${OUT}"
