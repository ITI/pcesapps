package main

// code to build template of MrNesbits CompPattern structures and their initialization structs.
// see adjacent README.md
import (
	"fmt"
	"github.com/iti/cmdline"
	"github.com/iti/mrnes"
	"github.com/iti/mrnesbits"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

// cmdlineParams defines the parameters recognized
// on the command line
func cmdlineParams() *cmdline.CmdParser {
	// command line parameters are all about file and directory locations.
	// Even though we don't need the flags for the other MrNesbits structures we
	// keep them here so that all the programs that build templates can use the same arguments file
	// create an argument parser

	cp := cmdline.NewCmdParser()
	cp.AddFlag(cmdline.StringFlag, "outputLib", true) // directory where model is written
	cp.AddFlag(cmdline.StringFlag, "db", true)        // directory where "database" is kept
	cp.AddFlag(cmdline.StringFlag, "cp", true)        // name of output file for computation pattern
	cp.AddFlag(cmdline.StringFlag, "cpInit", true)    // name of output file for initialization blocks for CPs
	cp.AddFlag(cmdline.StringFlag, "funcExec", true)  // name of output file used for function timings
	cp.AddFlag(cmdline.StringFlag, "devExec", true)   // name of output file used for device operations timings
	cp.AddFlag(cmdline.StringFlag, "devDesc", true)   // name of input file
	cp.AddFlag(cmdline.StringFlag, "srdState", false) // name of output file holding shared state description
	cp.AddFlag(cmdline.StringFlag, "map", true)       // file with mapping of comp pattern functions to devices
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

	// make sure this directory exists
	dirs := []string{dbLib, outputLib, funcXDir, devXDir}
	valid, err := mrnesbits.CheckDirectories(dirs)
	if !valid {
		panic(err)
	}

	// name of the file with hardware description dictionary
	devDescName := cp.GetVar("devDesc").(string)

	// make sure that devDesc exists
	devDescFile := filepath.Join(outputLib, devDescName)
	_, errc := mrnesbits.CheckFiles([]string{devDescFile}, true)
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
	outFiles := []string{"cp", "cpInit", "devExec", "funcExec", "exp", "topo", "map", "srdState"}
	fullpath := []string{}
	for _, filename := range outFiles {
		basefile := cp.GetVar(filename).(string)
		fullfile := filepath.Join(outputLib, basefile)
		fullpath = append(fullpath, fullfile)
		fullpathmap[filename] = fullfile
	}

	valid, err = mrnesbits.CheckOutputFiles(fullpath)
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

	// create a CompPattern modeling generation of a packet, followed by encryption, decryption, processing,
	// encryption, decryption, and completions
	cmpPtnType := "encryptPerf-" + archType

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
	// create a computational pattern data structure
	encryptPerf := mrnesbits.CreateCompPattern(cmpPtnType)
	
	srcFunc := mrnesbits.CreateFunc("connSrc", "src")
	encryptOutFunc := mrnesbits.CreateFunc("processPckt", "encryptOut")
	decryptOutFunc := mrnesbits.CreateFunc("processPckt", "decryptOut")
	processFunc := mrnesbits.CreateFunc("processPckt", "process")
	encryptRtnFunc := mrnesbits.CreateFunc("processPckt", "encryptRtn")
	decryptRtnFunc := mrnesbits.CreateFunc("processPckt", "decryptRtn")

	// add these to the computational pattern
	encryptPerf.AddFunc(srcFunc)
	encryptPerf.AddFunc(encryptOutFunc)
	encryptPerf.AddFunc(decryptOutFunc)
	encryptPerf.AddFunc(processFunc)
	encryptPerf.AddFunc(encryptRtnFunc)
	encryptPerf.AddFunc(decryptRtnFunc)

	// create a CPInit structure that will serve as the template for each
	// individual CompPattern to be created.   We can leave the name of the CompPattern
	// empty but state the type
	epCPInit := mrnesbits.CreateCPInitList("", cmpPtnType, true)

	// flesh out the messages.
	msgLen := pcktSize + 36
	epCPInit.AddMsg(mrnesbits.CreateCompPatternMsg("initiate", pcktSize, msgLen))
	epCPInit.AddMsg(mrnesbits.CreateCompPatternMsg("plaintext", pcktSize, msgLen))
	epCPInit.AddMsg(mrnesbits.CreateCompPatternMsg("encryptext", pcktSize, msgLen))

	// add edges
	// self-initiation message has type 'initiate'
	encryptPerf.AddEdge(srcFunc.Label, srcFunc.Label, "initiate", "generateOp", &epCPInit.Msgs)

	// chain out and back
	encryptPerf.AddEdge(srcFunc.Label, encryptOutFunc.Label, "plaintext", "processOp", &epCPInit.Msgs)
	encryptPerf.AddEdge(encryptOutFunc.Label, decryptOutFunc.Label, "encryptext", "processOp", &epCPInit.Msgs)
	encryptPerf.AddEdge(decryptOutFunc.Label, processFunc.Label, "plaintext", "processOp", &epCPInit.Msgs)
	encryptPerf.AddEdge(processFunc.Label, encryptRtnFunc.Label, "plaintext", "processOp", &epCPInit.Msgs)
	encryptPerf.AddEdge(encryptRtnFunc.Label, decryptRtnFunc.Label, "encryptext", "processOp", &epCPInit.Msgs)
	encryptPerf.AddEdge(decryptRtnFunc.Label, srcFunc.Label, "plaintext", "completeOp", &epCPInit.Msgs)

	// create edge variables, follow sequence from src back and save that sequence
	nodes := []*mrnesbits.Func{srcFunc, encryptOutFunc, decryptOutFunc, processFunc,
		encryptRtnFunc, decryptRtnFunc}

	// put in parameters for srcFunc node (n0)
	// srcParams := mrnesbits.CreateFuncParameters(encryptPerf.CPType, srcFunc.Label)
	srcState := mrnesbits.ClassCreateConnSrc()

	// each source has the same mean packet inter-arrival time, one round-trip packet,
	// exponential distribution between packets, and a message lenght and packet size determined by input parameters

	if pcktMu > 0.0 {
		srcState.Populate(1.0/pcktMu, pcktBurst, "exp", "plaintext", msgLen, pcktSize, 0, false)
	} else {
		srcState.Populate(0.0, pcktBurst, "const", "plaintext", msgLen, pcktSize, 0, false)
	}

	srcState.AddOpName("generateOp", "generateOp")
	srcState.AddOpName("completeOp", "completeOp")

	// serialize the class-dependent state structure
	serialSrcState, err0 := srcState.Serialize(useYAML)
	if err0 != nil {
		panic(err0)
	}
	epCPInit.AddState(encryptPerf, srcFunc, serialSrcState)

	// put in parameters for encryptOutFunc (n1)
	n1StateStr := createCryptoPcktState("encrypt", cryptoAlg, keyLength, false)
	epCPInit.AddState(encryptPerf, nodes[1], n1StateStr)

	// put in parameters for decryptOutFunc (n2)
	n2StateStr := createCryptoPcktState("decrypt", cryptoAlg, keyLength, false)
	epCPInit.AddState(encryptPerf, nodes[2], n2StateStr)

	// put in parameters for processFunc (n3)
	n3StateStr := createProcessPcktState(encryptPerf, nodes[3], "processOp")
	epCPInit.AddState(encryptPerf, nodes[3], n3StateStr)

	// put in parameters for encryptRtnFunc (n4)
	n4StateStr := createCryptoPcktState("encrypt", cryptoAlg, keyLength, false)
	epCPInit.AddState(encryptPerf, nodes[4], n4StateStr)

	// put in parameters for decryptRtnFunc (n5)
	n5StateStr := createCryptoPcktState("decrypt", cryptoAlg, keyLength, false)
	epCPInit.AddState(encryptPerf, nodes[5], n5StateStr)

	cpDict := mrnesbits.CreateCompPatternDict("beta")
	cpInitDict := mrnesbits.CreateCPInitListDict("beta")

	// make a unique comp pattern instance for every eud
	for idx := 0; idx < euds; idx++ {

		eudIdx := strconv.Itoa(idx)
		// create a copy from the template
		cpyCP := encryptPerf.DeepCopy()

		// give it a unique name
		cpyCP.Name = cpyCP.CPType + "-" + eudIdx

		// save it in the dictionary
		cpDict.AddCompPattern(cpyCP)

		// create a copy of CPInit
		cpyCPInitList := epCPInit.DeepCopy()

		// give it a unique name
		cpyCPInitList.Name = cpyCPInitList.CPType + "-" + eudIdx

		// save it in the dictionary
		cpInitDict.AddCPInitList(cpyCPInitList)
	}

	// write the comp pattern stuff out
	cpDict.WriteToFile(fullpathmap["cp"])
	cpInitDict.WriteToFile(fullpathmap["cpInit"])

	// build a topology named with two networks
	tcf := mrnes.CreateTopoCfgFrame("EvaluateCrypto")

	// the two networks
	pvtNet := mrnes.CreateNetwork("private", "LAN", "wired")
	pubNet := mrnes.CreateNetwork("public", "LAN", "wired")

	// create a source node in pvtnet.  pcktsrc device will hold "src" function
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

	// map the set of CompPatterns to the architecture
	cmpMapDict := mrnesbits.CreateCompPatternMapDict("Maps")
	for ptnName := range cpDict.Patterns {
		cmpMap := mrnesbits.CreateCompPatternMap(ptnName)
		// the comp pattern name codes the idx of the EUD
		splitName := strings.Split(ptnName, "-")
		eudIdx := splitName[len(splitName)-1]

		cmpMap.AddMapping(nodes[0].Label, "pcktsrc", false)
		if archType != "SSL" {
			cmpMap.AddMapping(nodes[1].Label, "pcktsrc", false)
		} else {
			cmpMap.AddMapping(nodes[1].Label, "sslSrvr", false)
		}
		eudDevName := "eudDev-" + eudIdx

		// decrypt happens on the EUD
		cmpMap.AddMapping(nodes[2].Label, eudDevName, false)

		// processing happens on the EUD
		cmpMap.AddMapping(nodes[3].Label, eudDevName, false)

		// re-encryption happens on the EUD
		cmpMap.AddMapping(nodes[4].Label, eudDevName, false)

		// location of decryption depends on the architecture type
		if archType == "SSL" {
			cmpMap.AddMapping(nodes[5].Label, "sslSrvr", false)
		} else {
			cmpMap.AddMapping(nodes[5].Label, "pcktsrc", false)
		}
		cmpMapDict.AddCompPatternMap(cmpMap, false)
	}

	cmpMapDict.WriteToFile(fullpathmap["map"])

	// bundle up all the function timing models and write them to funcExec.yaml
	pattern := filepath.Join(funcXDir,"*.yaml")
	funcXFiles, err := filepath.Glob(pattern)

	// create a function execution list that will hold them all
	fel := mrnesbits.CreateFuncExecList("beta")

	for _, fXFile := range funcXFiles {
		var emptyBytes []byte
		felx, err := mrnesbits.ReadFuncExecList(fXFile,true,emptyBytes)
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

}

func createProcessPcktState(cp *mrnesbits.CompPattern, node *mrnesbits.Func, opName string) string {
	// params := mrnesbits.CreateFuncParameters(cp.CPType, node.Label)
	state := mrnesbits.ClassCreateProcessPckt()
	state.ClassName = node.Class
	state.OpName = map[string]string{"processOp": opName}

	// serialize the class-dependent state structure
	serialState, err0 := state.Serialize(useYAML)
	if err0 != nil {
		panic(err0)
	}
	return serialState
}

func createCryptoPcktState(op, alg, keylength string, rtn bool) string {
	state := mrnesbits.ClassCreateCryptoPckt()
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

