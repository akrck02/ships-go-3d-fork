CHANGES
=======
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
