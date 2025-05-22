package main

import (
	"fmt"
	"github.com/iti/pces"
	"github.com/iti/mrnes"
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

var SaveLambda map[int]float64

// tag the MetaData in msg with 'client'
func outboundClientFunc(dev mrnes.TopoDev, metaKey string, msg *mrnes.NetworkMsg) float64 {
	msg.MetaData["source"] = true
	return mrnes.DelayThruDevice(dev.DevModel(), mrnes.DefaultSwitchOp, msg.MsgLen)
}

// tag the MetaData in msg with 'WE_src'
func outboundWEFunc(dev mrnes.TopoDev, metaKey string, msg *mrnes.NetworkMsg) float64 {
	msg.MetaData["WE_src"] = true
	return mrnes.DelayThruDevice(dev.DevModel(), mrnes.DefaultSwitchOp, msg.MsgLen)
}

func checkSrcFunc(dev mrnes.TopoDev, metaKey string, msg *mrnes.NetworkMsg) float64 {
	// complain if the msg does not have 'source' meta data
	_, presents := msg.MetaData["source"]
	_, presentf := msg.MetaData["WE_src"]

	if !presents && !presentf {
		fmt.Printf("unexpected message seen at %s\n", dev.DevName())	
	}

	if presents {
		delete(msg.MetaData,"source")
	}

	if presentf {
		delete(msg.MetaData,"WE_src")
	}
	return mrnes.DelayThruDevice(dev.DevModel(), mrnes.DefaultSwitchOp, msg.MsgLen)
}




// user extension of core methods for srvRsp and finish function classes
func extendSetup() {
    // visit every CmpPtn
    for _, cpi := range pces.CmpPtnInstByID {
        // find list of functions in group "EndMsr", if any
        funcLists, present := cpi.FuncsByGroup["EndMsr"] 
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

	// include functions for bespoke switch delays
	hubWest := mrnes.SwitchDevByName["hubWest"]
	hubWest.AddDevExecOp("outboundClient", outboundClientFunc)
	hubWest.AddDevExecOp("outboundWE", outboundWEFunc)

	hubEast := mrnes.SwitchDevByName["hubEast"]
	hubEast.AddDevExecOp("checkSrc", checkSrcFunc)
}

// CPClassify returns the CmpPtn name of execution thread source
func CPClassify(execID int, visited []int) string {
	CPName, CPpresent := pces.ExecIDCP[execID]
	if !CPpresent {
		return "default"
	}
	LabelName, Labelpresent := pces.ExecIDLabel[execID]
	if !Labelpresent {
		return CPName+":default"
	}
	return CPName+":"+LabelName
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
                if err != nil || (lambda < 0.0) {
                    panic("expect lambda to be non-negative float")
                }
                // skip flows with no rate
                if lambda == 0.0 {
                    continue
                }
				SaveLambda[cpfi.ID] = lambda
				arrival := interArrival(SaveLambda[cpfi.ID])
                evtMgr.Schedule(cpfi, nil, startEnterCycle, 
					vrtime.SecondsToTime(arrival))
            }
			if cpfi.Class == "streamsrc" {
				evtMgr.Schedule(cpfi, nil, pces.EnterFunc, vrtime.SecondsToTime(0.0))
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

    cpm.ExecID = pces.NewExecID(cpfi.PtnName, cpfi.Label) 

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
	pces.Batch = 20 
	pces.Skip = 5
	csvFileName := *context.(*string)
	exprmntName := *data.(*string)

	// create a csv header
	empty := []string{}

	// visit every group and build csv representation of its data	
	for _, msrg := range pces.MsrGrpByID {
		msrg.PrepCSVRow(csvFileName, pces.ExprmntsFile,
			exprmntName, empty, empty, "CI") 
		msrg.AddCSVData(csvFileName, exprmntName, empty)
	}

	// put the trace file in the same directory as the msrFile
	tdirectory, file := filepath.Split(csvFileName)
	file = "trace.yaml"
	traceFile := filepath.Join(tdirectory, file)
	pces.TraceMgr.WriteToFile(traceFile, false)

	fmt.Printf("Number of events executed %d\n", evtMgr.NumEvts)

	return nil 
}
