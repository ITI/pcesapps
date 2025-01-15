package cntrl

import (
	"github.com/iti/pces"
	"github.com/iti/mrnes"
	"github.com/iti/evt/evtm"
	"github.com/iti/evt/vrtime"
	"strconv"
	"path/filepath"
	"math/rand"
	"sort"
)

var seed int64 = 1324356

func interArrival(lambda float64) float64 {
	return rand.ExpFloat64()/lambda
}

var SaveLambda float64

var eudCPID []int
var tgtCP []int 
var eudPr map[int]float64

// ExpCntrl is called to initialize the events for the simulation
func ExpCntrl(evtMgr *evtm.EventManager, context any, data any) any {
	rand.Seed(seed)

	// get the lambda parameter
	cpi  := pces.CmpPtnInstByName["HMI"]
	cpfi := cpi.Funcs["startThread"]

	// check validity
	srtcfg := cpfi.Cfg.(*pces.StartCfg)
	lambda, err := strconv.ParseFloat(srtcfg.Data, 64)
	if err != nil || !(lambda > 0.0) {
		panic("expect lambda to be positive float")
	}

	// kick off the first initiation, after one inter-initiation time
	arrivalTime := interArrival(lambda)
	SaveLambda = lambda
	// schedule the first start
	evtMgr.Schedule(cpfi, nil, startEnterCycle, 
		vrtime.SecondsToTime(arrivalTime))

	// initialize data structures focus on EUD CmpPtns	
	eudCPID = make([]int,4)
	eudCPID[0] = pces.CmpPtnInstByName["EUD-0"].ID
	eudCPID[1] = pces.CmpPtnInstByName["EUD-1"].ID
	eudCPID[2] = pces.CmpPtnInstByName["EUD-2"].ID
	eudCPID[3] = pces.CmpPtnInstByName["EUD-2"].ID

	// order the list for repeatability
	sort.Ints(eudCPID)

	// compute demonstration probabilties of EUD target
	eudPr = make(map[int]float64)
	eudPr[eudCPID[0]] = 0.1
	eudPr[eudCPID[1]] = 0.3
	eudPr[eudCPID[2]] = 0.6
	eudPr[eudCPID[3]] = 1.0

	// put in a custom path classifier	
	cpfi = cpi.Funcs["endMeasure"]
	state := cpfi.State.(*pces.MeasureState)
	state.Classify = eudClassify

	// make a map of cpfi.ID to cpfi.PtnName
	fID2CP = make(map[int]string)

	cpi    = pces.CmpPtnInstByName["EUD-0"]
	cpfi   = cpi.Funcs["eudProcess"]
	fID2CP[cpfi.ID] = "EUD-0"

	cpi    = pces.CmpPtnInstByName["EUD-1"]
	cpfi   = cpi.Funcs["eudProcess"]
	fID2CP[cpfi.ID] = "EUD-1"

	cpi    = pces.CmpPtnInstByName["EUD-2"]
	cpfi   = cpi.Funcs["eudProcess"]
	fID2CP[cpfi.ID] = "EUD-2"

	cpi    = pces.CmpPtnInstByName["EUD-3"]
	cpfi   = cpi.Funcs["eudProcess"]
	fID2CP[cpfi.ID] = "EUD-3"

	return nil 
}

var fID2CP map[int]string
func eudClassify(visited []int) string {
	for _, devID := range visited {
		name, present := fID2CP[devID]
		if present {
			return name
		}
	}
	return "Default"	
}

var numExecThreads int = 1

func startEnterCycle(evtMgr *evtm.EventManager, context any, data any) any {
	cpfi := context.(*pces.CmpPtnFuncInst)
	srts := cpfi.State.(*pces.StartState)
    cpm := new(pces.CmpPtnMsg)

    pces.NumExecThreads += 1
    cpm.ExecID = pces.NumExecThreads

    cpm.PcktLen = srts.PcktLen
    cpm.MsgLen = srts.MsgLen

	// randomly select which next EUD to generate an arrival for
	u := rand.Float64()
	cpid := 0
	for idx:=0; idx<len(eudCPID); idx++ {
		if u < eudPr[eudCPID[idx]] {
			cpid = idx
			break
		}
	}		

	// get the CPID of the next EUD to target
	cpm.XCPID    = eudCPID[cpid]

	srts.Calls   += 1
	cpm.XLabel   = "decryptPckt" 
	cpm.XMsgType = "decrypt"

    endptName := cpfi.Host
    endpt := mrnes.EndptDevByName[endptName]
    pces.AddCPTrace(pces.TraceMgr, cpfi.Trace, evtMgr.CurrentTime(), 
		cpm.ExecID, endpt.DevID(), 
			pces.FullFuncName(cpfi,"startEnterCycle"), cpm) 

    // out edge destination a function of the message type
    cpm = pces.AdvanceMsg(cpfi, cpm, srts.MsgType)

    cpfi.AddResponse(cpm.ExecID, []*pces.CmpPtnMsg{cpm})
    evtMgr.Schedule(cpfi, cpm, pces.ExitFunc, vrtime.SecondsToTime(0.0))

	// schedule the next invocation of startEnterCycle after arrivalTime amount of time
	if srts.Calls < pces.Samples {
		arrivalTime := interArrival(SaveLambda)
		evtMgr.Schedule(cpfi, nil, startEnterCycle, vrtime.SecondsToTime(arrivalTime))
	}
	return nil
}


// ExpCmplt is called at the end of a simulation run, to aggregate statistics
// on the measurements
func ExpCmplt(evtMgr *evtm.EventManager, context any, data any) any {
	csvFileName := *context.(*string)
	exprmntName := *data.(*string)

	// create a csv header
	hdr := []string{"EUD"}
	empty := []string{}

	// cycle through groups
	for _, msrg := range pces.MsrGrpByID {
		msrg.PrepCSVRow(csvFileName, pces.ExprmntsFile,
		      exprmntName, hdr, empty, "Range") 

		data := []string{msrg.GroupDesc}
		msrg.AddCSVData(csvFileName, exprmntName, data)
	}

	// put the trace file in the same directory as the msrFile
	tdirectory, _ := filepath.Split(csvFileName)
	file := "trace.yaml"
	traceFile := filepath.Join(tdirectory, file)
	pces.TraceMgr.WriteToFile(traceFile, false)
	return nil 
}

