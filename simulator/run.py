
# this script requires python 3.10 or later to be called when the command 'python3' is
# passed through the subprocess command (for the 'match' statement).
# It needs the pandas and openpyxl packages to be installed
#
import argparse
import subprocess
import shutil
import datetime
import yaml
import json
import sys
import pdb
import os
import glob

sheetNames = ('cp', 'topo', 'execTime', 'netParams', 'mapping')

def main():
    global workingDir, full_csvDir

    parser = argparse.ArgumentParser()
    parser.add_argument(u'-template', metavar = u'directory with experiment templates', dest=u'template', required=True)
    parser.add_argument(u'-input', metavar = u'directory where transformed files reside', dest=u'input', required=True)
    parser.add_argument(u'-output', metavar = u'directory where simulation output is written', 
        dest=u'output', required=True)
    parser.add_argument(u'-args', metavar = u'directory where argument files are stored', 
        dest=u'argsDir', required=True)
    parser.add_argument(u'-sim', metavar = u'directory where simulation executatable and argument file reside', 
        dest=u'sim', required=True)

    cmdline = sys.argv[1:]
    cmdline = []
    with open(sys.argv[2],"r") as rf:
        for line in rf:
            line = line.strip()
            if len(line) == 0 or line.startswith('#'):
                continue
            cmdline.extend(line.split()) 

    args = parser.parse_args(cmdline)

    templateDir = args.template
    inputDir = args.input
    outputDir = args.output
    simDir = args.sim
    argsDir = args.argsDir

    full_templateDir = os.path.abspath(templateDir)
    full_inputDir = os.path.abspath(inputDir)
    full_outputDir = os.path.abspath(outputDir)
    full_simDir = os.path.abspath(simDir)
    full_argsDir = os.path.abspath(argsDir)

    commonDir = (full_templateDir, full_inputDir, full_outputDir, full_simDir, full_argsDir)

    # ensure that these directories exist and are accessible
    errs = 0
    for cd in commonDir:
        if not os.path.isdir(cd):
            print('argument directory', cd, 'not accessible')
            errs += 1

    if errs>0:
        exit(1)

    # make sure we can get to all the files we expect in template
    sheet2Files = {}
    sheet2Files['cp'] = ['cp.yaml', 'cpInit.yaml']
    sheet2Files['topo'] = ['topo.yaml']
    sheet2Files['execTime'] = ['funcExec.yaml', 'devExec.yaml']
    sheet2Files['mapping'] = ['map.yaml']
    sheet2Files['netParams'] = ['exp.yaml']
    sheet2Files['experiments'] = ['experiments.yaml']

    for _, fList in sheet2Files.items():
        for fileName in fList:
            filePath = os.path.join(full_templateDir, fileName)
            if not os.path.isfile(filePath):
                print('expected file {} does not exist'.format(filePath))
                errs += 1
            
    if errs > 0:
        exit(1)

    # get the dictionary describing the experiments
    experiment_input_file = os.path.join(full_templateDir, 'experiments.yaml')
    with open(experiment_input_file, 'r') as rf:
        exprmnts = yaml.safe_load(rf)

    # copy over everything
    directory_path = full_templateDir
    file_pattern = '*.yaml'
    filenames = glob.glob(f'{directory_path}/{file_pattern}')

    for filePath in filenames:
        basename = os.path.basename(filePath) 
        input_file = os.path.join(full_inputDir, basename)
        shutil.copyfile(filePath, input_file)            

    aggOutFile = os.path.join(full_outputDir, 'results.yaml')
    with open(aggOutFile, 'w') as wf:
        print('experiment set run at time {}'.format(datetime.datetime.now()), file=wf) 

    allMsr = []
        
    for exprmnt in exprmnts:
        exprmntName = exprmnt['name']

        with open(os.path.join(full_argsDir,'args-sim-template'), 'r') as tf, open(os.path.join(full_argsDir,'args-sim'), 'w') as wf:
            wf.write('-exprmnt {}\n'.format(exprmntName))
            for line in tf:
                wf.write(line)

        sheetFlag = {}
        # get the files to be modified
        for code in exprmnt:
            if code == 'name':
                continue
            pieces = code.split(',')
            if len(pieces) > 1:
                sheets = pieces[1:]
            else:
                sheets = sheetNames

            for sheet in sheetNames:
                sheetFlag[sheet] = True

        # copy the files to be modified
        for sheet in sheetFlag:
            for file in sheet2Files[sheet]:
                templateFile = os.path.join(full_templateDir, file)
                inputFile = os.path.join(full_inputDir, file)
                shutil.copyfile(templateFile, inputFile)
 
        # make the modifications
        for code, value in exprmnt.items():
            if code == 'name':
                continue
            pieces = code.split(',')
            token = pieces[0]
            if len(pieces) > 1:
                sheets = pieces[1:]
            else:
                sheets = sheetNames

            for sheet in sheets:
                for file in sheet2Files[sheet]:
                    inputFile = os.path.join(full_inputDir, file)
                    tmpFile = os.path.join(full_inputDir, 'tmp-'+file)
                    with open(inputFile, 'r') as rf:
                        with open(tmpFile, 'w') as wf:
                            for line in rf:
                                newline = line.replace(token, value)
                                wf.write(newline)
                    shutil.copyfile(tmpFile, inputFile)
                    os.remove(tmpFile)

        # run the simulation
   
        simExec = os.path.join(full_simDir,"sim")
        simArgs = os.path.join(full_argsDir, "args-sim")

        process = subprocess.Popen([simExec, "-is", simArgs], 
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        stdout, stderr = process.communicate()
       
        if process.returncode != 0:
            print("Error from simulation run")
        else:
            if len(stderr) > 0 :
                print(stderr)
    
            # append the msr file in output to output/aggMsr
            outputMsr = os.path.join(full_outputDir, 'msr.yaml')

            with open(outputMsr, 'r') as fsrc:
                nxtMsr = yaml.safe_load(fsrc)
                allMsr.append(nxtMsr)

    with open(aggOutFile, 'a') as fdst:
        yaml.dump(allMsr, fdst) 


if __name__ == "__main__":
    main()


