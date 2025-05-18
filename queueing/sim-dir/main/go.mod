module main

go 1.23.4

replace local/cntrl => ../cntrl

require (
	github.com/iti/pces v0.0.26
	local/cntrl v0.0.0-00010101000000-000000000000
)

require (
	github.com/iti/cmdline v0.1.2 // indirect
	github.com/iti/evt v0.1.6 // indirect
	github.com/iti/mrnes v0.0.25 // indirect
	github.com/iti/rngstream v0.2.2 // indirect
	golang.org/x/exp v0.0.0-20250506013437-ce4c2cf36ca6 // indirect
	gonum.org/v1/gonum v0.16.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
