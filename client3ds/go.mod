module ships3d/client3ds

go 1.25.6

require (
	github.com/akrck02/go-3ds v0.0.0-00010101000000-000000000000
	github.com/akrck02/go-renderer v0.0.0
	github.com/akrck02/go-ui v0.0.0-20261018091606-e6c4eac3b690
	github.com/gorilla/websocket v1.5.3
)

require (
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

// go-3ds is built from a local copy, with its network package (branch feature/network).
replace github.com/akrck02/go-3ds => ../../go-3ds

// As go-3ds: go-renderer's feature/3ds, which has ctr.
replace github.com/akrck02/go-renderer => github.com/akrck02/go-renderer v0.0.0-20261017105741-76ca23f13554
