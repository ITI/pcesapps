package main

// code to build pces computational pattern (CmpPtn) structures and their initialization structs
// for the beta architecture(s)

import (
	"fmt"
	"github.com/iti/cmdline"
	"github.com/iti/mrnes"
	"github.com/iti/pces"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

// cmdlineParams defines the parameters recognized
// on the command line
func cmdlineParams() *cmdline.CmdParser {
	// command line parameters are all about file and directory locations.
	// Even though we don't need the flags for the other pces structures we
	// keep them here so that all the programs that build templates can use the same arguments file
	// create an argument parser

	cp := cmdline.NewCmdParser()
	cp.AddFlag(cmdline.StringFlag, "outputLib", true) // directory where model is written
	cp.AddFlag(cmdline.StringFlag, "db", true)        // directory where some 'database' records are organized
	cp.AddFlag(cmdline.StringFlag, "cp", true)        // name of output file for computation pattern
	cp.AddFlag(cmdline.StringFlag, "cpInit", true)    // name of output file for initialization blocks for CPs
	cp.AddFlag(cmdline.StringFlag, "funcExec", true)  // name of output file used for function timings
	cp.AddFlag(cmdline.StringFlag, "devExec", true)   // name of output file used for device operations timings
	cp.AddFlag(cmdline.StringFlag, "devDesc", true)   // name of input file
	cp.AddFlag(cmdline.StringFlag, "srdState", false) // name of output file holding shared state description
	cp.AddFlag(cmdline.StringFlag, "map", true)       // file with mapping of CmpPtn functions to devices
	cp.AddFlag(cmdline.StringFlag, "exp", true)       // name of output file used for run-time experiment parameters
	cp.AddFlag(cmdline.StringFlag, "topo", false)     // name of output file used for topo templates
	cp.AddFlag(cmdline.BoolFlag, "useJSON", false)    // use JSON rather than YAML for serialization

	cp.AddFlag(cmdline.BoolFlag, "sslsrvr", true)       // if true include an SSL server, else not (different topo)
	cp.AddFlag(cmdline.IntFlag, "euds", true)           // number of EUDs in model
	cp.AddFlag(cmdline.IntFlag, "switchports", true)    // number of ports per switch
	cp.AddFlag(cmdline.IntFlag, "srccores", true)       // number of cores used on srcPckt
	cp.AddFlag(cmdline.IntFlag, "sslcores", false)       // number of cores used on srcPckt
	cp.AddFlag(cmdline.IntFlag, "eudcores", true)       // number of cores used on srcPckt
	cp.AddFlag(cmdline.StringFlag, "cryptoalg", true)   // string description which crypto algorithm is used
	cp.AddFlag(cmdline.StringFlag, "keylength", true)   // string description which crypto algorithm is used
	cp.AddFlag(cmdline.IntFlag, "pcktlen", true)        // length of packet in data message
	cp.AddFlag(cmdline.IntFlag, "pcktburst", true)      // number of packets in a burst
	cp.AddFlag(cmdline.IntFlag, "eudcycles", false)      // number of times to cycle through the EUD bursts
	cp.AddFlag(cmdline.StringFlag, "pcktMu", true)       // mean inter-arrival time  between to src process
	cp.AddFlag(cmdline.StringFlag, "burstMu", false)      // mean inter-burst time 
	cp.AddFlag(cmdline.StringFlag, "cycleMu", false)  // mean inter-cycle time 
	cp.AddFlag(cmdline.StringFlag, "srcCPU", true)      // type of CPU on src device
	cp.AddFlag(cmdline.StringFlag, "srcCPUBw", true)    // Mbs of interfaces on srcCPU
	cp.AddFlag(cmdline.StringFlag, "eudCPU", true)      // type of CPU on eud device
	cp.AddFlag(cmdline.StringFlag, "eudCPUBw", true)    // Mbs of interfaces on eudCPU
	cp.AddFlag(cmdline.StringFlag, "pubSwitch", true)   // switch type in PubNet
	cp.AddFlag(cmdline.StringFlag, "pvtSwitch", true)   // switch type in PvtNet
	cp.AddFlag(cmdline.StringFlag, "pvtNetBw", true)    // Mbs of switch interfaces in the private network
	cp.AddFlag(cmdline.StringFlag, "pubNetBw", true)    // Mbs of switch interfaces in the public network
	cp.AddFlag(cmdline.StringFlag, "pvtSwitchBw", true) // Mbs of switch interfaces in the private network
	cp.AddFlag(cmdline.StringFlag, "pubSwitchBw", true) // Mbs of switch interfaces in the public network
	cp.AddFlag(cmdline.StringFlag, "pubRtr", true)      // type of router in PubNet
	cp.AddFlag(cmdline.StringFlag, "pvtRtr", true)      // type of router in PvtNet
	cp.AddFlag(cmdline.StringFlag, "pvtRtrBw", true)    // Mbs of router interfaces in the private network
	cp.AddFlag(cmdline.StringFlag, "pubRtrBw", true)    // Mbs of router interfaces in the public network
	cp.AddFlag(cmdline.StringFlag, "sslCPU", false)     // CPU type for ssl device when present
	cp.AddFlag(cmdline.StringFlag, "sslCPUBw", false)   // Mbs of interfaces on ssl when present
	return cp
}

var useYAML bool

// devDescMap is a description of hardware devices, read up
// from an auxilary file
var devDescMap map[string]mrnes.DevDesc

// main gives the entry point
func main() {
	// define the command line parameters
	cp := cmdlineParams()

	// parse the command line
	cp.Parse()

	// string for the output directory
	outputLib := cp.GetVar("outputLib").(string)

	// string for the database directory
	dbLib := cp.GetVar("db").(string)

	timingDir := filepath.Join(dbLib,"timing")
	funcXDir  := filepath.Join(timingDir,"funcExec")
	devXDir  := filepath.Join(timingDir,"devExec")

	// make sure these directories exist
	dirs := []string{outputLib, dbLib, funcXDir, devXDir}
	valid, err := pces.CheckDirectories(dirs)
	if !valid {
		panic(err)
	}

	// name of the file with hardware description dictionary
	devDescName := cp.GetVar("devDesc").(string)

	// make sure that devDesc exists
	devDescFile := filepath.Join(outputLib, devDescName)
	_, errc := pces.CheckFiles([]string{devDescFile}, true)
	if errc != nil {
		panic(errc)
	}

	empty := []byte{}
	devDescDict, dderr := mrnes.ReadDevDescDict(devDescFile, true, empty)
	if dderr != nil {
		panic(dderr)
	}

	devDescMap = devDescDict.DescMap

	// check for access to the files the application will write
	fullpathmap := make(map[string]string)
	outFiles := []string{"cp", "cpInit", "funcExec", "devExec", "exp", "topo", "map", "srdState"}
	fullpath := []string{}
	for _, filename := range outFiles {
		basefile := cp.GetVar(filename).(string)
		fullfile := filepath.Join(outputLib, basefile)
		fullpath = append(fullpath, fullfile)
		fullpathmap[filename] = fullfile
	}

	valid, err = pces.CheckOutputFiles(fullpath)
	if !valid {
		panic(err)
	}

	useYAML = true
	if cp.IsLoaded("useJSON") {
		useJSON := cp.GetVar("useJSON").(bool)
		if useJSON {
			useYAML = false
		}
	}

	// the application will build an architecture that comes with a dedicate
	// SSL server, or another that doesn't.   The sslsrvr flag indicates which
	hasSSL := cp.GetVar("sslsrvr").(bool)
	archType := "SSL"

	if !hasSSL {
		archType = "NoSSL"
	} else {
		// if we're building in an SSL server its CPU and interface bandwidth needs to be specified
		if !cp.IsLoaded("sslCPU") {
			panic(fmt.Errorf("must specify CPU for SSL server"))
		}
		if !cp.IsLoaded("sslCPUBw") {
			panic(fmt.Errorf("must specify bandwidth for SSL server"))
		}
	}

	// read in parameters describing packet behavior
	pcktSize := cp.GetVar("pcktlen").(int)
	pcktBurst := cp.GetVar("pcktburst").(int)
	keyLength := cp.GetVar("keylength").(string)

	// read the inter-arrival means as strings in order to
	// determine distribution type.  Floating point (with ".")
	// means
	pcktMuStr := cp.GetVar("pcktMu").(string)

	burstMuStr := pcktMuStr
	if cp.IsLoaded("burstMu") {	
		burstMuStr = cp.GetVar("burstMu").(string)
	}

	cycleMuStr := pcktMuStr
	if cp.IsLoaded("cycleMu") {	
		cycleMuStr = cp.GetVar("cycleMu").(string)
	}

	pcktMu, _  := strconv.ParseFloat(pcktMuStr, 64)
	burstMu, _ := strconv.ParseFloat(burstMuStr, 64)
	cycleMu, _ := strconv.ParseFloat(cycleMuStr, 64)

	// mu values are in milliseconds, so scale to be in seconds
	pcktMu /= 1000.0
	burstMu /= 1000.0
	cycleMu /= 1000.0

	pcktMuDist := "const"
	burstMuDist := "const"
	cycleMuDist := "const"

	if strings.Contains(pcktMuStr,"e") {
		pcktMuDist = "exp"
	} 

	if strings.Contains(burstMuStr,"e") {
		burstMuDist = "exp"
	} 

	if strings.Contains(cycleMuStr,"e") {
		cycleMuDist = "exp"
	} 

	eudCycles := int(1)
	if cp.IsLoaded("eudcycles") {
		eudCycles = cp.GetVar("eudcycles").(int)
	}

	// euds is the number of external user devices in the architecture
	euds := cp.GetVar("euds").(int)

	// switchports is the number of ports in the switches used to build
	// a tree of switches from the PubNet switch to the EUDs
	switchports := cp.GetVar("switchports").(int)
	srccores := cp.GetVar("srccores").(int)
	eudcores := cp.GetVar("eudcores").(int)
	sslcores := int(1)
	if archType == "SSL" {	
		sslcores = cp.GetVar("sslcores").(int)
	}
	
	// cryptoalg indicates which of several crypto algorithms
	// have performance profiles we can use
	cryptoAlg := cp.GetVar("cryptoalg").(string)
	cryptoAlg = strings.ToLower(cryptoAlg)

	// the base computational pattern is the one where packets are generated and
	// to which they return.  The 'type' is a string, here effectively used as a name
	// also which appends the architectural selection to the string 'encryptPerf'
	cmpPtnType := "encryptPerf-" + archType

	// create a computational pattern data structure
	encryptPerf := pces.CreateCompPattern(cmpPtnType)

	// gather up performance parameters for the networks to be modeled
	srcCPUType := cp.GetVar("srcCPU").(string)
	srcCPUBw := cp.GetVar("srcCPUBw").(string)
	pvtSwitchType := cp.GetVar("pvtSwitch").(string)
	pubSwitchType := cp.GetVar("pubSwitch").(string)
	pvtNetBw := cp.GetVar("pvtNetBw").(string)
	pubNetBw := cp.GetVar("pubNetBw").(string)
	pvtSwitchBw := cp.GetVar("pvtSwitchBw").(string)
	pubSwitchBw := cp.GetVar("pubSwitchBw").(string)
	pvtRtrType := cp.GetVar("pvtRtr").(string)
	pubRtrType := cp.GetVar("pubRtr").(string)
	pvtRtrBw := cp.GetVar("pvtRtrBw").(string)
	pubRtrBw := cp.GetVar("pubRtrBw").(string)
	sslCPUType := cp.GetVar("sslCPU").(string)
	sslCPUBw := cp.GetVar("sslCPUBw").(string)
	eudCPUType := cp.GetVar("eudCPU").(string)
	eudCPUBw := cp.GetVar("eudCPUBw").(string)

	// assume that the packets fit tightly into an IP/TCP ethernet frame,
	// one per frame.
	msgLen := pcktSize + 36

	// each EUD will have its own instance of a computational pattern,
	// a chain from "decryptOut" -> "eudProcess" -> "encryptRtn" -> "chgCPRtn"
	// The last function in the chain exists to highlight that the outboound message
	// changes the computational pattern from the EUD's to another, in this case
	// that holding the packet generating server

	// eudCmpPtn will be a template that all the EUD CmpPtns will follow,
	// and be lightly customized after being copied from the template
	eudCmpPtn := pces.CreateCompPattern("eudCmpPtn")

	// create the EUD computation pattern functions
	// The first parameter identifies the name of a Class the function belongs to,
	// and the second a name for this instance of the function.   There are Class-specific
	// methods in the simulator used to model the execution of these functions
	decryptOutFunc := pces.CreateFunc("cryptoPckt", "decryptOut")
	processFunc := pces.CreateFunc("processPckt", "eudProcess")
	encryptRtnFunc := pces.CreateFunc("cryptoPckt", "encryptRtn")
	chgCPRtnFunc := pces.CreateFunc("chgCP", "chgCPRtn")

	// include the functions in the EUD CmpPtn template
	eudCmpPtn.AddFunc(decryptOutFunc)
	eudCmpPtn.AddFunc(processFunc)
	eudCmpPtn.AddFunc(encryptRtnFunc)
	eudCmpPtn.AddFunc(chgCPRtnFunc)

	// create a CmpPtn that models a single process which cycles through target EUDs, shooting
	// a burst of packets at each.  The pattern is comprised of the chain
	//    burstSrc -> encryptOut -> chgCPOut,
	// and also (separately) decryptRtn -> finish
	//  'finish' calls out points where movement of message ends and performance measurements are taken
	srcFunc := pces.CreateFunc("cycleDst", "cycleDst")
	encryptOutFunc := pces.CreateFunc("cryptoPckt", "encryptOut")
	chgCPOutFunc := pces.CreateFunc("chgCP", "chgCPOut")
	decryptRtnFunc := pces.CreateFunc("cryptoPckt", "decryptRtn")

	finishFunc := pces.CreateFunc("finish", "finish")

	// add the functions to the packet generation CmpPtn
	encryptPerf.AddFunc(srcFunc)
	encryptPerf.AddFunc(encryptOutFunc)
	encryptPerf.AddFunc(chgCPOutFunc)
	encryptPerf.AddFunc(decryptRtnFunc)
	encryptPerf.AddFunc(finishFunc)

	// The CmpPtn functions (and TBD edges) define CmpPtn topology.
	// For each CmpPtn we also define a dictionary that has data and structures
	// specific to the individual components of the CmpPtn, in the output file
	// called cpInit.yaml. When we define edges for the CmpPtn we'll also specify
	// message 'types' (really just a label), and so we plunk these into a cpInit
	// structure before declaring the edges, so that we can better error check
	// the inclusion of edges

	// epCPInit is a template for the cpInit structures of the EUD CmpPtn
	// The first (here empty) argument is a name, the second a type.  The
	// block we are defining here is a template, each instance will get its own
	// name
	epCPInit := pces.CreateCPInitList("", "eudCmpPtn", true)

	// describe the types, packet sizes, and frame lengths of the messages
	// that pass between functions in the EUD CmpPtn
	epCPInit.AddMsg(pces.CreateCompPatternMsg("plaintext", pcktSize, msgLen))
	epCPInit.AddMsg(pces.CreateCompPatternMsg("encryptext", pcktSize, msgLen))

	// create the CP init data structure for the packet source CmpPtn and add the message types it sees
	epCPSrcInit := pces.CreateCPInitList(encryptPerf.Name, cmpPtnType, true)
	epCPSrcInit.AddMsg(pces.CreateCompPatternMsg("initiate", pcktSize, msgLen))
	epCPSrcInit.AddMsg(pces.CreateCompPatternMsg("plaintext", pcktSize, msgLen))
	epCPSrcInit.AddMsg(pces.CreateCompPatternMsg("encryptext", pcktSize, msgLen))

	// data connections between functions are described as directed 'edges'
	// Each call to AddEdge below is specific to a CmpPtn,
	// and gives the names of the source and destination functions,
	// the 'type' label of the message that is carried, and a name of a method
	// at the recipient to be called on receipt of such a message from that source.
	// The method code must be defined for the class of the destination function, and
	// indicates particular methods to be invoked in the processing of this message
	eudCmpPtn.AddEdge(decryptOutFunc.Label, processFunc.Label, "plaintext", "processOp", &epCPInit.Msgs)
	eudCmpPtn.AddEdge(processFunc.Label, encryptRtnFunc.Label, "plaintext", "cryptoOp", &epCPInit.Msgs)
	eudCmpPtn.AddEdge(encryptRtnFunc.Label, chgCPRtnFunc.Label, "encryptext", "chgCP", &epCPInit.Msgs)

	// each of the CmpPtn's functions gets a state dictionary whose structure is defined
	// by the function's class. Here we create and populate those structures, which
	// are serialized for storage to file

	// createProcessPcktState is a function defined in this file that creates a 'processPckt' class
	// state dictionary given required parameters.  It returns a string that results from serialization
	// The function whose state is created here models the decryption of a packet as it arrives
	// at the EUD.  The state of a processPckt class function includes a code for the particular
	// operation it models, here, a code like 'decrypt-aes' that indicates the operation and encryption
	// algorithm.   There is no particular grammer or limitations on what these codes are,
	// but the simulator will assume that certain table entries exist that match them.
	// We will later check this validity as it depends also on the mapping of functions to processors
	// that has not yet been specified.
	decryptOutStr := createCryptoPcktState("decrypt", cryptoAlg, keyLength, false)
	epCPInit.AddState(eudCmpPtn, decryptOutFunc, decryptOutStr)

	// the 'processFunc' function in an EUD CmpPtn models the computational delay of doing something
	// with the decrypted packet, before encrypting a response
	processStr := createProcessPcktState(eudCmpPtn, processFunc, "processOp", true)
	epCPInit.AddState(eudCmpPtn, processFunc, processStr)

	// the 'encryptRtn' function in an EUD CmpPtn models the delay of encrypting
	// a response to the message sent to the EUD
	encryptRtnStr := createCryptoPcktState("encrypt", cryptoAlg, keyLength, false)
	epCPInit.AddState(eudCmpPtn, encryptRtnFunc, encryptRtnStr)

	// the 'chgDesc' function moves a message from one CmpPtn into another.
	// The state is a map whose index is name of the CmpPtn in the destination field
	// of the message, whose attribute is a record that give the name of the new CmpPtn,
	// the label of the function in that CmpPtn to receive the message, and the message type
	// of the message as it makes the transition.  Four steps below

	// first step, create a state dictionary for the chgCP function
	chg := pces.CreateChgDesc(encryptPerf.Name, decryptRtnFunc.Label, "encryptext")

	// second step, create a map that will be put in the state dictionary
	chgMap := map[string]pces.ChgDesc{encryptPerf.Name: chg}

	// third step, insert the map, serialize the dictionary, and return the
	// string resulting from the serialization
	chgCPRtnStr := createChgCPState(chgMap)

	// fourth state, insert the func's serialized state dictionary into the cpInit structure
	epCPInit.AddState(eudCmpPtn, chgCPRtnFunc, chgCPRtnStr)

	// The overall model creates a CmpPtn for each EUD, named
	// "eudCmpPtn-x" for x between 0 and the number of EUDs specified (minus one).
	//  Structures eudCmpPtn
	// and epCPInit are templates for these.  For each EUD we make copies and then
	// modify slightly as needed to specialize for the specific EUD

	// create dictionaries for all the CmpPtns and all their cpInit auxilary structures
	cpDict := pces.CreateCompPatternDict("beta")
	cpInitDict := pces.CreateCPInitListDict("beta")

	// we'll glue on index strings to tailor a name for each EUD CmpPtn
	eudCPBaseName := "eudCmpPtn"

	// the encryptPerf CmpPtn function for changing CmpPtn will have an entry
	// for each of the eudCmpPtns.  We initialize it here and fill it in
	// the per-EUD loop below
	outbndChgMap := make(map[string]pces.ChgDesc)

	// make a unique CmpPtn instance for every eud
	for idx := 0; idx < euds; idx++ {

		eudIdx := strconv.Itoa(idx)

		// create a copy from the template
		cpyCP := eudCmpPtn.DeepCopy()

		// give it a unique name
		cpyCP.SetName(eudCPBaseName + "-" + strconv.Itoa(idx))

		// put in the external edge back to the encryptPerf CmpPtn, and
		// an external edge from encryptOut to the EUD.  Note that a different method (AddExtEdge)
		// is used to specify the cross-CmpPtn connections
		cpyCP.AddExtEdge(cpyCP.Name, encryptPerf.Name, encryptPerf.Name, chgCPRtnFunc.Label, decryptRtnFunc.Label,
			"encryptext", "cryptoOp", &epCPInit.Msgs, &epCPSrcInit.Msgs)
		encryptPerf.AddExtEdge(encryptPerf.Name, cpyCP.Name, cpyCP.Name, chgCPOutFunc.Label, decryptOutFunc.Label,
			"encryptext", "cryptoOp", &epCPSrcInit.Msgs, &epCPInit.Msgs)

		// include the outbound change of CmpPtn in the map to be put in chgCPOutFunc's state table
		outbndChgMap[cpyCP.Name] = pces.CreateChgDesc(cpyCP.Name, "decryptOut", "encryptext")

		// save the EUD CmpPtn in the output dictionary
		cpDict.AddCompPattern(cpyCP)

		// create a copy of CPInit
		//cpyCPInitList := new(pces.CPInitList)
		cpyCPInitList := epCPInit.DeepCopy()

		// give it a unique name
		cpyCPInitList.Name = cpyCPInitList.CPType + "-" + eudIdx

		// save it in the dictionary
		cpInitDict.AddCPInitList(cpyCPInitList)
	}

	// add edges to the packet source CmpPtn
	encryptPerf.AddEdge(srcFunc.Label, srcFunc.Label, "initiate", "generateOp", &epCPSrcInit.Msgs)
	encryptPerf.AddEdge(srcFunc.Label, encryptOutFunc.Label, "plaintext", "cryptoOp", &epCPSrcInit.Msgs)
	encryptPerf.AddEdge(encryptOutFunc.Label, chgCPOutFunc.Label, "encryptext", "chgCP", &epCPSrcInit.Msgs)
	encryptPerf.AddEdge(decryptRtnFunc.Label, finishFunc.Label, "plaintext", "finishOp", &epCPSrcInit.Msgs)

	// put in state parameters for srcFunc node.
	// Function type is 'cycleDst', which is tailored for this source.
	srcState := pces.ClassCreateCycleDst()

	// build out the state dictionary for the srcFunc
	srcState.Populate(euds, eudCPBaseName, 
		pcktMuDist, pcktMu, burstMuDist, burstMu, pcktBurst, 
		cycleMuDist, cycleMu, eudCycles, 
		msgLen, pcktSize, false) 

	// add a couple of operation descriptions
	srcState.AddOpName("generateOp", "generateOp")
	srcState.AddOpName("completeOp", "completeOp")

	// serialize srcFunc's state and add it to cpCPSrcInit
	serialSrcState, err0 := srcState.Serialize(useYAML)
	if err0 != nil {
		panic(err0)
	}
	epCPSrcInit.AddState(encryptPerf, srcFunc, serialSrcState)

	// put in parameters for encryptOutFunc
	encryptOutStr := createCryptoPcktState("encrypt", cryptoAlg, keyLength, false)
	epCPSrcInit.AddState(encryptPerf, encryptOutFunc, encryptOutStr)

	// put in parameters for decryptRtnFunc
	decryptOutStr = createCryptoPcktState("decrypt", cryptoAlg, keyLength, false)
	epCPSrcInit.AddState(encryptPerf, decryptRtnFunc, decryptOutStr)

	// make a minimalistic state for finish
	finishStr := createFinishState()
	epCPSrcInit.AddState(encryptPerf, finishFunc, finishStr)

	// create state for changing CmpPtn at function chgCPOutFunc
	chgCPOutStr := createChgCPState(outbndChgMap)
	epCPSrcInit.AddState(encryptPerf, chgCPOutFunc, chgCPOutStr)

	cpDict.AddCompPattern(encryptPerf)
	cpInitDict.AddCPInitList(epCPSrcInit)

	// write the CmpPtn stuff out
	cpDict.WriteToFile(fullpathmap["cp"])
	cpInitDict.WriteToFile(fullpathmap["cpInit"])

	// build a topology named with two networks, "Private" and "Public"
	tcf := mrnes.CreateTopoCfgFrame("EvaluateCrypto")

	// the two networks
	pvtNet := mrnes.CreateNetwork("private", "LAN", "wired")
	pubNet := mrnes.CreateNetwork("public", "LAN", "wired")

	// create a source node in pvtnet.
	var srcNode *mrnes.EndptFrame

	srcNode = mrnes.CreateHost("pcktsrc", srcCPUType, srccores)

	// create a switch and connect the pcktsrc to it
	pvtSwitch := mrnes.CreateSwitch("pvtSwitch", pvtSwitchType)
	mrnes.ConnectDevs(srcNode, pvtSwitch, true, pvtNet.Name)
	pvtNet.IncludeDev(pvtSwitch, "wired", true)

	// create a router and connect the switch to it
	pvtRtr := mrnes.CreateRouter("pvtRtr", pvtRtrType)
	mrnes.ConnectDevs(pvtSwitch, pvtRtr, true, pvtNet.Name)
	pvtNet.IncludeDev(pvtRtr, "wired", true)

	// default router that connects to the EUD connection tree, assumes no SSL device
	bridgeRtr := pvtRtr

	// if SSL is selected, create a router that joins pvtNet and pubNet
	if archType == "SSL" {
		sslSrvr := mrnes.CreateSrvr("sslSrvr", sslCPUType, sslcores)
		mrnes.ConnectDevs(pvtRtr, sslSrvr, true, pvtNet.Name)
		pvtNet.IncludeDev(sslSrvr, "wired", true)

		// change bridgeRtr that connects to SSL server
		bridgeRtr = mrnes.CreateRouter("pubRtr", pubRtrType)
		pubNet.IncludeDev(bridgeRtr, "wired", true)

		mrnes.ConnectDevs(sslSrvr, bridgeRtr, true, pubNet.Name)
	}

	// how many switches for direct connects to euds are needed?
	baseSwitches, excess := math.Modf(float64(euds) / float64(switchports-1))
	if excess > 0.0 {
		baseSwitches += 1
	}

	// create the switches
	eudSwitches := make([]*mrnes.SwitchFrame, 0)
	eudSwitches = append(eudSwitches, mrnes.CreateSwitch("eudSwitch-0", pubSwitchType))

	// connect eudSwitches[0] to bridgeRtr
	mrnes.ConnectDevs(bridgeRtr, eudSwitches[0], true, pubNet.Name)
	availablePorts := switchports - 1

	expandSwitchIdx := 0

	// so long as the unassigned ports on the switches in the switch tree don't accomodate all euds
	for availablePorts < euds {

		// make the switch a parent of up to switchports-1 descendent switches
		children := make([]*mrnes.SwitchFrame, 0)
		jdx := 0

		// create another if still needed and have not overflowed the paraent's capacity
		for jdx < switchports-1 && availablePorts < euds {
			nswtch := mrnes.CreateSwitch("eudswitch-"+strconv.Itoa(len(eudSwitches)+jdx), pubSwitchType)
			mrnes.ConnectDevs(nswtch, eudSwitches[expandSwitchIdx], true, pubNet.Name)
			children = append(children, nswtch)

			// availablePorts increases by the free ports of the new switch, less the parent port
			// used to connect to it
			availablePorts += (switchports - 2)
			jdx += 1
		}
		eudSwitches = append(eudSwitches, children...)
		// move to the next unparented switch
		expandSwitchIdx += 1
	}
	// create the EUDs and connect to the switches
	eudDevs := make([]*mrnes.EndptFrame, euds)
	assignTo := len(eudSwitches) - 1
	assignedThisSwitch := 0

	for jdx := 0; jdx < euds; jdx++ {
		eudDevs[jdx] = mrnes.CreateEUD("eudDev-"+strconv.Itoa(jdx), eudCPUType, eudcores)
		pubNet.IncludeDev(eudDevs[jdx], "wired", true)
		mrnes.ConnectDevs(eudDevs[jdx], eudSwitches[assignTo], true, pubNet.Name)
		assignedThisSwitch += 1
		if assignedThisSwitch == switchports-1 {
			assignedThisSwitch = 0
			assignTo -= 1
		}
	}

	// include the networks in the topo configuration
	tcf.AddNetwork(pubNet)
	tcf.AddNetwork(pvtNet)

	// fill in any missing parts needed for the topology description
	topoCfgerr := tcf.Consolidate()
	if topoCfgerr != nil {
		panic(topoCfgerr)
	}

	// turn the pointer-oriented data structures into a flat string-based
	// version for serialization, then save to file
	tc := tcf.Transform()

	tc.WriteToFile(fullpathmap["topo"])

	// create the dictionary to be populated
	expCfg := mrnes.CreateExpCfg("beta")
	mrnes.GetExpParamDesc()

	// experiment parameters are largely about architectural parameters
	// that impact performance. Define some defaults (which can be overwritten later)
	//

	// default parameters
	wcAttrbs := []mrnes.AttrbStruct{mrnes.AttrbStruct{AttrbName: "*", AttrbValue: ""}}
	expCfg.AddParameter("Interface", wcAttrbs, "delay", "1e-6")
	expCfg.AddParameter("Interface", wcAttrbs, "latency", "1e-6")

	// every network to have a latency of 1e-5
	expCfg.AddParameter("Network", wcAttrbs, "latency", "1e-4")

	// every network to have a bandwidth of 1000 (Mbits)
	expCfg.AddParameter("Network", wcAttrbs, "bandwidth", pubNetBw)

	// every interface to have a bandwidth of minimum pubNetBw and pvtNetBw
	pubNetBwFloat, _ := strconv.ParseFloat(pubNetBw, 64)
	pvtNetBwFloat, _ := strconv.ParseFloat(pvtNetBw, 64)
	minBw := pubNetBw
	if pvtNetBwFloat < pubNetBwFloat {
		minBw = pvtNetBw
	}
	expCfg.AddParameter("Interface", wcAttrbs, "bandwidth", minBw)

	// every interface to have an MTU of 1500 bytes
	expCfg.AddParameter("Interface", wcAttrbs, "MTU", "1500")

	// trace on, every device
	expCfg.AddParameter("Endpt", wcAttrbs, "trace", "true")
	expCfg.AddParameter("Switch", wcAttrbs, "trace", "false")
	expCfg.AddParameter("Router", wcAttrbs, "trace", "true")
	expCfg.AddParameter("Interface", wcAttrbs, "trace", "false")

	// endptAttrbs := []mrnes.AttrbStruct{mrnes.AttrbStruct{AttrbName: "group", AttrbValue: "EUD"}}
	// expCfg.AddParameter("Endpt", endptAttrbs, "trace", "false")

	swAttrbs := []mrnes.AttrbStruct{mrnes.AttrbStruct{AttrbName: "name", AttrbValue: "pvtSwitch"}}
	expCfg.AddParameter("Switch", swAttrbs, "trace", "true")
	swAttrbs = []mrnes.AttrbStruct{mrnes.AttrbStruct{AttrbName: "name", AttrbValue: "eudSwitch-0"}}
	expCfg.AddParameter("Switch", swAttrbs, "trace", "true")

	// parameters for individual devices.
	// interface for pcktsrc
	asv := mrnes.AttrbStruct{AttrbName: "device", AttrbValue: "pcktsrc"}
	as := []mrnes.AttrbStruct{asv}
	expCfg.AddParameter("Interface", as, "bandwidth", srcCPUBw)

	// interfaces for pubSwitch
	asv.AttrbValue = "pubSwitch"
	expCfg.AddParameter("Interface", as, "bandwidth", pubSwitchBw)

	// interfaces for pubRtr
	asv.AttrbValue = "pubRtr"
	expCfg.AddParameter("Interface", as, "bandwidth", pubRtrBw)

	// interfaces for sslSrvr (when present)
	if archType == "SSL" {
		asv.AttrbValue = "sslSrvr"
		expCfg.AddParameter("Interface", as, "bandwidth", sslCPUBw)
	}

	// interfaces for pvtSwitch
	asv.AttrbValue = "pvtSwitch"
	expCfg.AddParameter("Interface", as, "bandwidth", pvtSwitchBw)

	// interfaces for pvtRtr
	asv.AttrbValue = "pvtRtr"
	expCfg.AddParameter("Interface", as, "bandwidth", pvtRtrBw)

	// interfaces for euds
	asv.AttrbName = "group"
	asv.AttrbValue = "EUD"
	expCfg.AddParameter("Interface", as, "bandwidth", eudCPUBw)

	expCfg.WriteToFile(fullpathmap["exp"])

	// create a dictionary to hold the mappings the set of CompPatterns to the architecture
	cmpMapDict := pces.CreateCompPatternMapDict("Maps")

	// map the functions of encryptPerf
	cmpMap := pces.CreateCompPatternMap(encryptPerf.Name)

	cmpMap.AddMapping(srcFunc.Label, "pcktsrc", false)
	cmpMap.AddMapping(chgCPOutFunc.Label, "pcktsrc", false)
	cmpMap.AddMapping(finishFunc.Label, "pcktsrc", false)

	if archType != "SSL" {
		cmpMap.AddMapping(encryptOutFunc.Label, "pcktsrc", false)
		cmpMap.AddMapping(decryptRtnFunc.Label, "pcktsrc", false)
	} else {
		cmpMap.AddMapping(encryptOutFunc.Label, "sslSrvr", false)
		cmpMap.AddMapping(decryptRtnFunc.Label, "sslSrvr", false)
	}
	cmpMapDict.AddCompPatternMap(cmpMap, false)

	for ptnName := range cpDict.Patterns {
		cmpMap := pces.CreateCompPatternMap(ptnName)
		if strings.Contains(ptnName, "encryptPerf") {
			continue
		}

		// the CmpPtn name codes the idx of the EUD
		splitName := strings.Split(ptnName, "-")
		eudIdx := splitName[len(splitName)-1]

		eudDevName := "eudDev-" + eudIdx

		cmpMap.AddMapping(decryptOutFunc.Label, eudDevName, false)
		cmpMap.AddMapping(processFunc.Label, eudDevName, false)
		cmpMap.AddMapping(encryptRtnFunc.Label, eudDevName, false)
		cmpMap.AddMapping(chgCPRtnFunc.Label, eudDevName, false)

		cmpMapDict.AddCompPatternMap(cmpMap, false)
	}

	cmpMapDict.WriteToFile(fullpathmap["map"])

	// bundle up all the function timing models and write them to funcExec.yaml
	pattern := filepath.Join(funcXDir,"*.yaml")
	funcXFiles, err := filepath.Glob(pattern)

	// create a function execution list that will hold them all
	fel := pces.CreateFuncExecList("beta")

	for _, fXFile := range funcXFiles {
		var emptyBytes []byte
		felx, err := pces.ReadFuncExecList(fXFile,true,emptyBytes)
		if err != nil {
			panic(err)
		}
		for identifier := range felx.Times {
			_, present := fel.Times[identifier]
			if present {
				panic(fmt.Errorf("duplicate function identifier observed merging function execution lists"))
			}
			fel.Times[identifier] = felx.Times[identifier]
		}
	}

	// write the combined list to the directory the simulation will read
	felFile := filepath.Join(outputLib,"funcExec.yaml")
	fel.WriteToFile(felFile)

	// bundle up all the device timing models and write them to devExec.yaml
	pattern = filepath.Join(devXDir,"*.yaml")
	devXFiles, err := filepath.Glob(pattern)

	// create a function execution list that will hold them all
	del := mrnes.CreateDevExecList("beta")

	for _, dXFile := range devXFiles {
		var emptyBytes []byte
		delx, err := mrnes.ReadDevExecList(dXFile,true,emptyBytes)
		if err != nil {
			panic(err)
		}

		for identifier := range delx.Times {
			_, present := del.Times[identifier]
			if present {
				panic(fmt.Errorf("duplicate function identifier observed merging function execution lists"))
			}
			del.Times[identifier] = delx.Times[identifier]
		}
	}

	// write the combined list to the directory the simulation will read
	delFile := filepath.Join(outputLib,"devExec.yaml")
	del.WriteToFile(delFile)

	// we don't have shared state in this model but need to create an empty file
	ssgl := pces.CreateSharedStateGroupList(true) 
	ssg := pces.CreateSharedStateGroup("empty","empty")
	ssgl.AddSharedStateGroup(ssg)

	ssgl.WriteToFile(fullpathmap["srdState"])

}

// createChgCPState constructs an instance of a chgCP state
// and returns a serialized representation of it
func createChgCPState(csm map[string]pces.ChgDesc) string {
	state := pces.ClassCreateChgCP()
	state.ClassName = "chgCP"

	state.ChgMap = make(map[string]pces.ChgDesc)
	for k, v := range csm {
		state.ChgMap[k] = v
	}

	// serialize the class-dependent state structure
	serialState, err0 := state.Serialize(useYAML)
	if err0 != nil {
		panic(err0)
	}
	return serialState
}

func createProcessPcktState(cp *pces.CompPattern, node *pces.Func, opName string, rtn bool) string {
	// params := pces.CreateFuncParameters(cp.CPType, node.Label)
	state := pces.ClassCreateProcessPckt()
	state.ClassName = node.Class
	state.OpName = map[string]string{"processOp": opName}
	state.Return = rtn

	// serialize the class-dependent state structure
	serialState, err0 := state.Serialize(useYAML)
	if err0 != nil {
		panic(err0)
	}
	return serialState
}

func createCryptoPcktState(op, alg, keylength string, rtn bool) string {
	state := pces.ClassCreateCryptoPckt()
	state.ClassName = "cryptoPckt"
	state.Populate(op, alg, keylength ) 
	state.Return = rtn

	// serialize the class-dependent state structure
	serialState, err0 := state.Serialize(useYAML)
	if err0 != nil {
		panic(err0)
	}
	return serialState
}

func createFinishState() string {
	state := pces.ClassCreateFinish()
	state.ClassName = "finish"

	// serialize the class-dependent state structure
	serialState, err0 := state.Serialize(useYAML)
	if err0 != nil {
		panic(err0)
	}
	return serialState
}

