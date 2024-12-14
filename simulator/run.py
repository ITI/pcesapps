
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
    global workingDir, csvDir

    parser = argparse.ArgumentParser()
    parser.add_argument(u'-template', metavar = u'directory with experiment templates', dest=u'template', required=True)
    parser.add_argument(u'-input', metavar = u'directory where transformed files reside', dest=u'input', required=True)
    parser.add_argument(u'-output', metavar = u'directory where simulation output is written', 
        dest=u'output', required=True)
    parser.add_argument(u'-sim', metavar = u'directory where simulation executatable and argument file reside', 
        dest=u'sim', required=True)
    parser.add_argument(u'-extern', metavar = u'if non-empty, container directory where shared input and output directories reside', 
        dest=u'extern', required=False)
    parser.add_argument(u'-container', metavar = u'tag to be applied to a container that is run', 
        dest=u'container', required=False)

    cmdline = sys.argv[1:]
    cmdline = []
    with open(sys.argv[2],"r") as rf:
        for line in rf:
            line = line.strip()
            if len(line) == 0 or line.startswith('#'):
                continue
            cmdline.extend(line.split()) 

    args = parser.parse_args(cmdline)

    # remember that path names are relative to current working directory
    templateDir = args.template
    externDir = args.extern

    if externDir is None:
        inputDir = args.input
        outputDir = args.output
    else:
        inputDir = os.path.join(externDir,'input')
        outputDir = os.path.join(externDir,'output')

    simDir = args.sim
    argsDir = './args'

    containerTag = args.container

    commonDir = (templateDir, inputDir, outputDir, simDir, argsDir)

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
            filePath = os.path.join(templateDir, fileName)
            if not os.path.isfile(filePath):
                print('expected file {} does not exist'.format(filePath))
                errs += 1
            
    if errs > 0:
        exit(1)

    # get the dictionary describing the experiments
    experiment_input_file = os.path.join(templateDir, 'experiments.yaml')
    with open(experiment_input_file, 'r') as rf:
        exprmnts = yaml.safe_load(rf)

    # copy over everything
    directory_path = templateDir
    file_pattern = '*.yaml'
    filenames = glob.glob(f'{directory_path}/{file_pattern}')

    for filePath in filenames:
        basename = os.path.basename(filePath) 
        input_file = os.path.join(inputDir, basename)
        shutil.copyfile(filePath, input_file)            

    aggOutFile = os.path.join(outputDir, 'results.yaml')
    with open(aggOutFile, 'w') as wf:
        print('experiment set run at time {}'.format(datetime.datetime.now()), file=wf) 

    allMsr = []
        
    for exprmnt in exprmnts:
        exprmntName = exprmnt['name']

        # write the name of the experiment into the args-sim file used to run the simulation
        with open(os.path.join(argsDir,'args-sim-template'), 'r') as tf, open(os.path.join(argsDir,'args-sim'), 'w') as wf:
            wf.write('-exprmnt {}\n'.format(exprmntName))

            if containerTag is not None:
                wf.write('-container\n')
 
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
                templateFile = os.path.join(templateDir, file)
                inputFile = os.path.join(inputDir, file)
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
                    inputFile = os.path.join(inputDir, file)
                    tmpFile = os.path.join(inputDir, 'tmp-'+file)
                    with open(inputFile, 'r') as rf:
                        with open(tmpFile, 'w') as wf:
                            for line in rf:
                                newline = line.replace(token, value)
                                wf.write(newline)
                    shutil.copyfile(tmpFile, inputFile)
                    os.remove(tmpFile)

        # run the simulation
   
        simArgs = os.path.join(argsDir, "args-sim")

        # if containerTag is not None, run the container to execute the simulation
        if containerTag is not None:
            cwd = os.getcwd()
            paths = os.path.split(os.getcwd())
            outsideDir = os.path.join(paths[:len(paths)-1])
            mountCmd = '{}:{}'.format(outsideDir, externDir)
            #cTag = "ghcr.io/iti/pcesapps-dev" 
            cTag = "pces/sim"
            simExecArgs = ["docker","run","-it", "--rm" "-v", mountCmd,  cTag]

            process = subprocess.Popen(simExecArgs,
                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
          
        else: 
            simExec = os.path.join(simDir,"sim")
            process = subprocess.Popen([simExec, "-is", simArgs], 
                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
           
        stdout, stderr = process.communicate()

        if process.returncode != 0:
            print("Error from simulation run")
        else:
            if len(stderr) > 0 :
                print(stderr)
    
            # append the msr file in output to output/aggMsr
            outputMsr = os.path.join(outputDir, 'msr.yaml')

            with open(outputMsr, 'r') as fsrc:
                nxtMsr = yaml.safe_load(fsrc)
                allMsr.append(nxtMsr)
        
            os.remove(outputMsr)

    with open(aggOutFile, 'a') as fdst:
        yaml.dump(allMsr, fdst) 


if __name__ == "__main__":
    main()


