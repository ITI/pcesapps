module bld-dir

go 1.21

toolchain go1.22.7

replace github.com/iti/probe => ../probe
replace github.com/iti/pces => ../../../pces
replace github.com/iti/mrnes => ../../../mrnes
replace github.com/iti/cmdline => ../../../cmdline

require (
	github.com/iti/cmdline v0.0.0-00010101000000-000000000000
	github.com/iti/mrnes v0.0.4
	github.com/iti/pces v0.0.5
	github.com/iti/probe v0.0.0-00010101000000-000000000000
)

require (
	github.com/iti/evt/evtm v0.1.4 // indirect
	github.com/iti/evt/evtq v0.1.4 // indirect
	github.com/iti/evt/vrtime v0.1.5 // indirect
	github.com/iti/rngstream v0.2.2 // indirect
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56 // indirect
	gonum.org/v1/gonum v0.15.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
