package cntrl
import (
	"fmt"
	"os"
	"github.com/iti/cmdline"
	"github.com/iti/pces"
)

// ReadUserArgs is called to pull off arguments from a user-defined command arguments list
func ReadUserArgs(check bool) {
	// return if either there is only one "-is" on the command-line, or
	// if the file referenced as the user command file is not present
	isFlags	:= 0
	for idx:=1; idx< len(os.Args); idx++ {
		if os.Args[idx] == "-is" {
			isFlags += 1
		}
	}

	if isFlags < 2 {
		if check {
			panic("expecting -is user-args on the command line")	
		} else {
			return
		}
	}

	userCmdFile := os.Args[2]
	_, err := os.Stat(userCmdFile)
	if err != nil {
		panic(fmt.Errorf("unable to open user cmd file %s", userCmdFile))	
	}	 

	cp := cmdline.NewCmdParser()
	cp.AddFlag(cmdline.IntFlag,    "samples", true) // nm
	cp.AddFlag(cmdline.IntFlag,    "batch", true) // nm
	cp.AddFlag(cmdline.IntFlag,    "skip", true) // nm

	cp.Parse()

	pces.Samples = cp.GetVar("samples").(int)
	pces.Batch = cp.GetVar("batch").(int)
	pces.Skip = cp.GetVar("skip").(int)

	// sanity check
	numBatches := pces.Samples/pces.Batch

	if pces.Skip > numBatches/2 {
		panic("Skipping more than 50% of the batches")
	}

	os.Args = append(os.Args[0:1], os.Args[3:]...)
}

