package main
import (
	"github.com/iti/pces"
)

func main() {
	pces.ReadSimArgs() 
	pces.RunExperiment(ExpCntrl, ExpCmplt) 
}


