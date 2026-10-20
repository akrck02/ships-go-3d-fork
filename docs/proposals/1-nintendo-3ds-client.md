# 1. A Nintendo 3DS client

> Status: done, pending review · Author's approval: proposal ☑ plan ☑

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

- [x] T1 — TCP sockets on the 3DS · go-3ds `network` package (released in v0.2.0) + `cmd/echo` · R1 · test: echo in Azahar
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
- [x] T16 — snapshots read by hand, allocating nothing (`protocol.SnapshotReader`); and in go-renderer (released in v0.2.0): `scene.Node.Moving` lights a node on the GPU instead of baking it at every turn, at most four bakes a mesh, no seabed without water. The client marks ships and the aim cross Moving and drops its workarounds (copies of meshes per node, the budget of bakes, the still ground). 57 fps with 5, 15, 30 and 60 bots in the emulator · tests: `TestTheReaderAgreesWithEncodingJson`, `TestReadingASnapshotAllocatesNothingOnceWarm`, `TestASceneWithoutWaterHasNoSeabed`, `TestTheLeastRecentlyDrawnVariantIsTheOneReplaced` (ctr, in Azahar)
- [x] T17 — a design at the server's limit (400 parts, most mirrored, round shapes) ran the console out of memory at the fine detail (about 200,000 triangles): each design now gets the finest detail that fits a budget of triangles, near and far (`ship.Within`, a third level `ship.Minimal`), its mesh made to measure; and a welcome carrying twenty such designs passed the websocket's 1 MB limit, now 8 MB. Twenty of them around the camera: 45 fps · tests: `TestTrianglesCountsWhatBuildMakes`, `TestAHeavyDesignGetsAPlainerDetail`
- [x] T18 — five more built-in ships on the server (Kestrel, Mule, Lancer, Bulwark, Wisp: 11 to 25 parts, designed as a player might), in the 3DS's list too, chosen with arrows since a drop-down of eight ran off the touch screen · tests: `TestEveryDefaultShipIsAValidDesign`, `TestTheDefaultShipsHaveTheirIds`
- [x] T19 — logging in (half of R8): email and password on the system keyboard (go-3ds `keyboard.Password`), `POST /login` over a small HTTP client (`client3ds/web`), the session's cookie on the websocket's handshake, and the player's own designs after the built-in ones (`client3ds/account`, `GET /game/ships`); log out. Tried in Azahar against the server: joined as the test user flying one of its own ships · tests: `TestALoginSetsItsCookie`, `TestALongAnswerInChunksIsJoined`, `TestASilentServerTimesOut`, `TestAPlayerLogsInAndSeesTheirShips`, `TestAWrongPasswordIsRefused`
- [x] T20 — the ship arrows offer what the chosen server has: three ships every version has until the server answers, its own list after typing its address, logging in or out, and checked again on Connect (a ship it lacks is refused with a message instead of an error from the server). Tried in Azahar against this branch's server and one built from `main`
- [x] T10 — TLS (the other half of R8): go-3ds `network.DialSecure`, TLS 1.2 through devkitPro's mbedtls 2.28, trusting Let's Encrypt's ISRG Root X1 and X2 and checking the server's name; a server typed as `https://name:port` is talked to over TLS for logging in, the ships and the websocket (`account.ParseAddress`). The real server, `https://jonbul.ddns.net:3000`, answers in about a second's handshake; a self-signed or misnamed certificate is refused. Found on the way: Close skipped freeing a connection the server had closed first, and the third TLS handshake failed to allocate · tests: `TestAddressesAreUnderstood`; `cmd/secure-echo` in Azahar against the real server and badssl.com · decided then · R8 · test: join against the HTTPS dev server
- [x] T11 — check the server is untouched · R9 · test: `go test ./...`

