package cntrl

import (
	"github.com/iti/pces"
	"github.com/iti/mrnes"
	"github.com/iti/evt/evtm"
	"github.com/iti/evt/vrtime"
	"local/sjf"
	"strconv"
	"path/filepath"
	"math/rand"
)

var seed int64 = 1324356

func interArrival(lambda float64) float64 {
	return rand.ExpFloat64()/lambda
}

var SaveLambda map[int]float64

// user extension of core methods for srvRsp and finish function classes
func extendSetup() {
    // visit every CmpPtn
    for _, cpi := range pces.CmpPtnInstByID {
        // find list of functions in group "SJF", if any
        funcLists, present := cpi.FuncsByGroup["SJF"]
        if present {
            // visit every function in that group, in this CmpPtn
            for _, cpfi := range funcLists {
                // we need a function of the srvRsp class
                if cpfi.Class != "srvRsp" {
                    continue
                }   
                // found one
                cpfi.AddStartMethod("sjf", sjf.SrvRspSJF)
            }   
        }   
        funcLists, present = cpi.FuncsByGroup["EndMsr"] 
		if present {
			// there is only one function in this group
			for _, cpfi := range funcLists {
				if cpfi.Class == "measure" {
					msrState := cpfi.State.(*pces.MeasureState)
					msrState.Classify = CPClassify
				}
			}
		}
	}   
}

// CPClassify returns the CmpPtn name of execution thread source
func CPClassify(ID int, visited []int) string {
	return pces.CmpPtnFuncInstByID[ visited[0] ].PtnName
}


// ExpCntrl is called to initialize the events for the simulation
func ExpCntrl(evtMgr *evtm.EventManager, context any, data any) any {
	rand.Seed(pces.GlobalSeed)

	extendSetup()
	SaveLambda = make(map[int]float64)

    // visit every CmpPtn
    for _, cpi := range pces.CmpPtnInstByID {
        for _, cpfi := range cpi.Funcs {
            if cpfi.Class == "start" {
                // get inter-arrival rate lambda 
                srtcfg := cpfi.Cfg.(*pces.StartCfg)
                param := srtcfg.Data
                lambda, err := strconv.ParseFloat(param, 64)
                if err != nil || !(lambda > 0.0) {
                    panic("expect lambda to be positive float")
                }
				SaveLambda[cpfi.ID] = lambda
				arrival := interArrival(SaveLambda[cpfi.ID])
                evtMgr.Schedule(cpfi, nil, startEnterCycle, 
					vrtime.SecondsToTime(arrival))
            }
        }
    }
    return nil
}


// startEnterCycle launches a new execution thread
func startEnterCycle(evtMgr *evtm.EventManager, context any, data any) any {
	cpfi := context.(*pces.CmpPtnFuncInst)
	srts := cpfi.State.(*pces.StartState)
    cpm := new(pces.CmpPtnMsg)

    pces.NumExecThreads += 1
    cpm.ExecID = pces.NumExecThreads

    cpm.PcktLen = srts.PcktLen
    cpm.MsgLen = srts.MsgLen

	srts.Calls   += 1

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
	arrivalTime := interArrival(SaveLambda[cpfi.ID])
	evtMgr.Schedule(cpfi, nil, startEnterCycle, vrtime.SecondsToTime(arrivalTime))
	return nil
}


// ExpCmplt is called at the end of a simulation run, to aggregate statistics
// on the measurements
func ExpCmplt(evtMgr *evtm.EventManager, context any, data any) any {
	csvFileName := *context.(*string)
	exprmntName := *data.(*string)

	// create a csv header
	empty := []string{}

	
	// visit every group and build csv representation of its data	
	for _, msrg := range pces.MsrGrpByID {
		msrg.PrepCSVRow(csvFileName, pces.ExprmntsFile,
			exprmntName, empty, empty, "Mean") 
		msrg.AddCSVData(csvFileName, exprmntName, empty)
	}

	// put the trace file in the same directory as the msrFile
	tdirectory, file := filepath.Split(csvFileName)
	file = "trace.yaml"
	traceFile := filepath.Join(tdirectory, file)
	pces.TraceMgr.WriteToFile(traceFile, false)
	return nil 
}
