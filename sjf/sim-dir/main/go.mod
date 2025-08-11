module main

go 1.24.5

replace local/cntrl => ../cntrl

replace local/sjf => ../sjf

require (
	github.com/iti/pces v0.1.2
	local/cntrl v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/gopacket v1.1.19 // indirect
	github.com/iti/cmdline v0.1.2 // indirect
	github.com/iti/evt v0.1.7 // indirect
	github.com/iti/mrnes v0.1.3 // indirect
	github.com/iti/rngstream v0.2.2 // indirect
	golang.org/x/exp v0.0.0-20250718183923-645b1fa84792 // indirect
	golang.org/x/sys v0.0.0-20190412213103-97732733099d // indirect
	gonum.org/v1/gonum v0.16.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	local/sjf v0.0.0-00010101000000-000000000000 // indirect
)
