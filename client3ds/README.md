# Ships 3D for the Nintendo 3DS

A client for the game server in this repository, for the New Nintendo 3DS (homebrew, `.3dsx`).
**Status: experiment**, written with an AI assistant and pending review. The plan is in
[docs/proposals/1-nintendo-3ds-client.md](../docs/proposals/1-nintendo-3ds-client.md).

```bash
tinygo build -target=nintendo3ds -ldflags "-X runtime.heapMegabytes=64" -o ships3ds.3dsx ./cmd/ships3ds
```

`runtime.heapMegabytes=64` leaves C its share of the console's memory: without it TinyGo takes all
it can, and the socket service doesn't find the megabyte it needs. `-X main.server=…` starts with
another server than the real one.

It needs [TinyGo for 3DS](https://github.com/akrck02/tinygo-3ds) and devkitPro's libraries, and
local copies of go-3ds (branch `feature/network`) and go-renderer (branch
`feature/3ds-moving-light`), at the paths `go.mod` names.

Playing: touch the title screen, type the server as `address:port` with **Change** (the system
keyboard), **Log in** to fly your own designs too (or stay a guest), choose a ship and **Connect**. A server typed
as `https://name:port` (the real one is `https://jonbul.ddns.net:3000`) is talked to over TLS; one
typed without, such as one at home, in plain text. In the game the touch screen is a 3D radar, as the browser's (a flat one in Settings), that turns
with the ship and sweeps once every 5 seconds, with the speed up its right edge; it flashes red when you are hit. Circle pad: nose and
turn. R or A: fire. C-stick, or left and right on the cross: roll. C-stick up and down, X and B:
faster and slower. With *Steer by tilting* on, tipping and turning the console steers too; L
centres it. START opens the options: game, Figures, Settings and Controls.

The packages `websocket`, `protocol`, `flight`, `ship`, `game`, `radar`, `figures`, `chase` and `tilt` are plain Go, tested on the PC
with `go test ./...`; only `cmd/ships3ds` needs the console.
