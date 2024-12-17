package main
import (
	"fmt"
	"github.com/iti/pces"
)

func main() {
	pces.ReadSimArgs() 
	pces.RunExperiment(expCntrl, expCmplt) 
	fmt.Println("Done")
}


