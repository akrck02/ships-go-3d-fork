# 1. A Nintendo 3DS client

> Status: building · Author's approval: proposal ☑ plan ☑

## Why

Ships 3D is only played in the browser today. The game is small (a few ships made of primitives,
one sphere of space, a JSON protocol), which makes it a good fit for a New Nintendo 3DS: fly with
the Circle Pad, see the ships in stereoscopic 3D on the top screen and the scoreboard on the touch
screen. The server already accepts native clients (a handshake without `Origin`), so it needs
almost no change.

## What

| Id | Requirement | Check |
|---|---|---|
| R1 | The 3DS opens a TCP connection to a host and port and exchanges bytes | Echo test run in the Azahar emulator against a local echo server |
| R2 | The 3DS speaks websocket (`ws://`): handshake, text frames both ways, masked client frames, ping/pong, close | Go tests of the websocket package on the PC, plus a round trip with the real `/ws` in Azahar |
| R3 | A guest can join with a built-in ship and appears to browser players | Join from Azahar against `scripts/runDev.sh`; the browser shows the ship |
| R4 | Ships look like their design: the 12 primitive types, Euler XYZ rotation, `mirror`, colours | Go test comparing mesh bounds with `BoundingRadius`; side-by-side screenshot with the browser |
| R5 | The player flies (Circle Pad pitch/yaw, L/R roll, X/B throttle, A fires) using the server's `Settings` and sends `state` ~20/s | Flight model test on the PC; the server's own checks accept the states |
| R6 | The player fires and hits; damage, deaths and respawns show | Hub accepts the `fire`/`hit` messages (existing hub tests) and a session in Azahar against the browser |
| R7 | The touch screen shows life, kills/deaths and the player list | Screenshot in Azahar |
| R8 | (last) `wss://` and logging in to fly one's own ships | Join against the HTTPS dev server from Azahar, then on the console |
| R9 | The server's behaviour doesn't change for browsers | `go test ./...` in the server stays green |

## Out of scope

- The ship editor on the 3DS.
- NPCs, sound, local multiplayer.
- Publishing: everything stays in a local branch until the author decides to offer it upstream.

## Design

**Where.** A separate Go module, `client3ds/`, inside this repository, next to the server it talks
to. It is a module of its own because TinyGo can't build Gin or the Mongo driver, and the server
must not depend on the renderer. The server's `go test ./...` doesn't see it.

**Pieces** (one package per idea, dependencies pointing down):

| Package | Job | Depends on |
|---|---|---|
| `client3ds/websocket` | Client side of RFC 6455 over any `io.ReadWriter`: handshake, masked text frames, ping/pong, close | standard library only, so it is tested on the PC |
| `client3ds/protocol` | The JSON messages of `game/protocol.go`, mirrored by hand like `protocol.ts` | `encoding/json` |
| `client3ds/ship` | Meshes for the 12 primitives, copied from `ships-vue-3d/src/shared/shipMesh.ts` (same segment counts, lathe profiles, wedge), Euler XYZ, `mirror`, flat colours | go-renderer `models` |
| `client3ds/flight` | `stepFlight` from `flight.ts`, the same arcade model fed by the server's `Settings` | none |
| `client3ds/game` | The world: own ship, other players interpolated from snapshots, bullets, hits tested on our side as the browser does | the packages above |
| `client3ds/cmd/ships3ds` | `app.Run` loop: input, network, drawing on top (stereo) and bottom (go-ui scoreboard) | go-3ds, go-renderer `ctr`, go-ui |

**Sockets.** tinygo-3ds (`feature/3ds`) gets TCP over libctru's `soc:u`: `socInit` with a
page-aligned buffer, then `socket/connect/send/recv` through cgo, exposed as a small
`net.Conn`-like type. Reads don't block the frame: a goroutine reads into a channel the game
drains each frame. Name resolution with `gethostbyname`.

**TLS (R8), left for last.** Options: the console's `sslc` service (no code to port, but the
cipher suites it supports are old), mbedtls from devkitPro's portlibs (modern, but adds a C
library and memory), or a plain `ws://` port for native clients. Decided when we get there, after
measuring.

