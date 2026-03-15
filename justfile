set shell := ["zsh", "-c"]

tidy:
  go mod tidy

run:
  env -u GOOS -u GOARCH -u GOARM CGO_ENABLED=1 go run ./src

# Fetch MMF SDL2 libs from MiyooPod into third_party/miyoopod-libs.
sdl2-fetch:
  rm -rf /tmp/miyoopod-fetch
  git clone --depth 1 --filter=blob:none --sparse https://github.com/danfragoso/miyoopod.git /tmp/miyoopod-fetch
  cd /tmp/miyoopod-fetch && git sparse-checkout set libs
  rm -rf {{justfile_directory()}}/third_party/miyoopod-libs
  mkdir -p {{justfile_directory()}}/third_party/miyoopod-libs
  rsync -a /tmp/miyoopod-fetch/libs/ {{justfile_directory()}}/third_party/miyoopod-libs/
  cd /tmp/miyoopod-fetch && git rev-parse HEAD > {{justfile_directory()}}/third_party/miyoopod-libs/COMMIT.txt

# Cross-build for Miyoo Mini Flip (MMF) via Docker. (tested on macOS)
# Requires `sdl2-fetch` to be run first to populate SDL2 libs.
build:
  docker build -t hmbrg-mmf -f docker/Dockerfile docker
  mkdir -p \
    {{justfile_directory()}}/.cache/gomod \
    {{justfile_directory()}}/.cache/gocache
  docker run --rm \
    -v {{justfile_directory()}}:/build/src \
    -v {{justfile_directory()}}/third_party/miyoopod-libs:/build/sdl2 \
    -v {{justfile_directory()}}/third_party/mmf-sysroot:/build/sysroot \
    -v {{justfile_directory()}}/.cache/gomod:/build/gomodcache \
    -v {{justfile_directory()}}/.cache/gocache:/build/gocache \
    -e MMF_SYSROOT=/build/sysroot \
    -e SDL2_INCLUDE=/usr/include/arm-linux-gnueabihf/SDL2 \
    hmbrg-mmf
  mkdir -p dist/libs
  if [ -f {{justfile_directory()}}/config.toml ]; then cp -f {{justfile_directory()}}/config.toml dist/config.toml; fi
  rsync -a --delete {{justfile_directory()}}/third_party/miyoopod-libs/ dist/libs/
  chmod +x dist/hmbrg dist/launch.sh

# Deploy built files to Miyoo Mini Flip (MMF) device over SSH.
# Requires `build` to be run first and a properly configured `config.toml`.
deploy host:
  rsync -avzu --no-perms --no-owner --no-group {{justfile_directory()}}/dist/ onion@{{host}}:/mnt/SDCARD/App/hmbrg

# Fetch MMF sysroot libraries (glibc and friends) from device for compatibility.
sysroot-fetch host:
  rm -rf {{justfile_directory()}}/third_party/mmf-sysroot
  mkdir -p {{justfile_directory()}}/third_party/mmf-sysroot
  rsync -avzu --no-perms --no-owner --no-group onion@{{host}}:/lib {{justfile_directory()}}/third_party/mmf-sysroot/
  rsync -avzu --no-perms --no-owner --no-group onion@{{host}}:/usr/lib {{justfile_directory()}}/third_party/mmf-sysroot/usr/ || true
  rsync -avzu --no-perms --no-owner --no-group onion@{{host}}:/usr/lib32 {{justfile_directory()}}/third_party/mmf-sysroot/usr/ || true
  rsync -avzu --no-perms --no-owner --no-group onion@{{host}}:/customer/lib {{justfile_directory()}}/third_party/mmf-sysroot/ || true
