module main

go 1.23.4

replace local/cntrl => ../cntrl

require (
	github.com/iti/pces v0.0.21
	local/cntrl v0.0.0-00010101000000-000000000000
)

require (
	github.com/iti/cmdline v0.1.2 // indirect
	github.com/iti/evt/evtm v0.1.4 // indirect
	github.com/iti/evt/evtq v0.1.4 // indirect
	github.com/iti/evt/vrtime v0.1.5 // indirect
	github.com/iti/mrnes v0.0.20 // indirect
	github.com/iti/rngstream v0.2.2 // indirect
	golang.org/x/exp v0.0.0-20250106191152-7588d65b2ba8 // indirect
	gonum.org/v1/gonum v0.15.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
