package main

import (
	"os"
	"encoding/csv"
	"github.com/iti/cmdline"
	"github.com/iti/mrnesbits"
	"github.com/iti/mrnes"
    "golang.org/x/exp/slices"
	"path/filepath"
	"strconv"
	"strings"
	"sort"
)

var isCryptoOp map[string]bool = map[string]bool{"encrypt":true, "decrypt":true, "hash":true, "sign":true} 

// cmdlineParams defines the parameters recognized
// on the command line
func cmdlineParams() *cmdline.CmdParser {
	// command line parameters are all about file and directory locations.
	cp := cmdline.NewCmdParser()
	cp.AddFlag(cmdline.StringFlag, "db", true)		  // directory where 'database' of csv models reside
	cp.AddFlag(cmdline.StringFlag, "outputDir", true) // directory where 'database' of csv models reside
	return cp
}

// main gives the entry point
func main() {
	// define the command line parameters
	cp := cmdlineParams()

	// parse the command line
	cp.Parse()

	// string for directory with source .csv files
	dbDir := cp.GetVar("db").(string)

	// where the cryptoDesc goes
	outputDir := cp.GetVar("outputDir").(string)

	timingDir := filepath.Join(dbDir,"timing")
	funcXDir := filepath.Join(timingDir,"funcExec")
	descDir  := filepath.Join(dbDir,"desc")
	devDescDir := filepath.Join(descDir,"devDesc")

	// make sure these directories exist
	dirs := []string{devDescDir, funcXDir, outputDir}
	valid, err := mrnesbits.CheckDirectories(dirs)
	if !valid {
		panic(err)
	}

	// gather up the names of all .csv files in the funcXDir directory
	pattern := "*.yaml"
	if len(funcXDir) > 0 {
		pattern = filepath.Join(funcXDir,"*.yaml")
	}

	funcXFiles, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}

	// index by algorithm code, list associated keys
	algKeys := make(map[string][]int)

	for _, fxFile := range funcXFiles {
		var emptybytes []byte
		fx, err := mrnesbits.ReadFuncExecList(fxFile, true, emptybytes)
		if err != nil {
			panic(err)
		}
		for op := range fx.Times {
			pieces := strings.Split(op,"-")
			if len(pieces) != 3 {
				continue
			}

			for idx, _ := range pieces {
				pieces[idx] = strings.TrimSpace(pieces[idx])
			}
			if !isCryptoOp[pieces[0]] {
				continue
			}
			algCode := pieces[1]
			keyLen, err := strconv.Atoi(pieces[2])
			if err != nil {
				continue
			}

			_, present := algKeys[algCode]
			if !present {
				algKeys[algCode] = []int{}
			}
			if !slices.Contains(algKeys[algCode], keyLen) {
				algKeys[algCode] = append(algKeys[algCode],keyLen)
			}
		}
	}
	
	// create and write out the cryptoDesc
	cryptoDesc := mrnesbits.CryptoDesc{Algs: []mrnesbits.AlgSpec{}}
	for alg := range algKeys {
		algDesc := mrnesbits.AlgSpec{Name:alg, KeyLen: algKeys[alg]}
		sort.Ints(algDesc.KeyLen)
		cryptoDesc.Algs = append(cryptoDesc.Algs, algDesc)
	}	

	cryptoDescFile := filepath.Join(outputDir, "cryptoDesc")+".yaml"
	cryptoDesc.WriteToFile(cryptoDescFile)


	// create and write out the device description 
	// find the files with device descriptions
	pattern = "*.csv"
	if len(devDescDir) > 0 {
		pattern = filepath.Join(devDescDir,"*.csv")
	}

	descFiles, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}

	// make a map of all the devDesc files
	devdd := mrnes.CreateDevDescDict("beta")

	for _, descFile := range descFiles {
		cdf, err := os.Open(descFile)
		if err != nil {
			panic(err)
		}

		cdfReader := csv.NewReader(cdf)
		records, err := cdfReader.ReadAll()
		if err != nil {
			panic(err)
		}
		cdf.Close()

		for idx := 1; idx < len(records); idx++ {
			ddr := records[idx]
			devType := ddr[0]
			manf := ddr[1]
			model := ddr[2]
			cores, _ := strconv.Atoi(ddr[3])
			freq, _ := strconv.ParseFloat(ddr[4], 64)
			cache, _ := strconv.ParseFloat(ddr[5], 64)
			dd := mrnes.CreateDevDesc(devType, manf, model, cores, freq, cache)
			devdd.AddDevDesc(dd)
		}
	}

	// write the device description in both the output directory
	// and the directory where the equivalent .csv version was extracted
	devDescFile := filepath.Join(outputDir, "devDesc")+".yaml"
	devdd.WriteToFile(devDescFile)
	devDescFile = filepath.Join(devDescDir, "devDesc")+".yaml"
	devdd.WriteToFile(devDescFile)
}			

