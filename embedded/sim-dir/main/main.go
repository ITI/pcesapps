package main
import (
	"github.com/iti/evt/evtm"
	"github.com/iti/evt/vrtime"
	"github.com/iti/pces"
	"path/filepath"
)

func main() {
	pces.ReadSimArgs() 
	pces.RunExperiment(expCntrl, expCmplt) 
}

func expCntrl(evtMgr *evtm.EventManager, context any, data any) any {
	// go through all comp patterns looking for functions of the 'start' class,
	// and schedule them for execution
	for _, cpi := range pces.CmpPtnInstByID {
		for _, cpfi := range cpi.Funcs {
			if cpfi.Class == "start" {
				evtMgr.Schedule(cpfi, &cpfi.Class, 
					pces.EnterFunc, vrtime.SecondsToTime(0.0))
			}
		}
	}
	return nil 
}

func expCmplt(evtMgr *evtm.EventManager, context any, data any) any {
	csvFileName := *context.(*string)
	exprmntName := *data.(*string)
	exprmntsFile := pces.ExprmntsFile

	// should be only one MsrGroup
	empty := []string{}
	for _, msrg := range pces.MsrGrpByID {

		msrg.PrepCSVRow(csvFileName, exprmntsFile,
		   exprmntName, empty, empty, "Samples") 

		// write out the samples
		msrg.AddCSVData(csvFileName, exprmntName, empty )
		break
	}

	// put the trace file in the same directory as the msrFile
	tdirectory, file := filepath.Split(csvFileName)
	file = "trace.yaml"
	traceFile := filepath.Join(tdirectory, file)
	pces.TraceMgr.WriteToFile(traceFile, false)
	return nil 
}

