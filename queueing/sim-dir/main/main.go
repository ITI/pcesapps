package main
import (
	"github.com/iti/pces"
	"local/cntrl"
)

func main() {
	cntrl.ReadUserArgs(true)
	pces.ReadSimArgs() 
	pces.RunExperiment(cntrl.ExpCntrl, cntrl.ExpCmplt) 
}