**Alternatives not taken.** `gorilla/websocket` (needs `net/http`, which our TinyGo target doesn't
have); a binary protocol only for the 3DS (it would split the server in two); rendering the ships
as `.glb` built on the PC (the ships are chosen at play time, so the meshes must be built on the
console).

## Tasks

- [x] T1 — TCP sockets on the 3DS · go-3ds `network` package (branch feature/network, local) + `cmd/echo` · R1 · test: echo in Azahar
- [x] T2 — websocket client · `client3ds/websocket` · R2 · test: `TestAClientTalksToAGorillaServer`, `TestClientFramesAreMasked`
- [x] T3 — protocol messages · `client3ds/protocol` · R3 · test: `TestAWelcomeIsRead`, `TestASnapshotGivesItsMovedPlayers`, `TestOutgoingMessagesUseTheServersFieldNames`
- [x] T4 — join as a guest from the console · `cmd/ships3ds` · R3 · test: Azahar against `runDev.sh`, browser sees the ship
- [x] T5 — primitive meshes · `client3ds/ship` · R4 · test: `TestEachPrimitiveFitsTheUnitCube`, `TestAMirroredPartIsReflectedAcrossX`
- [x] T6 — draw the ships, top screen in stereo · `cmd/ships3ds` · R4 · test: screenshot next to the browser
- [x] T7 — flight · `client3ds/flight` · R5 · test: `TestRaisingTheNoseClimbs`, `TestTheWorldEdgeIsAWall`
- [x] T8 — fire, hits, damage, respawn · `client3ds/game` · R6 · test: `TestABulletAlongTheNoseHitsAShipAhead`
- [x] T9 — touch-screen scoreboard · `cmd/ships3ds` · R7 · test: screenshot in Azahar
- [x] T12 — asked for after the demo: title screen, server typed on the system keyboard (go-3ds `keyboard`), the touch screen as a sweeping radar (`client3ds/radar`), START options with Game, Figures (`client3ds/figures`), Settings and Controls tabs, the C-stick (go-3ds `app`), ship choice, aim cross and chase camera (`client3ds/chase`) · tests: `TestAShipShowsOnlyOnceTheSweepPassesIt`, `TestASecondOfFramesIsShownWhole`, `TestTheCameraLagsThenCatchesUp`, `TestTheCurveKeepsTheEdgesAndSoftensTheCentre`, `TestTheRespawnCountsDown`
- [x] T13 — fixes from the review: connect and send time out (go-3ds `network`), websocket frames out of order refused, pending bounded and reused, handshake refusal closes, header injection refused, error codes kept through wrapping, `protocol.unreadable-message`, the C-stick service released · tests: `TestFramesOutOfOrderAreRefused`, `TestARefusedHandshakeClosesTheClient`, `TestHeadersCannotCarryNewLines`, `TestTheTransportsErrorIsKept`, `TestBrokenJsonHasItsCode`, echo against an unreachable address in Azahar
- [x] T14 — many players: measured with a swarm of bots in Azahar (5 / 15 / 30 / 60 bots: 10 / 3 / under 3 / — fps before, 56 / 32 / 25 / 14 after). The costs were go-renderer's sea-floor measure redone over every lit mesh whenever one changed (a still ground mesh makes it skip), every turning ship baked again every frame (now at most two a frame, the most visible first: `client3ds/turning`), fine detail on every ship (coarse beyond 60 units: `ship.Coarse`), bullets baked again when their node changed hands (now cubes with no turn), and snapshots decoded twice · tests: `TestNoMoreThanTheBudgetIsChosen`, `TestTheNearestAndMostTurnedGoFirst`, `TestChoosingAllocatesNothingOnceWarm`, `TestTheCoarseDetailHasFarFewerTriangles`, `TestASnapshotWithTypeLastIsStillRead`
- [x] T15 — only the ships the camera can see are drawn and get a mesh (a cone round the screen's corners, up to 700 units: `chase.View.Sees`); the radar still shows them all. 60 bots: 22 fps facing them, 37 with them behind · test: `TestTheViewSeesAheadAndNotBehind`
- [x] T16 — snapshots read by hand, allocating nothing (`protocol.SnapshotReader`); and in go-renderer (local branch `feature/3ds-moving-light`): `scene.Node.Moving` lights a node on the GPU instead of baking it at every turn, at most four bakes a mesh, no seabed without water. The client marks ships and the aim cross Moving and drops its workarounds (copies of meshes per node, the budget of bakes, the still ground). 57 fps with 5, 15, 30 and 60 bots in the emulator · tests: `TestTheReaderAgreesWithEncodingJson`, `TestReadingASnapshotAllocatesNothingOnceWarm`, `TestASceneWithoutWaterHasNoSeabed`, `TestTheLeastRecentlyDrawnVariantIsTheOneReplaced` (ctr, in Azahar)
- [x] T17 — a design at the server's limit (400 parts, most mirrored, round shapes) ran the console out of memory at the fine detail (about 200,000 triangles): each design now gets the finest detail that fits a budget of triangles, near and far (`ship.Within`, a third level `ship.Minimal`), its mesh made to measure; and a welcome carrying twenty such designs passed the websocket's 1 MB limit, now 8 MB. Twenty of them around the camera: 45 fps · tests: `TestTrianglesCountsWhatBuildMakes`, `TestAHeavyDesignGetsAPlainerDetail`
- [x] T18 — five more built-in ships on the server (Kestrel, Mule, Lancer, Bulwark, Wisp: 11 to 25 parts, designed as a player might), in the 3DS's list too, chosen with arrows since a drop-down of eight ran off the touch screen · tests: `TestEveryDefaultShipIsAValidDesign`, `TestTheDefaultShipsHaveTheirIds`
- [x] T19 — logging in (half of R8): email and password on the system keyboard (go-3ds `keyboard.Password`), `POST /login` over a small HTTP client (`client3ds/web`), the session's cookie on the websocket's handshake, and the player's own designs after the built-in ones (`client3ds/account`, `GET /game/ships`); log out. Tried in Azahar against the server: joined as the test user flying one of its own ships · tests: `TestALoginSetsItsCookie`, `TestALongAnswerInChunksIsJoined`, `TestASilentServerTimesOut`, `TestAPlayerLogsInAndSeesTheirShips`, `TestAWrongPasswordIsRefused`
- [ ] T10 — TLS (the other half of R8) · decided then · R8 · test: join against the HTTPS dev server
- [x] T11 — check the server is untouched · R9 · test: `go test ./...`

## Result

Released as prereleases `demo-1` … `demo-12` in the author's private copy (akrck02/ships-go-3d),
built from `feature/3ds-client`. Tested in the Azahar emulator against the server and a bot on
the PC, and the author has joined a game from the console (server log, 2026-10-18 22:35).

What the client does: join as a guest with a built-in ship chosen on the touch screen, fly with
the circle pad (fine aim and invert settings), the C-stick or the buttons, fire and hit (the
server applies the damage), respawn, and see the others in stereoscopic 3D with an aim cross and
a lagging chase camera. The touch screen is a radar that sweeps once every 5 seconds, with the
speed, hits and the respawn countdown; START opens the game, figures, settings and controls.

Changed from the plan: sockets, the keyboard, START handling and the C-stick live in go-3ds
(`network`, `keyboard`, `app`), on its local branch `feature/network`, which is not pushed:
building the client needs that copy next to this repository (`replace` in `client3ds/go.mod`).
go-renderer's `ctr` kept one baked copy of a mesh per orientation, forever, so turning ships
filled the GPU memory, and baking them again was most of a frame; fixed in go-renderer on a local
branch, `feature/3ds-moving-light` (not pushed), which `client3ds/go.mod` points at.

Measured in the emulator with up to 60 bots (T14): the server handles 60 without errors (4,400 messages and 12.7 MB a second); the client draws 14 frames a second there, 25 with 30 bots and 56 with 5. On the console the numbers will differ: the Figures tab shows them.

Pending: R8 (TLS and login: needs a decision between mbedtls, sslc and a plain port), a
reviewer's second pass, and a person's review of all of it. The reviewer's notes on comments of
private helpers and per-frame allocations were addressed; `println` stays for the debug log, as
TinyGo on the 3DS has no `log/slog` output.
