package probe

import (
	"sort"
	"github.com/iti/evt/evtm"
	"github.com/iti/evt/vrtime"
	"github.com/iti/mrnes"
	"gopkg.in/yaml.v3"
	"encoding/json"
	"os"
)

// ProbeExp is a global list that holds the identity of the endpoint pairs to work through.
// Note that the names are of the devices being probed, not the probe functions on them
var ProbeExp []ProbePair = make([]ProbePair,0)

// ProbePair specifies a source and destination endpoint for a probe.
// Need a structure pair to list them together to define an experiment
type ProbePair struct {
	SrcDev string
	DstDev string
}

type ExpPairs struct {
	Probes	[]ProbePair		`json:"probes"	yaml:"probes"`
}

var experiment ExpPairs

func ProbeEndpts(probeFile string, useYAML bool) map[string]bool {
	ReadExpPairs(probeFile, useYAML)
	endptMap := make(map[string]bool)	
	for _, probe := range experiment.Probes {
		endptMap[probe.SrcDev] = true
		endptMap[probe.DstDev] = true
	}
	return endptMap
}

func BuildProbeExp(probeFile string, useYAML bool) {
	experiment.Probes = make([]ProbePair,0) 

	endpts := make(map[string]bool)
	for endptName, _ := range mrnes.EndptDevByName {
		endpts[endptName] = true	
	}

	if len(probeFile) > 0 {

		// read in the file
		ReadExpPairs(probeFile, useYAML)
		for _, probe :=  range experiment.Probes {
			_,  present0 := endpts[probe.SrcDev]
			_,  present1 := endpts[probe.DstDev]
			if !present0 || !present1 {
				panic("experiment pairs input file contains reference to non endpoint")
			}
		}

		// copy to ProbeExp
		for _, probe := range experiment.Probes {
			ProbeExp = append(ProbeExp, probe)
		}
		// we're done
		return
	}

	// no input file provided
	BuildAllEndPtsProbeExp()
}

// BuildAllEndPtsProbeExp is called by the experiment control to
// creates a CP probe pairs list comparing every pair of endpoints in the model.
func BuildAllEndPtsProbeExp() {
	var endpts sort.StringSlice

	// create the list of comp pattern endpoints
	for endptname := range mrnes.EndptDevByName {
		endpts = append(endpts, endptname)
	}	

	endpts.Sort()	

	// build the ProbeExp list
	for idx:=0; idx< len(endpts); idx++ {
		for jdx:=0; jdx<len(endpts); jdx++ {
			if idx==jdx {
				continue
			}
			ProbeExp = append(ProbeExp, ProbePair{SrcDev: endpts[idx], DstDev: endpts[jdx]})
		}
	}
}

// StartProbeExp is called by the experiment control to launch the experiment
func StartProbeExp(evtMgr *evtm.EventManager) {
	evtMgr.Schedule("start", nil, ProbeControl, vrtime.SecondsToTime(0.0))
}

// SaveProbeResults is called by the experiment control to save the measurements
// into the file whose name is passed
func SaveProbeResults(msrFileName string, useYAML bool) {

	// simulation is done, write out the Measurements
	outputStr, _ := Measured.Serialize(useYAML)

	f, cerr := os.Create(msrFileName)
	if cerr != nil {
		panic(cerr)
	}
	_, werr := f.WriteString(outputStr)
	if werr != nil {
		panic(werr)
	}
	f.Close()
}

func ReadExpPairs(filename string, useYAML bool) {
	var err error
	var dict []byte

	dict, err = os.ReadFile(filename)
	if err != nil {
		panic("error reading experiment pairs file")	
	}

	// Select whether we read in json or yaml
	if useYAML {
		err = yaml.Unmarshal(dict, &experiment)
	} else {
		err = json.Unmarshal(dict, &experiment)
	}

	if err != nil {
		panic("error reading experiment pairs")
	}
}


