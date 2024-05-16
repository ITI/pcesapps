package main

import (
	"fmt"
	"github.com/iti/cmdline"
	"github.com/iti/mrnes"
	"github.com/iti/mrnesbits"
	"path/filepath"
	"strconv"
)

// cmdlineParams defines the parameters recognized
// on the command line
func mdfycmdlineParams() *cmdline.CmdParser {
	// command line parameters are all about file and directory locations.
	// Even though we don't need the flags for the other MrNesbits structures we
	// keep them here so that all the programs that build templates can use the same arguments file
	// create an argument parser
	cp := cmdline.NewCmdParser()
	cp.AddFlag(cmdline.StringFlag, "srccpu", true)
	cp.AddFlag(cmdline.FloatFlag, "srcls", true)

	cp.AddFlag(cmdline.StringFlag, "swpvtm", true)
	cp.AddFlag(cmdline.FloatFlag, "swpvtls", true)
	cp.AddFlag(cmdline.StringFlag, "rtrpvtm", true)
	cp.AddFlag(cmdline.FloatFlag, "rtrpvtls", true)

	cp.AddFlag(cmdline.StringFlag, "swpubm", true)
	cp.AddFlag(cmdline.FloatFlag, "swpubls", true)
	cp.AddFlag(cmdline.StringFlag, "rtrpubm", true)
	cp.AddFlag(cmdline.FloatFlag, "rtrpubls", true)

	cp.AddFlag(cmdline.StringFlag, "sslcpu", true)
	cp.AddFlag(cmdline.FloatFlag, "sslls", true)
	cp.AddFlag(cmdline.StringFlag, "eudcpu", true)
	cp.AddFlag(cmdline.FloatFlag, "eudls", true)
	cp.AddFlag(cmdline.StringFlag, "cryptoalg", false)

	addArgChk(cp, "srccpu", []string{"x86", "i3", "i7"})
	addArgChk(cp, "sslcpu", []string{"x86", "i3", "i7"})
	addArgChk(cp, "eudcpu", []string{"x86", "i3", "i7"})

	addArgChk(cp, "swpvtm", []string{"FS_N8560-32C", "FS_S3900-48T6S-R"})
	addArgChk(cp, "swpubm", []string{"FS_N8560-32C", "FS_S3900-48T6S-R"})
	addArgChk(cp, "rtrpvtm", []string{"Juniper_MX_240", "Cisco_Catalyst_8200"})
	addArgChk(cp, "rtrpubm", []string{"Juniper_MX_240", "Cisco_Catalyst_8200"})

	addArgChk(cp, "srcls", []string{"float"})
	addArgChk(cp, "swpvtls", []string{"float"})
	addArgChk(cp, "swpubls", []string{"float"})
	addArgChk(cp, "rtrpubls", []string{"float"})
	addArgChk(cp, "rtrpubls", []string{"float"})
	addArgChk(cp, "sslls", []string{"float"})
	addArgChk(cp, "eudls", []string{"float"})
	addArgChk(cp, "cryptoalg", []string{"aes", "rsa-3072", "none"})

	return cp
}

type argChk struct {
	name   string
	values []string
}

var argChks = []argChk{}

func addArgChk(cp *cmdline.CmdParser, name string, values []string) {
	argChks = append(argChks, argChk{name: name, values: values})
}

var useYAML bool

