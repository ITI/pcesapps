package main
import (
	"github.com/iti/pces"
	"local/cntrl"
)

func main() {
	pces.ReadSimArgs() 
	pces.RunExperiment(cntrl.ExpCntrl, cntrl.ExpCmplt) 
}