- [x] T21 — a 3D radar on the touch screen, as the browser's: a dish seen from above and behind, blips with stalks to its plane, the sweep turning in it, drawn by a second ctr.SceneRenderer with the readings painted over it (go-3ds `bottom.PaintOver`); the flat one stays in Settings. 60 fps with 30 bots in Azahar · test: `TestABlipOnTheRightIsOnMinusX`
- [x] T22 — space as the browser's: a near-black background, 1,500 stars and its three planets on a sky that follows the camera, the world's edge as a faint grid (a subdivided icosahedron), the browser's light (hemisphere sky and ground, warm sun), and dust in cells of 300 that follow the ship, so it no longer runs out away from the centre. 60 fps with 30 bots in Azahar
- [x] T23 — quality of life: the title enters the remembered server at once (the real one by default) with the remembered session (go-3ds `storage`, `client3ds/profile`, `account.Who`), then a ship screen (the ship turning on the top screen, its name, arrows, Fly), then the game; the server and the account live in the options' Account tab and warn that the game will restart; ZL/ZR slow down and speed up; the radar says its zoom instead of a distance with no unit; the gyroscope's rates were thousands of times too large (multiplied by libctru's coefficient instead of divided) · tests: `TestAProfileComesBackAsItWasSaved`, `TestABrokenFileStartsAfresh`, `TestARememberedSessionIsCheckedWithTheServer`; `cmd/storage-check` in Azahar
- [x] T24 — pixel art in the colours of space (OFL pixel fonts, square corners, hard shadows), the ship screen's turning ship on the touch screen seen from above, the gyroscope's axes swapped (reported on the console), the socket service started first with Go's heap capped at 64 MB; and running into the world's edge: the grid glows around the touch and fades, sparks fly, the camera shakes, with the edge's lines a pixel wide wherever the camera is · tests: `TestRunningIntoTheEdgeIsNotedOnce`, `TestTheGlowIsBrightestAtTheHitAndGoneAtItsReach`, `TestTheShakeDiesAway`
- [x] T25 — sound through go-sound (released in v0.1.1, the console's DSP): chiptune effects and a looping track worked out in code (`client3ds/chiptune`): shots, hits, blasts, the force field, the menus; volumes for effects and music in Settings, remembered · tests: `TestEverySoundStaysWithinRangeAndEnds`, `TestTheMusicNeverEndsAndStaysWithinRange`, `TestTheWaveSpreadsFastThenSlows`

## Result

Tried out as prereleases `demo-1` … `demo-36`, built from `feature/3ds-client`, in the Azahar
emulator against a local server with bots, and on a New 3DS XL against the real server
(`https://jonbul.ddns.net:3000`, over TLS) with test accounts.

What the client does: the title, then a hangar where your ships (built-in and your own designs)
turn on a platform; log in and pick the server from the menu, remembered on the SD card; fly with
the circle pad, the C-stick, the buttons or the gyroscope; fire, take hits, die and fly again; see
the others in stereoscopic 3D with a 3D radar on the touch screen; chiptune sound and an engine
that follows the speed; at the world's edge a force field, a camera that draws back and a copilot
who turns you back.

Changed from the plan: sockets, TLS, the SD card, the keyboard, START handling, the C-stick and
the gyroscope live in Go for 3DS (`network`, `storage`, `keyboard`, `app`), released in go-3ds
v0.2.0. go-renderer's `ctr` kept one baked copy of a mesh per orientation, forever, so turning
ships filled the GPU memory; moving nodes are now lit on the GPU instead (go-renderer v0.2.0).
Go Sound plays on the console's DSP (go-sound v0.1.1). `client3ds/go.mod` asks for those
released versions, with no `replace`.

Measured in the emulator with up to 60 bots (T14): the server handles 60 without errors (4,400
messages and 12.7 MB a second); the client draws 14 frames a second there, 25 with 30 bots and 56
with 5. On the console, with about a dozen players, it holds about 60. The menu's STATS page
shows the numbers.

Pending: a person's review of all of it, and a try on an older (non-New) 3DS, which has less
memory and a slower processor. `println` stays for the debug log, as TinyGo on the 3DS has no
`log/slog` output.
