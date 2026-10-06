module example.com/external-edition

go 1.27.0

require (
	github.com/a-h/templ v0.3.1020
	github.com/ardaozyildirim/hydrogo v0.0.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/redis/go-redis/v9 v9.22.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)

replace github.com/ardaozyildirim/hydrogo => ../..
