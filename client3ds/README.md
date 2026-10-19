# Ships 3D for the Nintendo 3DS

A client for the game server in this repository, for the New Nintendo 3DS (homebrew, `.3dsx`).
**Status: experiment**, written with an AI assistant and pending review. The plan is in
[docs/proposals/1-nintendo-3ds-client.md](../docs/proposals/1-nintendo-3ds-client.md).

```bash
tinygo build -target=nintendo3ds -ldflags "-X main.host=192.168.1.20" -o ships3ds.3dsx ./cmd/ships3ds
```

It needs [TinyGo for 3DS](https://github.com/akrck02/tinygo-3ds) and devkitPro's libraries, and
a copy of go-3ds with its `network` package next to this repository (see `go.mod`).

Playing: touch the title screen, type the server as `address:port` with **Change** (the system
keyboard), choose a ship and **Connect** (plain `ws://`: no TLS yet). In the game the touch screen is a radar that turns with the ship and sweeps once every 5
seconds, with the speed up its right edge; it flashes red when you are hit. Circle pad: nose and
turn. R or A: fire. C-stick, or left and right on the cross: roll. C-stick up and down, X and B:
faster and slower. With *Steer by tilting* on, tipping and turning the console steers too; L
centres it. START opens the options: game, Figures, Settings and Controls.

The packages `websocket`, `protocol`, `flight`, `ship`, `game`, `radar`, `figures`, `chase` and `tilt` are plain Go, tested on the PC
with `go test ./...`; only `cmd/ships3ds` needs the console.
