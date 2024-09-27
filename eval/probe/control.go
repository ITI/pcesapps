package probe

// control.go holds variables and functions related to the
// ProbeControl logic for running a probing experiment
//
import (
	"fmt"
	"github.com/iti/pces"
	"github.com/iti/evt/evtm"
	"github.com/iti/evt/vrtime"
	"github.com/iti/mrnes"
	"gopkg.in/yaml.v3"
	"encoding/json"
)

// PerfRecord stores the Measured information about the packet probe and the
// flow probe from SrcName CP to DstName CP
type PerfRecord struct {
	SrcDev string
	DstDev string
	ClassID int
	Latency float64
	Bndwdth float64
	PrLoss float64
}


// Three separate communications from source to destination are involved in a measurement,
// so workingRecord holds the state of the measurements as they are gathered for that trio
// and saves it when they are all finished.
var workingRecord PerfRecord

// MsrData is the structure which holds the Measured output, 
// and whose serialization is written to file when finished
type MsrData struct {
	Measurements []PerfRecord
}

// Serialize creates a string from an instance of the MsrData structure
func (msrd *MsrData) Serialize(useYAML bool) (string, error) {
	var bytes []byte
	var merr error
	if useYAML {
		bytes, merr = yaml.Marshal(*msrd)
	} else {
		bytes, merr = json.Marshal(*msrd)
	}
	if merr != nil {
		panic(fmt.Errorf("measurement marshalling error"))
	}
	outputStr := string(bytes[:])
	return outputStr, nil
}

// Measured is the data variable holding the measurements
var Measured *MsrData 

// variable used to create the execution ID for the probes
var numExecThreads int = 1

// genProbeMsg creates a pces.CmpPtnMsg instance for a probe, given the inputs
// of the probe endpoints 
func genProbeMsg(srcCPName, dstCPName string, execID int, classID int) *pces.CmpPtnMsg {
	cpm := new(pces.CmpPtnMsg)
	
	// get the integer IDs from the string names
	srcCP := pces.CmpPtnInstByName[srcCPName]
	dstCP := pces.CmpPtnInstByName[dstCPName]

	// fill in the source coordinates in the message's CP header
	cpm.CmpHdr.SrtCPID = srcCP.ID
	cpm.CmpHdr.SrtLabel = "probe"

	// flag to start the timer on this execution thread
	cpm.Start = true
	
	cpm.PcktLen = 1500
	cpm.MsgLen  = 1500
	cpm.ExecID  = execID
	cpm.ClassID = classID
	
	numExecThreads += 1
	cpm.MsgType  = "probe"

	// pces needs to see that the next CP is different from the current one,
	// and the current one (and current label) are assumed to be in the PrevCPID
	// and PrevLabel fields, which they would be if the function evaluation was triggered
	// by the receipt of a message from an edge
	cpm.PrevCPID = srcCP.ID
	cpm.PrevLabel = "probe"

	cpm.NxtCPID    = dstCP.ID
	cpm.NxtLabel = "probe"

	// A non-empty NxtMC field is used as a method code, rather than
	// extracting one from an edge
	cpm.NxtMC    = "probesend"

	return cpm
}

type msgEntry int

const (
	entry msgEntry = iota
	bckgrndEstablished	
	rtnFlowClassProbe
	rtnLowClassProbe
	nxtPair	
)

// scheduleProbe schedules the sending Func to initiate a probe
func scheduleProbe(evtMgr *evtm.EventManager, msg *pces.CmpPtnMsg) {
	DstCP := pces.CmpPtnInstByID[msg.NxtCPID]
	msg.Payload = CreateProbePayload(DstCP.Name, "probe", ProbeControl)
	msg.NxtMC = "probesend"
	srcCP := pces.CmpPtnInstByID[msg.CmpHdr.SrtCPID]
	cpfi := srcCP.Funcs["probe"]
	evtMgr.Schedule(cpfi, msg, pces.EnterFunc, vrtime.SecondsToTime(0.0))
}

// the pcState and expIdx variables hold the state information for ProbeControl
// as it cycles through the probe pairs, and for a probe pair, cycles through the
// three src -> dst initiations 
var pcState = entry
var state map[int]msgEntry = make(map[int]msgEntry)

var expIdx int
var bckgrndRate float64
var bgf *mrnes.BckgrndFlow

