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
keyboard), choose a ship and **Connect** (plain `ws://`: no TLS yet). In the game the touch screen
is a radar that turns with the ship and sweeps once every 5 seconds, with the speed up its right
edge; it flashes red when you are hit. On the top screen a cross ahead of the nose shows where bullets go,
and the camera follows a little late, as in the browser. START opens the options: the game and its players, the
**Figures** (frames, CPU and GPU time, triangles, heap, network), the **Settings** (which way the
circle pad pitches, fine aim) and the **Controls**.

The packages `websocket`, `protocol`, `flight`, `ship`, `game`, `radar` and `figures` are plain Go, tested on the PC
with `go test ./...`; only `cmd/ships3ds` needs the console.
