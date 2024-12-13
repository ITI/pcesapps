package main

import (
	"fmt"
	"github.com/iti/pces"
)

// main gives the entry point
func main() {
	pces.ReadSimArgs() 
	pces.RunExperiment(expCntrl, expCmplt) 
	fmt.Println("Done")
}


