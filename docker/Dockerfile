FROM debian:10

ARG GO_VERSION=1.22.4

RUN sed -i 's|deb.debian.org|archive.debian.org|g' /etc/apt/sources.list && \
  sed -i 's|security.debian.org|archive.debian.org|g' /etc/apt/sources.list && \
  printf 'Acquire::Check-Valid-Until "false";\n' > /etc/apt/apt.conf.d/99no-check-valid && \
  dpkg --add-architecture armhf && \
  apt-get update && \
  DEBIAN_FRONTEND=noninteractive apt-get install -y \
  ca-certificates \
  curl \
  gcc-arm-linux-gnueabihf \
  g++-arm-linux-gnueabihf \
  binutils-arm-linux-gnueabihf \
  libsdl2-dev:armhf \
  pkg-config \
  && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tgz && \
  tar -C /usr/local -xzf /tmp/go.tgz && \
  rm -f /tmp/go.tgz

ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /build

# Expect project to be mounted at /build/src and SDL2 at /build/sdl2
ENV GOOS=linux \
  GOARCH=arm \
  GOARM=7 \
  CGO_ENABLED=1 \
  CC=arm-linux-gnueabihf-gcc \
  CXX=arm-linux-gnueabihf-g++

CMD ["/build/src/docker/build.sh"]
