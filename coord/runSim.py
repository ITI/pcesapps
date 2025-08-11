
# this script requires python 3.10 or later to be called when the command 'python3' is
# passed through the subprocess command (for the 'match' statement).
# It needs the pandas and openpyxl packages to be installed
#
import argparse
import subprocess
import time
import shutil
import datetime
import yaml
import json
import sys
import pdb
import os
import glob

def boolRep(v):
    if isinstance(v,int):
        if v==0 or v==1:
            return v
        return 0

    if isinstance(v,str):
        if v in ('T', 'True', 'TRUE', 't', 'true', 'Y', 'Yes', 'YES', 'yes', '1'):
            return 1
        return 0



def main():
    global workingDir, csvDir

    parser = argparse.ArgumentParser()
    parser.add_argument(u'-templates', metavar = u'directory with experiment templates', dest=u'templates', required=True)
    parser.add_argument(u'-input', metavar = u'directory where transformed files reside', dest=u'input', required=True)
    parser.add_argument(u'-output', metavar = u'directory where simulation output is written', 
        dest=u'output', required=True)
    parser.add_argument(u'-sim', metavar = u'directory where simulation executatable and argument file reside', 
        dest=u'sim', required=False)
    parser.add_argument(u'-user', action='store_true', required=False)
    parser.add_argument(u'-extern', metavar = u'if non-empty, container directory where shared input and output directories reside', 
        dest=u'extern', required=False)
    parser.add_argument(u'-container', metavar = u'tag to be applied to a container that is run', 
        dest=u'container', required=False)
    parser.add_argument(u'-name', metavar = u'name of model, to be included in output', 
        dest=u'name', required=True)

    # get command line argument file
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
    templateDir = args.templates
    externDir = args.extern

    modelName = args.name

    if externDir is None:
        inputDir = args.input
        outputDir = args.output
    else:
        inputDir = os.path.join(externDir,'input')
        outputDir = os.path.join(externDir,'output')


    if args.sim is None and args.container is None:
        print("expect either -sim or -container on command line")
        exit(1)

    if args.sim is not None and args.container is not None:
        print("expect exactly one of -sim or -container on command line")
        exit(1)

    argsDir = './args'
    containerTag = args.container

    if args.sim is not None:
        simDir = args.sim
        mainDir = os.path.join(simDir,'main')
        commonDir = (templateDir, inputDir, outputDir, simDir, mainDir, argsDir)
    else:
        commonDir = (templateDir, inputDir, outputDir, argsDir)

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
    sheet2Files['exectime'] = ['funcExec.yaml', 'devExec.yaml']
    sheet2Files['mapping'] = ['mapping.yaml']
    sheet2Files['netParams'] = ['exp.yaml']
    sheet2Files['experiments'] = ['experiments.yaml']
    sheet2Files['ipmap'] = ['ipmap.yaml']

    sheetNames = sorted(list(sheet2Files.keys()))

    for sheet, fList in sheet2Files.items():
        # skip sheets not required
        if sheet in ('ipmap'):
            continue
        for fileName in fList:
            # skip files that do not have to be present
            filePath = os.path.join(templateDir, fileName)
            if not os.path.isfile(filePath):
                print('expected file {} does not exist in templates directory'.format(filePath))
                errs += 1
            
    if errs > 0:
        exit(1)

    # get the dictionary describing the experiments
    experiment_input_file = os.path.join(templateDir, 'experiments.yaml')
    with open(experiment_input_file, 'r') as rf:
        exprmnts = yaml.safe_load(rf)

    # copy over everything into input directory
    directory_path = templateDir
    file_pattern = '*.yaml'
    filenames = glob.glob(f'{directory_path}/{file_pattern}')

    for filePath in filenames:
        basename = os.path.basename(filePath) 
        input_file = os.path.join(inputDir, basename)
        shutil.copyfile(filePath, input_file)            

    csvOutFile = os.path.join(outputDir, 'results.csv')
    if os.path.isfile(csvOutFile):
        os.remove(csvOutFile)

    # run each experiment individually   
    for exprmnt in exprmnts:
        exprmntName = exprmnt['name']

        # write the name of the experiment into the args-sim file used to run the simulation
        with open(os.path.join(argsDir,'args-sim-template'), 'r') as tf, open(os.path.join(argsDir,'args-sim'), 'w') as wf:
            wf.write('-exprmnt {}\n'.format(exprmntName))

            tagPairs = [('-inputLib', inputDir), ('-outputLib', outputDir), ('-cp', 'cp.yaml'), ('-cpInit', 'cpInit.yaml'),
                    ('-funcExec', 'funcExec.yaml'), ('-devExec', 'devExec.yaml'), ('-exp', 'exp.yaml'),('-mapping', 'mapping.yaml'),
                        ('-topo', 'topo.yaml'), ('-csv', 'results.csv'),  ('-experiments', 'experiments.yaml'), ('-ipmap', 'ipmap.yaml')]
           
 
            if containerTag is not None:
                tagPairs.append(('-container', ''))

            for tag, value in tagPairs:
                wf.write(tag+' '+value+'\n')
 
            for line in tf:
                # hard wire the input file names
                if line.startswith('#'):
                    continue
                pieces = line.split()
                if pieces[0] == '-container':
                    continue
                if pieces[0] == '-cp':
                    continue
                if pieces[0] == '-cpInit':
                    continue
                elif pieces[0] == '-funcExec':
                    continue
                elif pieces[0] == '-devExec':
                    continue
                elif pieces[0] == '-mapping':
                    continue 
                elif pieces[0] == '-exp':
                    continue
                elif pieces[0] == '-topo':
                    continue
                elif pieces[0] == 'inputLib':
                    continue
                elif pieces[0] == 'outputLib':
                    continue
                elif pieces[0] == 'ipmap':
                    continue
                else:      
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

            for sheet in sheets:
                sheetFlag[sheet] = True

        # copy the files to be modified into the input directory
        # (over-writing their state from the previous experiment)
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

            # the presence of a comma implies a comma-separated list of sheets to modify
            if len(pieces) > 1:
                sheets = pieces[1:]
            else:
                # no comma, modify all sheets, lack of symbols means no change
                sheets = sheetNames

            for sheet in sheets:
                for file in sheet2Files[sheet]:
                    inputFile = os.path.join(inputDir, file)
                    tmpFile = os.path.join(inputDir, 'tmp-'+file)
                    qt_strToken = '"str('+token
                    strToken = 'str('+token
                    qt_intToken = '"int('+token
                    intToken = 'int('+token
                    qt_floatToken = '"float('+token
                    floatToken = 'float('+token
                    qt_boolToken = '"bool('+token
                    boolToken = 'bool('+token

                    with open(inputFile, 'r') as rf:
                        with open(tmpFile, 'w') as wf:
                            for line in rf:
                                while line.find(token)> -1:
                                    if line.find(qt_strToken) > -1:
                                        start = line.find(strToken)
                                        front = line[:start]
                                        tail  = line[start+1:]
                                        end   = tail.find(')"')
                                        tail = tail[end+1:]
                                        line = front+str(value)+tail 
                                            
                                    if line.find(qt_intToken) > -1:
                                        start = line.find(qt_intToken)
                                        front = line[:start-1]
                                        tail = line[start+1:]
                                        end = tail.find(')"')
                                        tail = tail[end+2:]
                                        line = front+str(int(value))+tail
                                    
                                    if line.find(qt_floatToken) > -1:
                                        start = line.find(qt_floatToken)
                                        front = line[:start-1]
                                        tail = line[start+1:]
                                        end = tail.find(')"')
                                        tail = tail[end+2:]
                                        line = front+str(float(value))+tail
                                    
                                    if line.find(qt_boolToken) > -1:
                                        start = line.find(qt_boolToken)
                                        front = line[:start-1]
                                        tail = line[start+1:]
                                        end = tail.find(')"')
                                        tail = tail[end+2:]
                                        line = front+str(boolRep(value))+tail
                                    
                                    if line.find(strToken) > -1:
                                            start = line.find(strToken)
                                            front = line[:start]
                                            tail  = line[start+1:]
                                            end   = tail.find(')')
                                            tail = tail[end+1:]
                                            line = front+str(value)+tail 
                                            
                                    if line.find(intToken) > -1:
                                        start = line.find(intToken)
                                        front = line[:start]
                                        tail = line[start+1:]
                                        end = tail.find(')')
                                        tail = tail[end+1:]
                                        line = front+str(int(value))+tail
                                    
                                    if line.find(floatToken) > -1:
                                        start = line.find(floatToken)
                                        front = line[:start]
                                        tail = line[start+1:]
                                        end = tail.find(')')
                                        tail = tail[end+1:]
                                        line = front+str(float(value))+tail
                                    
                                    if line.find(boolToken) > -1:
                                        start = line.find(boolToken)
                                        front = line[:start]
                                        tail = line[start+1:]
                                        end = tail.find(')')
                                        tail = tail[end+1:]
                                        line = front+str(boolRep(value))+tail
                                    
                                wf.write(line)

                    shutil.copyfile(tmpFile, inputFile)
                    os.remove(tmpFile)

        # run the simulation
        simArgs = os.path.join(argsDir, "args-sim")
       
        time.sleep(1.0)
        print('running experiment {}'.format(exprmntName))


        # if containerTag is not None, run the container to execute the simulation
        if containerTag is not None:
            cwd = os.getcwd()
            paths = os.path.split(os.getcwd())
            outsideDir = paths[0]

            for directory in paths[1:len(paths)]:
                outsideDir = os.path.join(outsideDir, directory)

            mountCmd = '{}:{}'.format(outsideDir, "/tmp/extern")
           
            cmdList = ["docker", "run", "-it", "--rm", "-v", mountCmd,  containerTag]

            process = subprocess.Popen(cmdList,
                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
          
        else: 
            simExec = os.path.join(mainDir,"sim")
            if args.user:
                userArgs = os.path.join(argsDir,"args-user")
                cmdList = [simExec, "-is", userArgs, "-is", simArgs]
            else:
                cmdList = [simExec, "-is", simArgs]

            process = subprocess.Popen(cmdList,
                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
           
        stdout, stderr = process.communicate()

        if process.returncode != 0:
            print("Error from simulation run")
            if len(stdout) > 0:
                print(stdout)

            if len(stderr) > 0 :
                print(stderr)

            exit(1)

        if len(stdout) > 0:
            print(stdout)

        if len(stderr) > 0 :
            print(stderr)

if __name__ == "__main__":
    main()


