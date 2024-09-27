package main

// code to build application to develop performance signature of network

import (
	"fmt"
	"github.com/iti/mrnes"
	"github.com/iti/cmdline"
	"github.com/iti/pces"
	"github.com/iti/probe"
	"path/filepath"
	"strconv"
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
	cp.AddFlag(cmdline.StringFlag, "srdCfg", false) // name of output file holding shared cfg description
	cp.AddFlag(cmdline.StringFlag, "map", true)       // file with mapping of CmpPtn functions to devices
	cp.AddFlag(cmdline.StringFlag, "exp", true)       // name of output file used for run-time experiment parameters
	cp.AddFlag(cmdline.StringFlag, "topo", false)     // name of output file used for topo templates
	cp.AddFlag(cmdline.BoolFlag,   "useJSON", false)  // use JSON rather than YAML for serialization

	cp.AddFlag(cmdline.IntFlag,     "endpts", true)       // number of endpoints per network
	cp.AddFlag(cmdline.IntFlag,     "networks", true)
	cp.AddFlag(cmdline.StringFlag,  "cpumodel", true)		  // model of cpu on all the endpoints	
	cp.AddFlag(cmdline.StringFlag,  "rtrmodel", true)		  // model of cpu on all the endpoints	
	cp.AddFlag(cmdline.StringFlag,  "switchmodel", true)		  // model of cpu on all the endpoints	

	// performance parameters
	cp.AddFlag(cmdline.StringFlag,   "endptbw", true)		// endpoint intrfc bandwidth
	cp.AddFlag(cmdline.StringFlag,   "rtrbw", true)			// router intrfc bandwidth
	cp.AddFlag(cmdline.StringFlag,   "switchbw", true)		// router intrfc bandwidth
	cp.AddFlag(cmdline.StringFlag,   "intrfcbfr", true)	    // buffer size of interfaces
	cp.AddFlag(cmdline.StringFlag,   "intrfcdelay", true)   // latency through interface
	cp.AddFlag(cmdline.StringFlag,   "netcapacity", true)	// carrying capacity of network
	cp.AddFlag(cmdline.StringFlag,   "netlatency", true)	// latency of networks

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
	outFiles := []string{"cp", "cpInit", "funcExec", "devExec", "exp", "topo", "map", "srdCfg"}
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

	// read in parameters describing network
	endPtsPerNet := cp.GetVar("endpts").(int)
	numNets := cp.GetVar("networks").(int)
	cpumodel := cp.GetVar("cpumodel").(string)
	rtrmodel := cp.GetVar("rtrmodel").(string)
	switchmodel := cp.GetVar("switchmodel").(string)

	endptbw := cp.GetVar("endptbw").(string)
	rtrbw   := cp.GetVar("rtrbw").(string)
	switchbw := cp.GetVar("switchbw").(string)
	intrfcbfr := cp.GetVar("intrfcbfr").(string)
	intrfcdelay := cp.GetVar("intrfcdelay").(string)
	netcapacity := cp.GetVar("netcapacity").(string)
	netlatency  := cp.GetVar("netlatency").(string)

	// build topology
	tcf := mrnes.CreateTopoCfgFrame("Eval")
	cmpMapDict := pces.CreateCompPatternMapDict("Maps")

	baseName := "net-"
	networks := make([]*mrnes.NetworkFrame, numNets)

	for idx:=0; idx<numNets; idx++ {
		netIdxStr := strconv.Itoa(idx)
		netName := baseName + netIdxStr

		// create the network
		networks[idx] = mrnes.CreateNetwork(netName, "WAN", "wired")
		tcf.AddNetwork(networks[idx])

		// create a switch within the network
		switchName := "switch-"+netIdxStr
		swtch := mrnes.CreateSwitch(switchName, switchmodel)
		networks[idx].IncludeDev(swtch, "wired", true)
		tcf.AddSwitch(swtch)

		// add endpoints
		for jdx:= 0; jdx< endPtsPerNet; jdx++ {
			endptIdxStr := strconv.Itoa(jdx)
			endptname := "endpt-"+netIdxStr+"-"+endptIdxStr
			var endpt *mrnes.EndptFrame = nil
			if jdx == 0 {
				// don't make the first endpoint a host
				endpt = mrnes.CreateNode(endptname, cpumodel, 1)
			} else {
				endpt = mrnes.CreateHost(endptname, cpumodel, 1)
			}
			tcf.AddEndpt(endpt)

			// add the host to the network
			networks[idx].IncludeDev(endpt, "wired", true)
			mrnes.ConnectDevs(endpt, swtch, false, netName)	
		}

		// add a router
		rtrname := "rtr-"+netIdxStr
		rtr := mrnes.CreateRouter(rtrname, rtrmodel)
		tcf.AddRouter(rtr)
	
		// include the router in both networks[idx] and networks[idx-1]
		networks[idx].IncludeDev(rtr, "wired", true)
		if idx > 0 {
			networks[idx].IncludeDev(networks[idx-1].Routers[0], "wired", true)
		} 

		if idx == numNets-1 {
			networks[0].IncludeDev(rtr, "wired", true)
		}	
	}

	// for each network connect its two routers directly, and connect each to the networks's switch
	for idx:=0; idx<numNets; idx++ {
		net := networks[idx]
		mrnes.ConnectDevs(net.Routers[0], net.Routers[1], false, net.Name)
		mrnes.ConnectDevs(net.Routers[0], net.Switches[0], false, net.Name)
		mrnes.ConnectDevs(net.Routers[1], net.Switches[0], false, net.Name)
	}

	tc := tcf.Transform()
	tc.WriteToFile(fullpathmap["topo"])

	cmpPtnDict := pces.CreateCompPatternDict("Evaluation")
	cpInitListDict := pces.CreateCPInitListDict("Evaluation")

	probeCmpPtnType := "probe"

	// create the comp patterns at all the endpoints
	for idx:=0; idx<numNets; idx++ {
		netIdxStr := strconv.Itoa(idx)

		for jdx:=0; jdx<endPtsPerNet; jdx++ {
			endptIdxStr := strconv.Itoa(jdx)

			// create a CompPattern
			probeName := "probe-"+netIdxStr+"-"+endptIdxStr
			probeCmpPtn := pces.CreateCompPattern(probeCmpPtnType)
			probeCmpPtn.SetName(probeName)
			probeCPInit := pces.CreateCPInitList(probeName, probeCmpPtnType, true) 

			// add a 'probe' type Func to it, called 'probe'
			pces.FuncClassNames["probe"] = true
			probeFunc := pces.CreateFunc("probe", "probe")
			probeCmpPtn.AddFunc(probeFunc)

			// two kinds of messages, 'probepckt' and 'probeflow'
			cpm := pces.CreateCompPatternMsg("probepckt", true)
			probeCPInit.AddMsg(cpm)
			cpm = pces.CreateCompPatternMsg("probeflow", false)
			probeCPInit.AddMsg(cpm)

			// create a cfg record, which turns out to be trivial
			probeFuncCfg := probe.CreateProbeCfg(probeName)
			serialProbeCfg, err := probeFuncCfg.Serialize(true)
			if err != nil {
				panic(err)
			}
			probeCPInit.AddCfg(probeCmpPtn, probeFunc, serialProbeCfg)	

			cmpPtnDict.AddCompPattern(probeCmpPtn)
			cpInitListDict.AddCPInitList(probeCPInit)

			// map the functions of probeCmpPtn
			cmpMap := pces.CreateCompPatternMap(probeCmpPtn.Name)
			endptname := "endpt-"+netIdxStr+"-"+endptIdxStr
			cmpMap.AddMapping(probeFunc.Label, endptname, false)
			cmpMapDict.AddCompPatternMap(cmpMap, false)
		}
	}
	cmpPtnDict.WriteToFile(fullpathmap["cp"])
	cpInitListDict.WriteToFile(fullpathmap["cpInit"])
	cmpMapDict.WriteToFile(fullpathmap["map"])

	// should create some configuration parameters

	// create the dictionary to be populated
	expCfg := mrnes.CreateExpCfg("Evaluation")

	// load up descriptions in global data structure for checking
	mrnes.GetExpParamDesc()

	// experiment parameters are largely about architectural parameters
	// that impact performance. Define some defaults (which can be overwritten later)
	//

	// default parameters
	wcAttrbs := []mrnes.AttrbStruct{mrnes.AttrbStruct{AttrbName: "*", AttrbValue: ""}}
	expCfg.AddParameter("Interface", wcAttrbs, "delay", intrfcdelay)
	expCfg.AddParameter("Interface", wcAttrbs, "latency", "1e-6")

	// every network to have a latency of 1e-4
	expCfg.AddParameter("Network", wcAttrbs, "latency", netlatency)

	// every network to have a bandwidth of 1000 (Mbits)
	expCfg.AddParameter("Network", wcAttrbs, "capacity", netcapacity)
	netCapacity, _ := strconv.ParseFloat(netcapacity, 64)
	fracCapacity := netCapacity/4.0
	fracCapacityStr := strconv.FormatFloat(fracCapacity, 'g', -1, 64)
	expCfg.AddParameter("Network", wcAttrbs, "bandwidth", fracCapacityStr)

	expCfg.AddParameter("Interface", wcAttrbs, "bandwidth", "10")

	// every interface to have an MTU of 1500 bytes
	expCfg.AddParameter("Interface", wcAttrbs, "MTU", "1500")

	// trace off, every device, but on every interface
	expCfg.AddParameter("Endpt", wcAttrbs, "trace", "false")
	expCfg.AddParameter("Switch", wcAttrbs, "trace", "false")
	expCfg.AddParameter("Router", wcAttrbs, "trace", "false")
	expCfg.AddParameter("Interface", wcAttrbs, "trace", "true")
	expCfg.AddParameter("Interface", wcAttrbs, "buffer", intrfcbfr)
	expCfg.AddParameter("Interface", wcAttrbs, "delay", intrfcdelay)

	endptAttrbs := []mrnes.AttrbStruct{ mrnes.AttrbStruct{AttrbName:"devtype", AttrbValue: "Endpt"}}
	expCfg.AddParameter("Interface", endptAttrbs, "bandwidth", endptbw)

	rtrAttrbs := []mrnes.AttrbStruct{mrnes.AttrbStruct{AttrbName:"devtype", AttrbValue: "Router"}}
	expCfg.AddParameter("Interface", rtrAttrbs, "bandwidth", rtrbw)

	switchAttrbs := []mrnes.AttrbStruct{mrnes.AttrbStruct{AttrbName:"devtype", AttrbValue: "Switch"}}
	expCfg.AddParameter("Interface", switchAttrbs, "bandwidth", switchbw)

	expCfg.WriteToFile(fullpathmap["exp"])

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

	// we don't have shared cfg in this model but need to create an empty file
	scgl := pces.CreateSharedCfgGroupList(true) 
	scgl.WriteToFile(fullpathmap["srdCfg"])

}

