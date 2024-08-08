package main

import (
	"fmt"
	"github.com/iti/cmdline"
	"github.com/iti/mrnes"
	"github.com/iti/pces"
	"github.com/iti/rngstream"
	"golang.org/x/exp/slices"
	"path/filepath"
)

// cmdlineParams defines the parameters recognized
// on the command line
func cmdlineParams() *cmdline.CmdParser {
	// command line parameters are all about file and directory locations.
	// Even though we don't need the flags for the other pces structures we
	// keep them here so that all the programs that build templates can use the same arguments file
	// create an argument parser
	cp := cmdline.NewCmdParser()
	cp.AddFlag(cmdline.StringFlag, "inputLib", true) // directory where model parameters are read from
	cp.AddFlag(cmdline.StringFlag, "cp", true)       //
	cp.AddFlag(cmdline.StringFlag, "cpInit", true)   //
	cp.AddFlag(cmdline.StringFlag, "funcExec", true) // name of input file holding descriptions of functional timings
	cp.AddFlag(cmdline.StringFlag, "devExec", true)  // name of input file holding descriptions of device timings
	cp.AddFlag(cmdline.StringFlag, "srdCfg", true)   // name of input file holding descriptions of functions that share configuration
	cp.AddFlag(cmdline.StringFlag, "map", true)      // file with mapping of comp pattern functions to hosts
	cp.AddFlag(cmdline.StringFlag, "exp", true)      // name of file used for run-time experiment parameters
	cp.AddFlag(cmdline.StringFlag, "mdfy", false)    // name of file used to modify exp experiment parameters
	cp.AddFlag(cmdline.StringFlag, "topo", false)    // name of output file used for topo templates
	cp.AddFlag(cmdline.StringFlag, "trace", false)   // path to output file of trace records
	cp.AddFlag(cmdline.BoolFlag, "qnetsim", false)   // flag indicating that network sim ought to be 'quick'
	cp.AddFlag(cmdline.FloatFlag, "stop", true)      // run the simulation until this time (in seconds)

	return cp
}

// main gives the entry point
func main() {
	// define the command line parameters
	cp := cmdlineParams()

	// parse the command line
	cp.Parse()

	// string for the input directory
	inputDir := cp.GetVar("inputLib").(string)

	// make sure these directories exist
	dirs := []string{inputDir}
	valid, err := pces.CheckDirectories(dirs)
	if !valid {
		panic(err)
	}

	// check for access to input files
	fullpathmap := make(map[string]string)
	inFiles := []string{"cp", "cpInit", "funcExec", "devExec", "srdCfg", "exp", "mdfy", "topo", "map"}
	optionalFiles := []string{"mdfy", "srdCfg"}

	fullpath := []string{}
	syn := make(map[string]string)
	errs := []error{}
	for _, filename := range inFiles {
		if !cp.IsLoaded(filename) {
			if !slices.Contains(optionalFiles, filename) {
				errs = append(errs, fmt.Errorf("command flag %s not included on the command line", filename))
			}
			continue
		}
		basefile := cp.GetVar(filename).(string)
		fullfile := filepath.Join(inputDir, basefile)
		fullpath = append(fullpath, fullfile)

		fullpathmap[filename] = fullfile
		syn[filename] = fullfile
	}

	err = pces.ReportErrs(errs)
	if err != nil {
		panic(err)
	}

	// validate that these are all readable
	ok, err := pces.CheckReadableFiles(fullpath)
	if !ok {
		panic(err)
	}

	// if we're saving traces check the path
	var traceFile string
	useTrace := false
	if cp.IsLoaded("trace") {
		traceFile = cp.GetVar("trace").(string)
		_, err := pces.CheckOutputFiles([]string{traceFile})
		if err != nil {
			panic(err)
		}
		useTrace = true
	}

	// if -qnetsim is set we use the 'skip over network devices' version of network simulation
	if cp.IsLoaded("qnetsim") {
		syn["qksim"] = "true"
	}

	traceMgr := mrnes.CreateTraceManager("experiment", useTrace)

	// if requested, set the rng seed
	if cp.IsLoaded("rngseed") {
		seed := cp.GetVar("rngseed").(int64)
		rngstream.SetRngStreamMasterSeed(uint64(seed))
	}

	// build the experiment.  First the network stuff
	// start the id counter at 1 (value passed is incremented before use)
	mrnes.BuildExperimentNet(syn, true, 0, traceMgr)

	// now the computation patterns, where initial events were scheduled
	evtMgr, err := pces.BuildExperimentCP(syn, true, mrnes.NumIDs, traceMgr)
	if err != nil {
		panic(err)
	}

	termination := cp.GetVar("stop").(float64)
	evtMgr.Run(termination)

	if useTrace {
		traceMgr.WriteToFile(traceFile)
	}

	pces.ReportStatistics()
	fmt.Println("Done")
}