// main gives the entry point
func main() {
	// define the command line parameters
	cp := mdfycmdlineParams()

	// parse the command line
	cp.Parse()

	// check the input values
	validateArgs(cp)

	// string for the output directory
	outputLib := "../input"

	// make sure this directory exists
	dirs := []string{outputLib}
	valid, err := mrnesbits.CheckDirectories(dirs)
	if !valid {
		panic(err)
	}

	// check for access to output files
	basefile := "mdfy.yaml"
	finalfile := filepath.Join(outputLib, basefile)
	fullpath := []string{finalfile}

	valid, err = mrnesbits.CheckOutputFiles(fullpath)
	if !valid {
		panic(err)
	}

	useYAML = true
	expCfg := mrnes.CreateExpCfg("update")

	// parameters for src
	nmAs := mrnes.AttrbStruct{AttrbName: "name", AttrbValue: "pcktsrc"}
	lsAs := mrnes.AttrbStruct{AttrbName: "device", AttrbValue: "pcktsrc"}
	nmAsv := []mrnes.AttrbStruct{nmAs}
	lsAsv := []mrnes.AttrbStruct{lsAs}
	srccpu := cp.GetVar("srccpu").(string)
	srcLS := cp.GetVar("srcls").(float64)
	expCfg.AddParameter("Endpt", nmAsv, "CPU", srccpu)
	expCfg.AddParameter("Interface", lsAsv, "bandwidth", strconv.FormatFloat(srcLS, 'E', -1, 64))

	// parameters for eud
	nmAs = mrnes.AttrbStruct{AttrbName: "name", AttrbValue: "eudDev"}
	lsAs = mrnes.AttrbStruct{AttrbName: "device", AttrbValue: "eudDev"}
	nmAsv = []mrnes.AttrbStruct{nmAs}
	lsAsv = []mrnes.AttrbStruct{lsAs}
	eudcpu := cp.GetVar("eudcpu").(string)
	eudLS := cp.GetVar("eudls").(float64)
	expCfg.AddParameter("Endpt", nmAsv, "CPU", eudcpu)
	expCfg.AddParameter("Interface", lsAsv, "bandwidth", strconv.FormatFloat(eudLS, 'E', -1, 64))

	// parameters for ssl
	nmAs = mrnes.AttrbStruct{AttrbName: "name", AttrbValue: "ssl"}
	lsAs = mrnes.AttrbStruct{AttrbName: "device", AttrbValue: "ssl"}
	nmAsv = []mrnes.AttrbStruct{nmAs}
	lsAsv = []mrnes.AttrbStruct{lsAs}
	sslcpu := cp.GetVar("sslcpu").(string)
	sslLS := cp.GetVar("sslls").(float64)
	expCfg.AddParameter("Filter", nmAsv, "CPU", sslcpu)
	expCfg.AddParameter("Interface", lsAsv, "bandwidth", strconv.FormatFloat(sslLS, 'E', -1, 64))

	// parameters for pvt switch
	nmAs = mrnes.AttrbStruct{AttrbName: "name", AttrbValue: "pvtSwitch"}
	lsAs = mrnes.AttrbStruct{AttrbName: "device", AttrbValue: "pvtSwitch"}
	nmAsv = []mrnes.AttrbStruct{nmAs}
	lsAsv = []mrnes.AttrbStruct{lsAs}
	swpvtm := cp.GetVar("swpvtm").(string)
	switchLS := cp.GetVar("swpvtls").(float64)
	expCfg.AddParameter("Switch", nmAsv, "model", swpvtm)
	expCfg.AddParameter("Interface", lsAsv, "bandwidth", strconv.FormatFloat(switchLS, 'E', -1, 64))

	// parameters for pub switch
	switchLS = cp.GetVar("swpubls").(float64)
	swpubm := cp.GetVar("swpubm").(string)
	nmAs = mrnes.AttrbStruct{AttrbName: "name", AttrbValue: "pubSwitch"}
	lsAs = mrnes.AttrbStruct{AttrbName: "device", AttrbValue: "pubSwitch"}
	nmAsv = []mrnes.AttrbStruct{nmAs}
	lsAsv = []mrnes.AttrbStruct{lsAs}
	expCfg.AddParameter("Switch", nmAsv, "model", swpubm)
	expCfg.AddParameter("Interface", lsAsv, "bandwidth", strconv.FormatFloat(switchLS, 'E', -1, 64))

	// parameters for pvt router
	nmAs = mrnes.AttrbStruct{AttrbName: "name", AttrbValue: "pvtRtr"}
	lsAs = mrnes.AttrbStruct{AttrbName: "device", AttrbValue: "pvtRtr"}
	nmAsv = []mrnes.AttrbStruct{nmAs}
	lsAsv = []mrnes.AttrbStruct{lsAs}
	rtrpvtm := cp.GetVar("rtrpvtm").(string)
	routerLS := cp.GetVar("rtrpvtls").(float64)
	expCfg.AddParameter("Router", nmAsv, "model", rtrpvtm)
	expCfg.AddParameter("Interface", lsAsv, "bandwidth", strconv.FormatFloat(routerLS, 'E', -1, 64))

	// parameters for pub router
	routerLS = cp.GetVar("rtrpubls").(float64)
	rtrpubm := cp.GetVar("rtrpubm").(string)
	nmAs = mrnes.AttrbStruct{AttrbName: "name", AttrbValue: "pubRtr"}
	lsAs = mrnes.AttrbStruct{AttrbName: "device", AttrbValue: "pubRtr"}
	nmAsv = []mrnes.AttrbStruct{nmAs}
	lsAsv = []mrnes.AttrbStruct{lsAs}
	expCfg.AddParameter("Router", nmAsv, "model", rtrpubm)
	expCfg.AddParameter("Interface", lsAsv, "bandwidth", strconv.FormatFloat(routerLS, 'E', -1, 64))

	expCfg.WriteToFile(finalfile)
}

func validateArgs(cp *cmdline.CmdParser) {
	for _, ac := range argChks {
		if len(ac.values) == 1 && ac.values[0] == "float" {
			value := cp.GetVar(ac.name).(float64)
			if value < 0.0 {
				panic(fmt.Errorf("negative floating point argument %f", value))
			}
		} else {
			value := cp.GetVar(ac.name).(string)
			found := false
			for _, allowed := range ac.values {
				if value == allowed {
					found = true
					break
				}
			}
			if !found {
				panic(fmt.Errorf("disallowed input -%s %s %v", ac.name, value, ac.values))
			}
		}
	}
}
