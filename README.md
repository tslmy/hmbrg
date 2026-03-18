# hmbrg

[![prek](https://img.shields.io/endpoint?url=https://raw.githubusercontent.com/j178/prek/master/docs/assets/badge-v0.json)](https://github.com/j178/prek)
[![codecov](https://codecov.io/gh/tslmy/hmbrg/branch/main/graph/badge.svg)](https://codecov.io/gh/tslmy/hmbrg)

A tiny SDL2 GUI client for turning switches on/off via Homebridge, for [Miyoo Mini Flip (MMF)][mmf] running [OnionOS][oos].

[mmf]: https://lomiyoo.com/products/miyoo-mini-flip
[oos]: https://onionui.github.io/

![demo](https://media1.giphy.com/media/v1.Y2lkPTc5MGI3NjExeWV4empyMmk0N2s2bjc0M3JkMjNyeTNlZ2pkNWI4cmttc3h1aHlwdiZlcD12MV9pbnRlcm5hbF9naWZfYnlfaWQmY3Q9Zw/lRVYPfjv4b0DR5Cdfi/giphy.gif)

## Setup

After cloning the repo, copy `config.example.toml` to `config.toml`. Fill in your Homebridge details in the latter:

```toml
endpoint = "http://homebridge.local:8581"
username = "your-username"
password = "your-password"
otp = ""
show_all = false
dump_accessories = false
```

## Usage

Controls:

* D-pad: move selection
* A button (or Enter/Space on keyboard): toggle on/off
* Menu button (or Home on keyboard): exit

### Build for -- and run on -- macOS

Prerequisites:

* Golang installed. I use `asdf` to manage compiler versions, so you can refer to `.tool-versions` for the version I use.
* The command runner [`just`](https://github.com/casey/just).
* [SDL2 headers](https://www.libsdl.org/), which provides low-level access to audio, keyboard, mouse, joystick, and graphics.

You can install all of them via Homebrew in one command: `brew install golang just sdl2`.

Run `just run`. This GUI program is keyboard-only:

* Arrow keys: move selection
* Enter/Space: toggle on/off

### Build for -- and deploy to --- MMF

We provide a toolchain via Docker (see `Dockerfile`).

**We need to link to the SLD2 libraries compiled for MMF**. I haven't yet got to automate its compilation, so let's just fetch a pre-compiled version from the beautiful [MiyooPod][myp] project. Just run `just sdl2-fetch` to get it ready under `third_party/miyoopod-libs`. Technical details:

* The Docker build will use SDL2 headers from `libsdl2-dev` inside the container, and link against the MMF SDL2 shared libraries in `third_party/miyoopod-libs`.
* The MMF build uses a small set of weak SDL2 stubs behind the `mmf` build tag to accommodate
the older SDL2 build shipped with MiyooPod.

[myp]: https://github.com/danfragoso/miyoopod/

**We also need the sysroot from the device**. Again, I haven't yet got to exploore the famous [`shauninman/union-miyoomini-toolchain`][mtc] repo, so I'll just copy the files from the real device for now. Similarly, run `just sysroot-fetch <mmf-host>` to get it ready under `third_party/mmf-sysroot`.

[mtc]: https://github.com/shauninman/union-miyoomini-toolchain

Run `just build` and find the binary at `dist/hmbrg`. Afterwards, run `just deploy <mmf-host>` to copy it to the MMF. (Requires SSH to be enabled on the MMF, of course.)

## Troubleshooting

### Expected accessories aren't appearing

Try enabling `show_all` in `config.toml` and run the program again.

Alternatively, set `dump_accessories` to `true` and find the `accessories_dump.json`. This is what `hmbrg` gets from the Homebridge server.

### "undefined: sdl.*" while building

Your Go build likely has `CGO_ENABLED=0` or SDL2 headers are missing.
Ensure `CGO_ENABLED=1` and that SDL2 development libraries are installed on your system.

### "GLIBC_2.32 not found" on MMF

If the program crashes after loading, check `/mnt/SDCARD/App/hmbrg/hmbrg.log` (assuming your MMF is running OnionOS). If it says `GLIBC_2.32 not found`, your binary was linked against a newer glibc.

Use a sysroot from the device so the linker targets MMF's glibc:

```bash
just sysroot-fetch <mmf-host>
just build
```

This pulls `/lib` and `/usr/lib` from the device into `third_party/mmf-sysroot` and the Docker build will use it automatically.
