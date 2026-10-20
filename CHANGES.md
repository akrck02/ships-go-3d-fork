CHANGES
=======
Version 0.6.0 - 2026-10-19
------------------
A new game mode, the assault, on its own websocket: `/ws/assault`. The free
for all at `/ws` is unchanged.

- Two teams and a mothership at the centre of the world. Attackers fly their
  ships and try to destroy it; gunners sit in turrets on its hull and shoot
  the attackers down. An attacker's ship takes 3 hits, and respawns after the
  usual delay; the mothership takes 100. Attackers can only hit the
  mothership and gunners only attackers: there is no friendly fire.
- The attackers win by destroying the mothership. The gunners win if it is
  still flying after 5 minutes, or the attackers all leave (and the other
  way round).
- Teams are dealt at random when a round starts: one gunner for about three
  attackers (4 players: 1 gunner, 20 players: 5), and always at least one.
  The turrets are spread around the ship, alternating between its top and
  its underside.
- Players wait in a lobby. It starts its round 30 seconds after the last
  player joined, once there are at least 4; with 20 it starts at once. A new
  lobby then opens for the next players. When a round ends, its result is
  shown for 10 seconds and then all its players go back to the open lobby
  together.
- Hits on the mothership are checked against its actual shape (every part's
  box), not a sphere: it never moves, so the server knows exactly where it is.
- The rules (players, waits, lives, reloads, round length) are in
  `AssaultRules`, sent to the players with each round.
- `/status` also counts the players in the assault mode.

Crashes, in both modes: a new client message, `crash {with}`, reports that
our ship ran into ship `with` (or `"mothership"` in the assault). The server
destroys both ships if its own record of their positions puts their hit
spheres within reach of each other (allowing for latency, as for hits), or
the attacker if it is at the mothership's hull. Gunners never crash, the
mothership takes no damage from one, and nobody gets a kill: `died` comes
with an empty `from`.

Version 0.5.0 - 2026-10-19
------------------
Messages for players in Spanish and Basque (Euskara), as well as English.

- Error messages from the API (login, registration, saving a ship...) are in
  the language of the request's Accept-Language header, which ships-vue-3d
  sets to the language picked on the site. Anything other than `es` or `eu`
  gets English, as before.
- The game websocket takes the language as a `lang` query parameter
  (`/ws?lang=eu`), since browsers can't set headers there. It applies to the
  game's error messages and to the default name for guests.
- CORS allows the Accept-Language header.
- The texture and image link errors from 0.4.0 are translated too.

Version 0.4.0 - 2026-10-19
------------------
Ship parts can have a texture: `texture` on a part is one of `panels`,
`grid`, `stripes`, `carbon`, `hexagons`, `worn`, `camo` or `windows`, or is
left out for a plain part. It is saved with the ship and sent to every player
in game; ships saved before stay plain.

A part can instead have `textureUrl`, an image link used as its texture. It
must be https, with a host, no spaces or control characters and at most 500
characters, and a part can't have both `texture` and `textureUrl`.

Version 0.3.0 - 2026-10-18
------------------
Six new part types for the ship designer: dome, capsule, pyramid, tube,
prism (hexagonal) and frustum (tapered cylinder). Ships using them can now be
saved and flown.

Version 0.2.0 - 2026-10-10
------------------
Players on mobile data are no longer thrown out of the game after a few
seconds.

Bugfixes
- Flood protection is now a token bucket (60 messages a second sustained,
  bursts of up to 400) instead of a hard 120 per second. A mobile connection
  that stalls delivers everything the browser queued meanwhile in one go,
  and that catch-up burst was being treated as a flood. A client that is cut
  off is now told why before its connection closes.

Version 0.1.0 - 2026-10-09
------------------
First version: the backend of Ships 3D, a 3D take on the Ships game where
players design ships from 3D primitives and fly them in open space.

- Users and sessions are shared with the 2D game (`users` and `sessions`
  collections, same `token` cookie), so one account and one login work in
  both. Registration, login (with "remember me"), logout and password change.
- 3D projects in their own `paintingProjects3d` collection, with a REST API
  (`/projects`) that only ever lets a user see or change their own projects,
  and server-side validation of every design.
- Real-time game over `/ws`: client-authoritative flight, server-authoritative
  life, kills, deaths and respawns. Reported hits are checked against the
  server's record of the bullet and the target before any damage applies.
- Built-in ships (Arrow, Hammer, Saucer) for guests and new players.
