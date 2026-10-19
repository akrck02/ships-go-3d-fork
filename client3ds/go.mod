module ships3d/client3ds

go 1.25.6

require (
	github.com/akrck02/go-ui v0.0.0-20261018091606-e6c4eac3b690
	github.com/gorilla/websocket v1.5.3
)

require (
	github.com/akrck02/go-3ds v0.0.0-20261019190756-d31b93d7aa49
	github.com/akrck02/go-renderer v0.0.0
	github.com/akrck02/go-sound v0.0.0-20261019223914-4bc0be972c29
	github.com/ebitengine/oto/v3 v3.5.1 // indirect
	github.com/ebitengine/purego v0.11.0 // indirect
	github.com/jfreymuth/pulse v0.1.3 // indirect
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)

// go-renderer comes from its branch feature/3ds-moving-light, which lights moving nodes on the
// GPU in ctr; go-3ds asks for it as v0.0.0, so it is replaced here too.
replace github.com/akrck02/go-renderer => github.com/akrck02/go-renderer v0.0.0-20261019061605-5d81abaaf2de
