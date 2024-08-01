package main

import (
	"fmt"
	"os"
	"github.com/iti/cmdline"
	"github.com/iti/pces"
	"github.com/iti/mrnes"
	"bufio"
	"path/filepath"
	"strconv"
	"strings"
)

// cmdlineParams defines the parameters recognized
// on the command line
func cmdlineParams() *cmdline.CmdParser {
	// command line parameters are all about file and directory locations.
	cp := cmdline.NewCmdParser()
	cp.AddFlag(cmdline.StringFlag, "db", true)		   // directory where 'database' of csv models reside
	cp.AddFlag(cmdline.StringFlag, "outputDir", false) // directory where 'database' of csv models reside
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

	timingDir := filepath.Join(dbDir,"timing")
	funcXDir := filepath.Join(timingDir,"funcExec")
	devXDir  := filepath.Join(timingDir,"devExec")

	// make sure these directories exist
	dirs := []string{funcXDir, devXDir}
	valid, err := pces.CheckDirectories(dirs)
	if !valid {
		panic(err)
	}

	// gather up the names of all .csv files in the funcXDir directory
	pattern := "*.csv"
	if len(funcXDir) > 0 {
		pattern = filepath.Join(funcXDir,"*.csv")
	}

	funcXFiles, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}

	for _, fXFile := range funcXFiles {
		// get the base file name
		fullBase := filepath.Base(fXFile)	
		base := strings.TrimSuffix(fullBase, filepath.Ext(fullBase)) 
		// create a timing list, give this list the base name of the csv file
		fel := pces.CreateFuncExecList(base)
		
		csvRdr, err := os.Open(fXFile)	
		if err != nil {
			panic(err)
		}

		fileScanner := bufio.NewScanner(csvRdr)
		fileScanner.Split(bufio.ScanLines)

		started := false	
		opName := ""
		devName := ""
		opFirst := true
		
		for fileScanner.Scan() {
			line := fileScanner.Text()
			fields := strings.Split(line,",")

			if len(fields) != 5 {
				panic(fmt.Errorf("each line of timing table requires 5 columns"))
			}

			empty := true
			for idx, _ := range fields {
				fields[idx] = strings.TrimSpace(fields[idx])
				if len(fields[idx]) > 0 {
					empty = false
				}
			}

			// skip lines with nothing there
			if empty {
				continue
			}
				
			// skip the first line, but note we've started
			if !started && (fields[0] == "operation" || fields[0] == "Operation") {
				started = true
				continue
			}

			// check for reversed organization
			if !started && (fields[1] == "operation" || fields[1] == "Operation") {
				started = true
				opFirst = false
				continue
			}

			// if "operation" is not leading, exchange fields[0] and fields[1]
			if !opFirst {
				tmp := fields[0]		
				fields[0] = fields[1]
				fields[1] = tmp
			}

			// is this the first appearance of a new operation?		
			if len(fields[0]) > 0 && opName != fields[0] {
				opName = fields[0]

				// new ops require an update devName
				if len(fields[1]) == 0 {
					panic(fmt.Errorf("timing table lines with op codes require device identity"))
				}
				devName = fields[1]
				
				pieces := strings.Split(opName,"-")
				for idx,_ := range pieces {
					pieces[idx] = strings.ToLower(pieces[idx])
				}
			}

			// possibly change the device Name
			if len(fields[1]) > 0 && devName !=  fields[1] {
				devName = fields[1]
			}

			// ensure the integrity of the packetLen and time values
			ps, err := strconv.Atoi(fields[2]) 
			if err != nil || ps < 1 {
				panic(fmt.Errorf("packetSize entry in timing table must be positive integer"))
			}

			ex, err  :=	strconv.ParseFloat(fields[4], 64)
			if err != nil || ex < 0.0 {
				panic(fmt.Errorf("execution time in table must be nonegative floating point"))
			}

			// scale ex (microseconds) to seconds
			ex = ex/1e+6			
			// add timing to list
			fel.AddTiming(opName, fields[3], devName, ps, ex)
		} 
		// write the timing description out to file
		// replace the .csv extention of csvFile to .yaml or .json
		outputFile := strings.TrimSuffix(fXFile, filepath.Ext(fXFile))+".yaml" 
		fel.WriteToFile(outputFile)
	}

	// gather up the names of all .csv files in the devXDir directory
	pattern = "*.csv"
	if len(devXDir) > 0 {
		pattern = filepath.Join(devXDir,"*.csv")
	}

	devXFiles, err := filepath.Glob(pattern)
	if err != nil {
		panic(err)
	}

	for _, dXFile := range devXFiles {
		// get the base file name
		fullBase := filepath.Base(dXFile)	
		base := strings.TrimSuffix(fullBase, filepath.Ext(fullBase)) 

		// create a timing list, give this list the base name of the csv file
		del := mrnes.CreateDevExecList(base)
		
		csvRdr, err := os.Open(dXFile)	
		if err != nil {
			panic(err)
		}

		fileScanner := bufio.NewScanner(csvRdr)
		fileScanner.Split(bufio.ScanLines)

		started := false	
		opName := ""
		devName := ""
		opFirst := false
	
		for fileScanner.Scan() {
			line := fileScanner.Text()
			fields := strings.Split(line,",")

			if len(fields) != 3 {
				panic(fmt.Errorf("each line of device timing table requires 3 columns"))
			}

			empty := true
			for idx, _ := range fields {
				fields[idx] = strings.TrimSpace(fields[idx])
				if len(fields[idx]) > 0 {
					empty = false
				}
			}

			// skip lines with nothing there
			if empty {
				continue
			}
				
			// skip the first line, but note we've started
			if !started && (fields[0] == "operation") {
				started = true
				opFirst = true	
				continue
			}

			// check for reversed organization
			if !started && (fields[1] == "operation" || fields[1] == "Operation") {
				started = true
				opFirst = false
				continue
			}

			// if "operation" is not leading, exchange fields[0] and fields[1]
			if !opFirst {
				tmp := fields[0]		
				fields[0] = fields[1]
				fields[1] = tmp
			}

			// is this the first appearance of a new operation?		
			if len(fields[0]) > 0 && opName != fields[0] {
				opName = fields[0]

				// new ops require an update devName
				if len(fields[1]) == 0 {
					panic(fmt.Errorf("device timing table lines with op codes require device identity"))
				}
				devName = fields[1]
			}

			// possibly change the device Name
			if len(fields[1]) > 0 && devName !=  fields[1] {
				devName = fields[1]
			}

			// ensure the integrity of the time value
			ex, err  :=	strconv.ParseFloat(fields[2], 64)
			if err != nil || ex < 0.0 {
				panic(fmt.Errorf("execution time in table must be nonegative floating point"))
			}

			// scale ex (microseconds) to seconds
			ex = ex/1e+6			

			// add timing to list
			del.AddTiming(opName, devName, ex) 
		} 
		// write the timing description out to file
		// replace the .csv extention of csvFile to .yaml or .json
		outputFile := strings.TrimSuffix(dXFile, filepath.Ext(dXFile))+".yaml" 
		del.WriteToFile(outputFile)
	}
}			



