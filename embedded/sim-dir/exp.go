package main

import (
	"github.com/iti/pces"
	"github.com/iti/evt/evtm"
	"github.com/iti/evt/vrtime"
	
)

var defaultStr string = "default"

func expCntrl(evtMgr *evtm.EventManager, context any, data any) any {
	// go through all comp patterns looking for functions of the 'start' class,
	// and schedule them for execution
	for _, cpi := range pces.CmpPtnInstByID {
		for _, cpfi := range cpi.Funcs {
			if cpfi.Class == "start" {
				evtMgr.Schedule(cpfi, &cpfi.Class, pces.EnterFunc, vrtime.SecondsToTime(0.0))
			}
		}
	}
	return nil 
}

func expComplete(evtMgr *evtm.EventManager, context any, data any) any {
	// go through all comp patterns looking for functions of the 'start' class,
	// and schedule them for execution
	msrFileName := *data.(*string)
	pces.SaveMeasureResults(msrFileName, true)
	return nil 
}


