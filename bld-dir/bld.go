package main

// code to build template of MrNesbits CompPattern structures and their initialization structs.
// see adjacent README.md
import (
	"github.com/iti/cmdline"
	"github.com/iti/mrnes"
	"github.com/iti/mrnesbits"
	"path/filepath"
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
	cp.AddFlag(cmdline.StringFlag, "cp", true)        // name of output file for computation pattern
	cp.AddFlag(cmdline.StringFlag, "cpInit", true)    // name of output file for initialization blocks for CPs
	cp.AddFlag(cmdline.StringFlag, "funcExec", true)  // name of output file used for function timings
	cp.AddFlag(cmdline.StringFlag, "devExec", true)   // name of output file used for device operations timings
	cp.AddFlag(cmdline.StringFlag, "map", true)       // file with mapping of comp pattern functions to devices
	cp.AddFlag(cmdline.StringFlag, "exp", true)       // name of output file used for run-time experiment parameters
	cp.AddFlag(cmdline.StringFlag, "topo", false)     // name of output file used for topo templates
	cp.AddFlag(cmdline.BoolFlag, "useJSON", false)    // use JSON rather than YAML for serialization

	return cp
}

var useYAML bool

// main gives the entry point
func main() {
	// define the command line parameters
	cp := cmdlineParams()

	// parse the command line
	cp.Parse()

	// string for the output directory
	outputLib := cp.GetVar("outputLib").(string)

	// make sure this directory exists
	dirs := []string{outputLib}
	valid, err := mrnesbits.CheckDirectories(dirs)
	if !valid {
		panic(err)
	}

	// check for access to output files
	fullpathmap := make(map[string]string)
	outFiles := []string{"cp", "cpInit", "funcExec", "devExec", "exp", "topo", "map"}
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

	// some of the outputs are stored in 'dictionaries' that have their own
	// name, and index contents by computational pattern.  The arguments
	// to CreateXXYYZZDict are dictionary names and are ignored withing mrnesbits
	cpDict := mrnesbits.CreateCompPatternDict("alpha", true)
	cpInitDict := mrnesbits.CreateCPInitListDict("alpha", true)

	// create a CompPattern that looks like the initial BITS architecture
	encryptPerf := mrnesbits.CreateCompPattern("encryptPerf")

	// 'src' will periodically generate a packet, destination of that embedded in the msg header

	// function class "GenPckt", function name "src"
	// This function generates the first message of a train that goes to a eud
	// it has selected, stopping first at a crypto server
	//
	src := mrnesbits.CreateFunc("GenPckt", "src")
	encryptPerf.AddFunc(src)

	// function class "server", with name "crypto".   Adds delays due to encryption
	// (as a function of load) and sends encrypted packet to the destination eud
	crypto := mrnesbits.CreateFunc("CryptoSrvr", "crypto")
	encryptPerf.AddFunc(crypto)

	processFunc := mrnesbits.CreateFunc("PcktProcess", "process")
	// include the eud in the computation pattern
	encryptPerf.AddFunc(processFunc)

	// self-initiation message has type 'initiate', and also edge label 'initiate'
	encryptPerf.AddEdge(src.Label, src.Label, "initiate", "initiate")

	// src directs an edge to crypto, message type "data", edge label "crypto"
	encryptPerf.AddEdge(src.Label, crypto.Label, "plaintext_out", "crypto")

	// crypto directs an edge to src, message type "data", edge label "src"
	encryptPerf.AddEdge(crypto.Label, src.Label, "plaintext_rtn", "src")

	// crypto directs an edge to process, message type "data" and edge label "process"
	encryptPerf.AddEdge(crypto.Label, processFunc.Label, "encrypted_out", processFunc.Label)

	// process directs an edge to crypto, message type "data" and edge label "crypto"
	encryptPerf.AddEdge(processFunc.Label, crypto.Label, "encrypted_rtn", "crypto")

	// include the encryptPerf CompPattern in the library
	cpDict.AddCompPattern(encryptPerf, true, false)

	// create initialization structs for each of the encryptPerf CP functions.
	//
	// create the InEdges and OutEdges needed for function initialization
	srcFromSrcEdge := encryptPerf.GetInEdges(src.Label, src.Label)[0]
	srcToCryptoEdge := encryptPerf.GetOutEdges(src.Label, crypto.Label)[0]
	srcFromCryptoEdge := encryptPerf.GetInEdges(crypto.Label, src.Label)[0]
	cryptoFromSrcEdge := encryptPerf.GetInEdges(src.Label, crypto.Label)[0]
	cryptoToSrcEdge := encryptPerf.GetOutEdges(crypto.Label, src.Label)[0]

	cryptoToProcessEdge := encryptPerf.GetOutEdges(crypto.Label, processFunc.Label)[0]
	cryptoFromProcessEdge := encryptPerf.GetInEdges(processFunc.Label, crypto.Label)[0]

	processToCryptoEdge := encryptPerf.GetOutEdges(processFunc.Label, crypto.Label)[0]
	processFromCryptoEdge := encryptPerf.GetInEdges(crypto.Label, processFunc.Label)[0]

	// we have the pieces now to put initializations together
	epCPInit := mrnesbits.CreateCPInitList("encryptPerf", "encryptPerf", true)

	// flesh out the messages. Each type has a packetlength of 1000 bytes and a message length of
	// 1500 bytes
	epInitMsg := mrnesbits.CreateCompPatternMsg("initiate", 256, 1500)
	epDataMsg := mrnesbits.CreateCompPatternMsg("data", 256, 1500)

	// remember these message specifications
	epCPInit.AddMsg(epInitMsg)
	epCPInit.AddMsg(epDataMsg)

	gblName := "encryptPerf+" + src.Label
	srcParams := mrnesbits.CreateFuncParameters(encryptPerf.CPType, src.Label, gblName)
	srcState := mrnesbits.ClassCreateGenPckt()
	// FIND come back to this
	srcState.Populate(2.0, 1, "exp", "data", 1500, 250, 0)

	// serialize the class-dependent state structure
	serialSrcState, err0 := srcState.Serialize(useYAML)
	if err0 != nil {
		panic(err0)
	}
	srcParams.AddState(serialSrcState)

	// create the container for all the parameters to be held by the src function.
	// constructor needs to know the type of the computational pattern, the label of the function,
	// and the initial state of the function

	// add the responses that may happen at src.  Express
	//  - source func and message type defining response.  Either may be wild-cards
	//  - destination func and message type for response.  Either may be wild-cards
	//  - the MethodOp code selected
	srcParams.AddResponse(srcFromSrcEdge, srcToCryptoEdge, "initiate-op")

	srcParams.AddResponse(srcFromCryptoEdge, mrnesbits.EmptyOutEdge, "return-op")

	// bundle it up
	srcParamStr, _ := srcParams.Serialize(useYAML)
	epCPInit.AddParam(src.Label, srcParamStr)

	gblName = "encryptPerf+" + crypto.Label
	cryptoParams := mrnesbits.CreateFuncParameters(encryptPerf.CPType, crypto.Label, gblName)

	//   maximum number of concurrent requests (100) and crypto algorithm ("aes")
	cryptoState := mrnesbits.ClassCreateCryptoSrvr()
	cryptoState.Populate(100, "rsa-3072", "")
	serialCryptoState, err1 := cryptoState.Serialize(useYAML)
	if err1 != nil {
		panic(err1)
	}
	cryptoParams.AddState(serialCryptoState)

	// serialize initialization
	// add the branching responses that may happen at crypto

	cryptoParams.AddResponse(cryptoFromSrcEdge, cryptoToProcessEdge, "encrypt-op")

	// response choice helps identify which response to choose, given the dst selected before
	cryptoParams.AddResponseChoice(cryptoFromSrcEdge, cryptoToProcessEdge, processFunc.Label)

	// no response choice needed for this direction
	cryptoParams.AddResponse(cryptoFromProcessEdge, cryptoToSrcEdge, "decrypt-op")

	cryptoParamStr, _ := cryptoParams.Serialize(useYAML)
	epCPInit.AddParam(crypto.Label, cryptoParamStr)

	processState := mrnesbits.ClassCreatePcktProcess()
	processState.Populate("rsa-3072")
	serialPcktProcessState, err1 := processState.Serialize(useYAML)
	if err1 != nil {
		panic(err1)
	}

	gblName = "EncryptPerf+" + processFunc.Label
	sparams := mrnesbits.CreateFuncParameters(encryptPerf.CPType, processFunc.Label, gblName)
	sparams.AddState(serialPcktProcessState)
	sparams.AddResponse(processFromCryptoEdge, processToCryptoEdge, "process-op")
	sparamStr, _ := sparams.Serialize(useYAML)
	epCPInit.AddParam(processFunc.Label, sparamStr)

	// include this CompPattern's initialization information into the CP initialization dictionary
	cpInitDict.AddCPInitList(epCPInit, false)

	fel := mrnesbits.CreateFuncExecList("BITS-1")
	fel.AddTiming("process-op", "x86", 1000, 5e-2)
	fel.AddTiming("process-op", "M1", 1000, 2e-2)
	fel.AddTiming("process-op", "M2", 1000, 5e-3)

	fel.AddTiming("process-op", "i7", 64, 1e-4)
	fel.AddTiming("process-op", "i7", 256, 1e-4)
	fel.AddTiming("process-op", "i5", 64, 2e-4)
	fel.AddTiming("process-op", "i5", 256, 2e-4)
	fel.AddTiming("process-op", "i3", 64, 3e-4)
	fel.AddTiming("process-op", "i3", 256, 3e-4)
	fel.AddTiming("process-op", "arm", 64, 4e-4)
	fel.AddTiming("process-op", "arm", 256, 4e-4)

	fel.AddTiming("generate-op", "x86", 1000, 5e-2)
	fel.AddTiming("generate-op", "M1", 1000, 2e-2)
	fel.AddTiming("generate-op", "M2", 1000, 5e-3)

	fel.AddTiming("generate-op", "i7", 64, 1e-4)
	fel.AddTiming("generate-op", "i7", 256, 1e-4)
	fel.AddTiming("generate-op", "i5", 64, 2e-4)
	fel.AddTiming("generate-op", "i5", 256, 2e-4)
	fel.AddTiming("generate-op", "i3", 64, 3e-4)
	fel.AddTiming("generate-op", "i3", 256, 3e-4)
	fel.AddTiming("generate-op", "arm", 64, 4e-4)
	fel.AddTiming("generate-op", "arm", 256, 4e-4)

	// include the timings for the operations just specified
	fel.AddTiming("select-op", "x86", 1000, 1e-4)
	fel.AddTiming("select-op", "M1", 1000, 5e-5)
	fel.AddTiming("select-op", "M2", 1000, 1e-5)
	fel.AddTiming("select-op", "i7", 256, 1e-5)
	fel.AddTiming("select-op", "i5", 256, 2e-5)
	fel.AddTiming("select-op", "i3", 256, 3e-5)
	fel.AddTiming("select-op", "arm", 256, 4e-5)

	fel.AddTiming("genflow-op", "x86", 1000, 1e-8)
	fel.AddTiming("genflow-op", "M1", 1000, 5e-8)
	fel.AddTiming("genflow-op", "M2", 1000, 1e-8)
	fel.AddTiming("genflow-op", "i7", 256, 1e-8)
	fel.AddTiming("genflow-op", "i5", 256, 2e-8)
	fel.AddTiming("genflow-op", "i3", 256, 3e-8)
	fel.AddTiming("genflow-op", "arm", 256, 4e-8)

	fel.AddTiming("return-op", "i7", 256, 1e-5)
	fel.AddTiming("return-op", "i5", 256, 3e-5)
	fel.AddTiming("return-op", "i3", 256, 6e-5)

	// include the timings for the operations just specified
	fel.AddTiming("return-op", "x86", 1000, 0.0)
	fel.AddTiming("return-op", "M1", 1000, 0.0)
	fel.AddTiming("return-op", "M2", 1000, 0.0)

	// include the timings for the operations just specified
	fel.AddTiming("decrypt-aes", "x86", 1000, 1e-4)
	fel.AddTiming("decrypt-aes", "accel-x86", 1000, 1e-5)
	fel.AddTiming("decrypt-aes", "M1", 1000, 5e-5)
	fel.AddTiming("decrypt-aes", "M2", 1000, 1e-5)

	// i7 aes.  64, 128, 256, 512 lengths
	fel.AddTiming("encrypt-aes", "i7", 64, 8.93e-06)
	fel.AddTiming("decrypt-aes", "i7", 64, 1.88e-05)
	fel.AddTiming("encrypt-aes", "i7", 128, 8.99e-06)
	fel.AddTiming("decrypt-aes", "i7", 128, 1.91e-05)
	fel.AddTiming("encrypt-aes", "i7", 256, 9.11e-06)
	fel.AddTiming("decrypt-aes", "i7", 256, 1.92e-05)
	fel.AddTiming("encrypt-aes", "i7", 512, 1.01e-05)
	fel.AddTiming("decrypt-aes", "i7", 512, 2.04e-05)

	// i3 aes.  64, 128, 256, 512 lengths.   Made up numbers
	fel.AddTiming("encrypt-aes", "i3", 64, 8.93e-05)
	fel.AddTiming("decrypt-aes", "i3", 64, 1.88e-04)
	fel.AddTiming("encrypt-aes", "i3", 128, 8.99e-05)
	fel.AddTiming("decrypt-aes", "i3", 128, 1.91e-04)
	fel.AddTiming("encrypt-aes", "i3", 256, 9.11e-05)
	fel.AddTiming("decrypt-aes", "i3", 256, 1.92e-04)
	fel.AddTiming("encrypt-aes", "i3", 512, 1.01e-04)
	fel.AddTiming("decrypt-aes", "i3", 512, 2.04e-04)

	// x86 aes.  64, 128, 256, 512 lengths.   Made up numbers
	fel.AddTiming("encrypt-aes", "i3", 64, 9.93e-05)
	fel.AddTiming("decrypt-aes", "i3", 64, 2.88e-04)
	fel.AddTiming("encrypt-aes", "i3", 128, 9.99e-05)
	fel.AddTiming("decrypt-aes", "i3", 128, 2.91e-04)
	fel.AddTiming("encrypt-aes", "i3", 256, 1.11e-04)
	fel.AddTiming("decrypt-aes", "i3", 256, 2.92e-04)
	fel.AddTiming("encrypt-aes", "i3", 512, 2.01e-04)
	fel.AddTiming("decrypt-aes", "i3", 512, 3.04e-04)

	// i7 none. 256 length
	fel.AddTiming("encrypt-none", "i7", 256, 2.0e-06)
	fel.AddTiming("decrypt-none", "i7", 256, 2.0e-05)

	// i3 none. 256 length
	fel.AddTiming("encrypt-none", "i3", 256, 2.0e-06)
	fel.AddTiming("decrypt-none", "i3", 256, 2.0e-05)

	// x86 none. 256 length
	fel.AddTiming("encrypt-none", "x86", 256, 2.0e-06)
	fel.AddTiming("decrypt-none", "x86", 256, 2.0e-05)

	// i7 rsa-3072. 64, 128, 256 packet lengths
	fel.AddTiming("encrypt-rsa-3072", "i7", 64, 5.63e-04)
	fel.AddTiming("decrypt-rsa-3072", "i7", 64, 3.33e-03)
	fel.AddTiming("encrypt-rsa-3072", "i7", 128, 5.51e-04)
	fel.AddTiming("decrypt-rsa-3072", "i7", 128, 3.33e-03)
	fel.AddTiming("encrypt-rsa-3072", "i7", 256, 5.63e-04)
	fel.AddTiming("decrypt-rsa-3072", "i7", 256, 3.32e-03)

	// i3 rsa-3072. 64, 128, 256 packet lengths
	fel.AddTiming("encrypt-rsa-3072", "i3", 64, 5.63e-03)
	fel.AddTiming("decrypt-rsa-3072", "i3", 64, 3.33e-02)
	fel.AddTiming("encrypt-rsa-3072", "i3", 128, 5.51e-03)
	fel.AddTiming("decrypt-rsa-3072", "i3", 128, 3.33e-02)
	fel.AddTiming("encrypt-rsa-3072", "i3", 256, 5.63e-03)
	fel.AddTiming("decrypt-rsa-3072", "i3", 256, 3.32e-02)

	// x86 rsa-3072. 64, 128, 256 packet lengths
	fel.AddTiming("encrypt-rsa-3072", "x86", 64, 7.63e-03)
	fel.AddTiming("decrypt-rsa-3072", "x86", 64, 5.33e-02)
	fel.AddTiming("encrypt-rsa-3072", "x86", 128, 7.51e-03)
	fel.AddTiming("decrypt-rsa-3072", "x86", 128, 5.33e-02)
	fel.AddTiming("encrypt-rsa-3072", "x86", 256, 7.63e-03)
	fel.AddTiming("decrypt-rsa-3072", "x86", 256, 3.32e-02)

	fel.AddTiming("decrypt-rsa", "x86", 1000, 0.33)
	fel.AddTiming("decrypt-rsa", "M1", 1000, 0.33)
	fel.AddTiming("decrypt-rsa", "M2", 1000, 0.33)
	fel.AddTiming("decrypt-rsa", "accel-x86", 1000, 0.15)

	// include the timings for the operations just specified
	fel.AddTiming("encrypt-aes", "x86", 1000, 1e-3)
	fel.AddTiming("encrypt-aes", "accel-x86", 1000, 1e-3)
	fel.AddTiming("encrypt-aes", "M1", 1000, 5e-4)
	fel.AddTiming("encrypt-aes", "M2", 1000, 1e-5)
	fel.AddTiming("encrypt-rsa", "x86", 1000, 0.33)
	fel.AddTiming("encrypt-rsa", "M1", 1000, 0.33)
	fel.AddTiming("encrypt-rsa", "M2", 1000, 0.33)

	// write the comp pattern stuff out
	cpDict.WriteToFile(fullpathmap["cp"])
	cpInitDict.WriteToFile(fullpathmap["cpInit"])

	// build a topology named 'Protected' with two networks
	tcf := mrnes.CreateTopoCfgFrame("Protected")

	// the two networks
	pvtNet := mrnes.CreateNetwork("private", "LAN", "wired")
	pubNet := mrnes.CreateNetwork("public", "LAN", "wired")

	// create a source node in pvtnet
	srcEUD := mrnes.CreateEUD("pcktsrc", "i3", 1)

	// create a switch and connect the pcktsrc to it
	pvtSwitch := mrnes.CreateSwitch("pvtSwitch", "cisco")
	mrnes.ConnectDevs(srcEUD, pvtSwitch, true, pvtNet.Name)
	pvtNet.IncludeDev(pvtSwitch, "wired", true)

	// create a router and connect the switch to it
	pvtRtr := mrnes.CreateRouter("pvtRtr", "cisco")
	mrnes.ConnectDevs(pvtSwitch, pvtRtr, true, pvtNet.Name)
	pvtNet.IncludeDev(pvtRtr, "wired", true)

	// create an SSL filter that joins pvtNet and pubNet
	sslFilter := mrnes.CreateFilter("ssl", "Filter", "crypto")
	mrnes.ConnectDevs(pvtRtr, sslFilter, true, pvtNet.Name)
	pvtNet.IncludeDev(sslFilter, "wired", true)

	// create a router to sit in the private network between the ssl and network switch
	pubRtr := mrnes.CreateRouter("pubRtr", "cisco")
	pubNet.IncludeDev(pubRtr, "wired", true)

	// put the router in group "interior"
	mrnes.ConnectDevs(sslFilter, pubRtr, true, pubNet.Name)

	// create a switch for pubNet and connect to the router
	pubSwitch := mrnes.CreateSwitch("pubSwitch", "cisco")
	mrnes.ConnectDevs(pubSwitch, pubRtr, true, pubNet.Name)

	// create the eud, add to the network, connect to switch
	eud := mrnes.CreateEUD("eudDev", "i3", 1)
	pubNet.IncludeDev(eud, "wired", true)
	mrnes.ConnectDevs(pubSwitch, eud, true, pubNet.Name)

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

	fel.WriteToFile(fullpathmap["funcExec"])

	// build a table of device timing parameters
	dev := mrnes.CreateDevExecList("BITS-1")
	// dev.AddTiming("switch", "Slow", 1e-4)
	dev.AddTiming("switch", "FS_S3900-48T6S-R", 1e-5)
	// dev.AddTiming("route", "Slow", 5e-4)
	// dev.AddTiming("route", "Fast", 1e-4)
	dev.AddTiming("route", "Juniper_MX_240", 5e-4)
	dev.AddTiming("route", "Cisco_Catalyst_8200", 1e-4)

	dev.WriteToFile(fullpathmap["devExec"])

	// create the dictionary to be populated
	expCfg := mrnes.CreateExpCfg("alpha")
	mrnes.GetExpParamDesc()

	// experiment parameters are largely about architectural parameters
	// that impact performance. Define some defaults (which can be overwritten later)
	//

	// default wired interface parameters
	wcAttrbs := []mrnes.AttrbStruct{mrnes.AttrbStruct{AttrbName: "*", AttrbValue: ""}}
	expCfg.AddParameter("Interface", wcAttrbs, "delay", "1e-6")
	expCfg.AddParameter("Interface", wcAttrbs, "latency", "1e-5")

	// every network to have a latency of 1e-5
	expCfg.AddParameter("Network", wcAttrbs, "latency", "1e-4")

	// every network to have a bandwidth of 1000 (Mbits)
	expCfg.AddParameter("Network", wcAttrbs, "bandwidth", "1000")

	// every wireless network to have bandwidth of 1000Mbits
	expCfg.AddParameter("Network", wcAttrbs, "bandwidth", "1000")

	// every interface to have a bandwidth of 100 Mbits
	expCfg.AddParameter("Interface", wcAttrbs, "bandwidth", "100")

	// every interface to have an MTU of 1500 bytes
	expCfg.AddParameter("Interface", wcAttrbs, "MTU", "1500")

	// both endpoints to have an i5 CPU
	expCfg.AddParameter("Endpt", wcAttrbs, "CPU", "i5")

	// trace on, every device
	expCfg.AddParameter("Endpt", wcAttrbs, "trace", "true")
	// expCfg.AddParameter("Switch", wcAttrbs, "model", "Fast")
	expCfg.AddParameter("Switch", wcAttrbs, "model", "FS_S3900-48T6S-R")

	expCfg.AddParameter("Switch", wcAttrbs, "trace", "true")
	expCfg.AddParameter("Router", wcAttrbs, "trace", "true")
	expCfg.AddParameter("Filter", wcAttrbs, "trace", "true")

	asv := mrnes.AttrbStruct{AttrbName: "device", AttrbValue: "pcktsrc"}
	as := []mrnes.AttrbStruct{asv}
	expCfg.AddParameter("Interface", as, "bandwidth", "0.01")
	asv.AttrbValue = "pubSwitch"
	expCfg.AddParameter("Interface", as, "bandwidth", "0.01")
	asv.AttrbValue = "pubRtr"
	expCfg.AddParameter("Interface", as, "bandwidth", "0.01")
	asv.AttrbValue = "ssl"
	expCfg.AddParameter("Interface", as, "bandwidth", "0.01")
	asv.AttrbValue = "pvtSwitch"
	expCfg.AddParameter("Interface", as, "bandwidth", "0.01")
	asv.AttrbValue = "pvtRtr"
	expCfg.AddParameter("Interface", as, "bandwidth", "0.01")
	asv.AttrbValue = "eudDev"
	expCfg.AddParameter("Interface", as, "bandwidth", "0.01")

	// every router, switch to have model "Slow"
	// expCfg.AddParameter("Router", wcAttrbs, "model", "Slow")
	expCfg.AddParameter("Router", wcAttrbs, "model", "Juniper_MX_240")
	// expCfg.AddParameter("Switch", wcAttrbs, "model", "Slow")
	expCfg.AddParameter("Switch", wcAttrbs, "model", "FS_N8560-32C")

	expCfg.WriteToFile(fullpathmap["exp"])

	cmpMapDict := mrnesbits.CreateCompPatternMapDict("Maps")
	cmpMap := mrnesbits.CreateCompPatternMap("encryptPerf")
	cmpMap.AddMapping("src", "pcktsrc", false)
	cmpMap.AddMapping("crypto", "ssl", false)
	cmpMap.AddMapping(processFunc.Label, "eudDev", false)

	cmpMapDict.AddCompPatternMap(cmpMap, false)
	flowMap := mrnesbits.CreateCompPatternMap("pubTraffic")
	flowMap.AddMapping("flowSrc", "flowSrcDev", false)
	flowMap.AddMapping("flowSink", "flowSinkDev", false)
	cmpMapDict.AddCompPatternMap(flowMap, false)

	cmpMapDict.WriteToFile(fullpathmap["map"])
}
