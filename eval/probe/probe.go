package probe

// the probe class is used to measure estimated latency, bandwidth, and packet loss
// in an mrnes network. The idea is to place one Func per Comp Pattern, and have that
// Func serve both as a source of the probe measurements and and destination.
// A probe experiment is controled by an event handler ProbeControl.  That handler
// works through a list of pairs of computational patterns, one a source, the other a destination.
// For each pair the ProbeControl sets up three src -> dst runs.  One passes an Ethernet frame size
// message, gathering latency and estimated packet loss information.  Another starts a "flow" which
// is just a bit rate that is shaped by the capabilities and current loading conditions of the devices
// on the path from source to destination.  The third run reports the closing of the previous run that
// set up flow, and removes the presence of the flow from the network data structures.  After each run
// the receiving probe Func reports the result of the run to ProbeControl, which assembles the 
// three src -> dst metrics and saves a record of them. After all pairs have been exercised
// the simulation code can have a record of them written to file.

import (
	"fmt"
	"github.com/iti/pces"
	"github.com/iti/evt/evtm"
	"github.com/iti/evt/vrtime"
	"gopkg.in/yaml.v3"
	"encoding/json"
)

// like every Func class, get the probe class recognized within pces
// when the file is loaded, by any application that imports it
var prbcfgVar *ProbeCfg = ClassCreateProbeCfg()
var prbcfgLoaded bool = pces.RegisterFuncClass(prbcfgVar)

var ProbeCPByDevName map[string]string = make(map[string]string)

// like every Func class, define a Cfg struct that will be put into the cpInit input
// file for the controler
type ProbeCfg struct {
	Trace     bool				  `yaml:"trace" json:"trace"`
	Device	  string			  `yaml:"device" json:"device"`
}

// CreateProbeCfg is a constructor.  
func CreateProbeCfg(device string) *ProbeCfg {
	pc := new(ProbeCfg)
	pc.Trace = false
	pc.Device = device 
	return pc
}

// ProbeState holds name of the device to which the probe is mapped
type ProbeState struct {
	calls     int
	device	  string
}

// ProbePayload is a type defining the payload of a pces.CmpPtnMsg
// meaning that it is conveyed from the probe source to the probe destination
// as the Msg field of the mrnes.NetworkMsg.   Created by the ProbeControl and
// passed to the probe sending function, that function reads from TgtCP and TgtLabel
// where to aim the probe.   On receipt, the receiving probe Func uses the Rtn field
// to report (to the ProbeControl) the receipt
//
type ProbePayload struct {
	TgtCP string
	TgtLabel string
	Rtn evtm.EventHandlerFunction
}

// CreateProbePayload is a constructor
func CreateProbePayload(cp, label string, rtn evtm.EventHandlerFunction) ProbePayload {
	ppl := new(ProbePayload)
	ppl.TgtCP = cp
	ppl.TgtLabel = label
	ppl.Rtn = rtn
	return *ppl
}

// ClassCreateProbe is a constructor called just to create an instance, 
// and put reference to probe and its methods in the pces data structures
func ClassCreateProbeCfg() *ProbeCfg {
	prbcfg := new(ProbeCfg)
	prbcfg.Trace = false

	// put the event handling information into pces.ClassMethods
	fmap := make(map[string]pces.RespMethod)
    fmap["probesend"] = pces.RespMethod{Start: probeSend, End: pces.ExitFunc}
    fmap["proberecv"] = pces.RespMethod{Start: probeRecv, End: pces.ExitFunc}
	pces.ClassMethods["probe"] = fmap

	return prbcfg
}

// createProbestate is a pretty boring constructor for a pretty boring state
func createProbeState() *ProbeState {
	prbs := new(ProbeState)
	prbs.calls = 0
	prbs.device = ""
	return prbs
}

// FuncClassName required for the FuncClassCfg interface
func (prbcfg *ProbeCfg) FuncClassName() string {
	return "probe"
}

// CreateCfg required for the FuncClassCfg interface
func (prbcfg *ProbeCfg) CreateCfg(cfgStr string, useYAML bool) any {
	prbcfgVarAny, err := prbcfg.Deserialize(cfgStr, useYAML)
	if err != nil {
		panic(fmt.Errorf("probe.InitCfg sees deserialization error"))
	}
	return prbcfgVarAny
}


