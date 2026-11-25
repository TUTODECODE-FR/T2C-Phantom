module t2c

go 1.26.11

require (
	github.com/flynn/noise v1.1.0
	github.com/gorilla/websocket v1.5.4-0.20240701034025-d67f41855da4
	github.com/prometheus/client_golang v1.25.0
	golang.org/x/crypto v0.59.0
	golang.org/x/sys v0.50.0
)

require golang.org/x/net v0.62.0 // indirect

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.3 // indirect
	github.com/prometheus/common v0.72.0 // indirect
	github.com/prometheus/procfs v0.22.0 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	go.yaml.in/yaml/v2 v2.4.4 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
	t2c-relay v0.0.0
)

replace t2c-relay => ../T2C-Phantom-relay