func ProbeControl(evtMgr *evtm.EventManager, context any, data any) any {
	// context is a string, "start" or "return" that either starts the experiment
	// or is a return from a completed probe

	cmd := context.(string)
	classID := 1
	var numFlows int

	// if the command is "start" we initialize the state and start
	if cmd == "start" {

		// make the structure where measurements are placed
		Measured = new(MsrData)
		Measured.Measurements = make([]PerfRecord,0)

		// create a backgrond flow	

		// make it look like the last step was the shutting down of a flow
		expIdx = -1 
		numFlows = len(ProbeExp)
		pcState = nxtPair
	}

	// action depends on state
	for true {
		switch pcState {
			case nxtPair:

				// advance the state
				expIdx += 1

				// detect termination
				if len(ProbeExp) <= expIdx {
					return nil
				}

				srcDev := ProbeExp[expIdx].SrcDev
				dstDev := ProbeExp[expIdx].DstDev

				// see how much bandwidth is not already used for a flow 

				if expIdx==0 {
					maxRate := mrnes.LimitingBndwdth(srcDev, dstDev)
					bckgrndRate = maxRate/float64(numFlows+1)
					fmt.Printf("Setting flow rate to %f\n", bckgrndRate)
				}

				// schedule the background flow from src to dst, and indicate how to come back
				var OK bool
				bgf, OK = mrnes.CreateBckgrndFlow(evtMgr, srcDev, dstDev, bckgrndRate, 
					3*(expIdx-1), nxtFlowID(), classID, "background", ProbeControl) 

				if !OK {
					panic("not OK")
				}

				// when the reservation has completed ProbeControl will be scheduled with a context of "bckgrnd"
				pcState = bckgrndEstablished
				return nil

			// return from establishing background flow
			case bckgrndEstablished:

				// create a packet probe message between same endpoints as the flow just established
				// with the same class as the flow
				srcCPName := ProbeCPByDevName[ProbeExp[expIdx].SrcDev]
				dstCPName := ProbeCPByDevName[ProbeExp[expIdx].DstDev]

				msg := genProbeMsg(srcCPName, dstCPName, 3*(expIdx-1)+1, bgf.ClassID)

				scheduleProbe(evtMgr, msg)
				pcState = rtnFlowClassProbe
				return nil
				
			// return from probe
			case rtnFlowClassProbe, rtnLowClassProbe:

				flowClassID := 0
				if pcState == rtnFlowClassProbe {
					flowClassID = classID
					pcState = rtnLowClassProbe
				} else {
					pcState = nxtPair
				}

				msg := data.(*pces.CmpPtnMsg)
				workingRecord = PerfRecord{SrcDev: ProbeExp[expIdx].SrcDev, ClassID: flowClassID, DstDev: ProbeExp[expIdx].DstDev, 
					Bndwdth: msg.NetBndwdth, Latency: msg.NetLatency, PrLoss: msg.NetPrLoss} 

				Measured.Measurements = append(Measured.Measurements, workingRecord)
		}
	}
	return nil
}

// in debug needed to detect when/if turning the flow off cleaned up properly
func chkDevState(devName string) bool {
	passed := true
	dev := mrnes.TopoDevByName[devName]
	for _, intrfc := range dev.DevIntrfcs() {
		if intrfc.State.IngressLambda >0.0 || intrfc.State.EgressLambda > 0.0 {
			fmt.Printf("non empty interface %s at %s\n", intrfc.Name, devName)
			fmt.Printf("\t ingress %f, egress %f\n", intrfc.State.IngressLambda, intrfc.State.EgressLambda)
			passed = false
		}
	}
	return passed
}

func chkNetState(netName string) bool {
	passed := true
	net := mrnes.NetworkByName[netName]
	if len(net.NetState.Flows) > 0 || net.NetState.Load > 0.0 {
		fmt.Printf("non empty network %s, flows %d, load %f\n", netName, len(net.NetState.Flows), net.NetState.Load)
		passed = false
	}
	return passed
}

func chkState() bool {
	passed := true
	for netName := range mrnes.NetworkByName {
		passed = passed && chkNetState(netName)
	}

	for switchName := range mrnes.SwitchDevByName {
		passed = passed && chkDevState(switchName)
	}

	for routerName := range mrnes.RouterDevByName {
		passed = passed && chkDevState(routerName)
	}
	return passed
}

var bckgrndFlowCnt int = 0
func nxtFlowID() int {
	bckgrndFlowCnt += 1
	return bckgrndFlowCnt
}