// InitCfg required for the FuncClassCfg interface
func (prbcfg *ProbeCfg) InitCfg(cpfi *pces.CmpPtnFuncInst, cfgStr string, useYAML bool) {

	// Deserialize the configuration for this Func
	prbcfgVarAny := prbcfg.CreateCfg(cfgStr, useYAML)
	prbcfgv := prbcfgVarAny.(*ProbeCfg)
	cpfi.Cfg = prbcfgv

	cpfi.State = createProbeState()
	cpfi.Trace = prbcfgv.Trace

	CPName := pces.CmpPtnInstByID[cpfi.CPID].Name
	ProbeCPByDevName[prbcfgv.Device] = CPName

	// put the event handling information into pces.ClassMethods
	fmap := make(map[string]pces.RespMethod)
    fmap["probesend"] = pces.RespMethod{Start: probeSend, End: pces.ExitFunc}
    fmap["proberecv"] = pces.RespMethod{Start: probeRecv, End: pces.ExitFunc}
	pces.ClassMethods["probe"] = fmap
}

// ValidateCfg is boring because there isn't really anything to validate
func (prbcfg *ProbeCfg) ValidateCfg(cpfi *pces.CmpPtnFuncInst) error {
	return nil
}

// Serialize transforms the probe cfg into string form for
// inclusion through a file
func (prbcfg *ProbeCfg) Serialize(useYAML bool) (string, error) {
	var bytes []byte
	var merr error

	if useYAML {
		bytes, merr = yaml.Marshal(*prbcfg)
	} else {
		bytes, merr = json.Marshal(*prbcfg)
	}

	if merr != nil {
		return "", merr
	}

	return string(bytes[:]), nil
}

// Deserialize recovers a serialized representation of a probe cfg structure
func (prbcfg *ProbeCfg) Deserialize(fss string, useYAML bool) (any, error) {
	// turn the string into a slice of bytes
	var err error
	fsb := []byte(fss)

	example := ProbeCfg{Trace: false, Device: ""}

	// Select whether we read in json or yaml
	if useYAML {
		err = yaml.Unmarshal(fsb, &example)
	} else {
		err = json.Unmarshal(fsb, &example)
	}

	if err != nil {
		return nil, err
	}
	return &example, nil
}

// now include the functions whose executions are triggered by messages to the probe function,
// passing through pces.EnterFunc.
//
// probeSend schedules the generation of one packet/or flow, from a specified source to a specified destination
// Initiated by a poke from the global controller
func probeSend(evtMgr *evtm.EventManager, cpfi *pces.CmpPtnFuncInst, methodCode string, msg *pces.CmpPtnMsg) {
	prbs := cpfi.State.(*ProbeState)
	prbs.calls += 1

	// extract the target CP and label from the CmpPtnMsg payload (set up by the ProbeControl logic)
	payload := msg.Payload.(ProbePayload)

	// Update message is going to move these to fields in the CmpPtnMsg where it will appear
	// as though they had been specifying a target (where the flow of control is right now)
	// Is needed to compare with the actual nextCP to detect that the Computational Pattern identity is changing
	msg.NxtCPID = cpfi.CPID
	msg.NxtLabel = "probe"

	// for UpdateMsg we figure out the destination coordinates
	nxtCPID := pces.CmpPtnInstByName[payload.TgtCP].ID
	nxtLabel := payload.TgtLabel
	
	// update the message to reflect next station. message type is "probe", receiver recipient code is "proberecv"
	// put message where ExitFunc will find it
	pces.UpdateMsg(msg, nxtCPID, nxtLabel, "probe", "proberecv")

	// put the message where pces.ExitFunc will be looking for it
	cpfi.AddResponse(msg.ExecID, []*pces.CmpPtnMsg{msg})
	
	// just schedule ExitFunc
	evtMgr.Schedule(cpfi, msg, pces.ExitFunc, vrtime.SecondsToTime(0.0))
	return
}

// probeRecv gets called when the probe shows up at the destination.  It's role
// is to pass the msg back to the return event handler, e.g. ProbeControl, with
// the latency, bandwidth, and packet loss information embedded in it. 
func probeRecv(evtMgr *evtm.EventManager, cpfi *pces.CmpPtnFuncInst, methodCode string, msg *pces.CmpPtnMsg) {
	prbs := cpfi.State.(*ProbeState)
	prbs.calls += 1

	// the payload carried across the network the identity of the event handler to schedule
	// on arrival
	payload := msg.Payload.(ProbePayload)	

	// just return the received message to the handler carried in the payload
	msg.NxtMC = "proberecv"
	//evtMgr.Schedule("return", msg, ProbeControl, vrtime.SecondsToTime(0.0))
	evtMgr.Schedule("return", msg, payload.Rtn, vrtime.SecondsToTime(0.0))
}

