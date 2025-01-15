package cntrl

import (
	"github.com/iti/pces"
	"github.com/iti/evt/evtm"
	"github.com/iti/evt/vrtime"
	"strconv"
	"path/filepath"
	"math/rand"
)

var seed int64 = 1324356

func interArrival(lambda float64) float64 {
	return rand.ExpFloat64()/lambda
}

var saveLambda float64

// ExpCntrl is called to initialize the events for the simulation
func ExpCntrl(evtMgr *evtm.EventManager, context any, data any) any {
	rand.Seed(seed)

	// get the lambda parameter
	cpi  := pces.CmpPtnInstByName["HMI"]
	cpfi := cpi.Funcs["startThread"]

	srtcfg := cpfi.Cfg.(*pces.StartCfg)
	lambda, err := strconv.ParseFloat(srtcfg.Data, 64)
	if err != nil {
		panic(err)
	}

	if !(lambda > 0.0) {
		panic("expect lambda to be positive float")
	}

	saveLambda = lambda

	arrivalTime := interArrival(lambda)
	for idx:=0; idx<pces.Samples; idx++ {
		evtMgr.Schedule(cpfi, &cpfi.Class, pces.EnterFunc, vrtime.SecondsToTime(arrivalTime))
		arrivalTime += interArrival(lambda)
	}
	return nil 
}

// ExpCmplt is called at the end of a simulation run, to aggregate statistics
// on the measurements
func ExpCmplt(evtMgr *evtm.EventManager, context any, data any) any {
	csvFileName := *context.(*string)
	exprmntName := *data.(*string)

	// create a csv header
	empty := []string{}

	// get the lone group and build the csv row
	for _, msrg := range pces.MsrGrpByID {
		msrg.PrepCSVRow(csvFileName, pces.ExprmntsFile,
			exprmntName, empty, empty, "CI") 
		msrg.AddCSVData(csvFileName, exprmntName, empty)
		break
	}

	// put the trace file in the same directory as the msrFile
	tdirectory, file := filepath.Split(csvFileName)
	file = "trace.yaml"
	traceFile := filepath.Join(tdirectory, file)
	pces.TraceMgr.WriteToFile(traceFile, false)
	return nil 
}

