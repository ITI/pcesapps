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

# build.py lives in folder for particular pces app
# Every pces app folder has the same structure:
#  input  - directory where input files for a given run are placed
#  output - directory where output files for a given run are placed
#  simExp.py   - script to manage multiple runs whose outputs are collected as one experiment
#  buildExp.py - script to build infrastructure to conduct one experiment
#  args - subdirectory with argument files for buildExp.py, runSim.py, and the simulator itself:
#       args/args-build
#       args/args-sim-template
#       args/args-user
#
#  templates - subdirectory where input file templates stay for an experiment
#  xlsx - subdirectory with xlsxPCES input file and infrastructure for running xlsxPCES on it
#    structure of xlsx 
#       input - directory in which .xlsx input file is stored
#       args - directory with arguments for each script, created when buildExp.py is run
#       subdirectories -  working, descDir, csvDir, templateDir, created when buildExp.run is run  
#    script runConvert.py
#  
#    
# buildxlsxDir creates the subdirectories of xlsx and argument templates for the
# xlsxPCES conversion scripts
#  
def buildxlsxArgs(name, xpenv, xlsxDir):

    # ensure that the xlsx subdirectory has  
    #  subdirectories: args, working, descDir, csvDir, templateDir
    names = ('args', 'working', 'descDir', 'csvDir', 'templateDir')
    for directory in names:
        dirName = os.path.join(xlsxDir, directory)
        if not os.path.isdir(dirName):
            # make one
            os.mkdir(dirName)

    # create base argument files for convert scripts
    argsDir = os.path.join(xlsxDir, 'args')
    sheets = ('cp', 'exec', 'experiments', 'map', 'netparams', 'topo')

    for sheet in sheets:
        with open(os.path.join(argsDir, 'args-{}'.format(sheet)),'w') as wf:
            if sheet=='cp':
                print('-name {}'.format(name), file=wf)
                print('-csvIn cp-sheet.csv', file=wf)
                print('-cpuOpsDescIn cpuOps.json', file=wf)
                print('-tcDescOut tc.json', file=wf)
                print('-mc mc.json', file=wf)
                print('-funcsDescOut funcs.json', file=wf)
                print('-cmpptn cp.yaml', file=wf)
                print('-cpInit cpInit.yaml', file=wf)
            elif sheet=='exec':
                print('-name {}'.format(name), file=wf)
                print('-csvIn execTime-sheet.csv', file=wf)
                print('-cpuOpsDescOut cpuOps.json', file=wf)
                print('-modelDescOut devModel.json', file=wf)
                print('-funcExecOut funcExec.yaml', file=wf)
                print('-devExecOut devExec.yaml', file=wf)    
            elif sheet=='experiments':
                print('-name {}'.format(name), file=wf)
                print('-csvIn experiments-sheet.csv', file=wf)
                print('-experiments experiments.yaml', file=wf)
            elif sheet=='map':
                print('-name {}'.format(name), file=wf)
                print('-csvIn mapping-sheet.csv', file=wf)
                print('-funcsDesc funcs.json', file=wf)
                print('-cpuDesc cpuDesc.json', file=wf)
                print('-cpuOpsDesc cpuOps.json', file=wf)
                print('-tcDesc tc.json', file=wf)
                print('-map map.yaml', file=wf)
            elif sheet=='netparams':
                print('-name {}'.format(name), file=wf)
                print('-name embed', file=wf)
                print('-csvIn netParams-sheet.csv', file=wf)
                print('-exp exp.yaml', file=wf)
                print('-attrbDescIn attrb.json', file=wf)
            elif sheet=='topo':
                print('-name {}'.format(name), file=wf)
                print('-name embed', file=wf)
                print('-csvIn topo-sheet.csv', file=wf)
                print('-topoOut topo.yaml', file=wf)
                print('-modelDescIn devModel.json', file=wf)
                print('-cpuDescOut cpuDesc.json', file=wf)
                print('-attrbDescOut attrb.json', file=wf)


def main():

    # what we need is environment variable for xlsxPCES directory, optional, absent means skip xlsxPCES build
    #   - assumed layout for various local directories used in construction,
    #   - templates subdirectory here for output
    #
    # path to xlsx file
    # -container flag
    #
    parser = argparse.ArgumentParser()
    parser.add_argument(u'-xpenv', metavar = u'name of environment holding xlsxPCES tool', dest=u'xpenv', required=False)
    parser.add_argument(u'-xlsx', metavar = u'name of xlsxPCES input file in input directory', dest=u'xlsx', required=False)
    parser.add_argument(u'-name', metavar = u'label for this set of runs', dest=u'name', required=True)
    parser.add_argument(u'-simexec', action='store_true', required=False)
    parser.add_argument(u'-user', action='store_true', required=False)
    parser.add_argument(u'-container', metavar = u'tag to be applied to a container that is run', 
        dest=u'container', required=False)

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

    # if xpenv is used ensure that it is set up properly
    xpenv = None
    if args.xpenv is not None:
        if args.xpenv not in os.environ:
            print('error: expected environment variable {} to be set in shell'.format(args.xpenv))
            exit(1)
        xpenv = os.environ[args.xpenv]
        if not os.path.isdir(xpenv):
            print('error: expected directory for xlsxPCES tool at {}'.format(xpenv))
            exit(1)
        

    xlsx_input_file = args.xlsx

    # make sure that the directory in which this script (buildExp.py) lives has
    # runSim.py
    # subdirectories input, output, templates
    #
    scriptDir = os.path.dirname(__file__)
    subDirs = ('input', 'output', 'templates', 'args', 'sim-dir')
    for sub in subDirs:
        neededDir = os.path.join(scriptDir, sub)
        if not os.path.isdir(neededDir):
            os.mkdir(neededDir)

    # if container is selected make sure that Dockerfile-template is present
    containerTag = args.container
    dockerFile = os.path.join(scriptDir, 'Dockerfile-template')

    simDir = os.path.join(scriptDir,'sim-dir')

    if containerTag is not None and not os.path.isfile(dockerFile):
        print('container flag implies presence of Dockerfile')
        exit(1)

    # make sure there is a Dockerfile-template file
    if containerTag is not None:
        if not os.path.isfile( os.path.join(scriptDir, 'Dockerfile-template')):
            print('error: expected {}'.format(os.path.join(scriptDir, 'Dockerfile-template')))
            exit(1)

    argsDir = os.path.join(scriptDir,'args')
    if args.user: 
        userArgsFile = os.path.join(argsDir,'args-user')
        if not os.path.isfile(userArgsFile):
            print('error: expected user arguments file {}'.format(userArgsFile))
            exit(1)
 
    name = args.name

    # build the simulator or container, if needed
    if containerTag is not None:
        dockerfile = os.path.join(simDir,'Dockerfile')

        # gather the subdirectories within sim-dir
        os.chdir("sim-dir")
        subdirs = []
        for entry in os.scandir(os.getcwd()):
            if entry.is_dir():
                name = os.path.basename(entry.path)
                if name == 'main':
                    continue
                subdirs.append(name)

        usermodsList = [] 
        for name in subdirs:
            usermodsList.append('cd ./{} && go mod tidy && cd ../'.format(name))
        usermodules = '&&'.join(usermodsList)

        usercmd = '-is args-user' if args.user else ''
         
        with open(dockerfile,'w') as wf:
            with open(os.path.join(scriptDir,'Dockerfile-template'), 'r') as rf:
                for line in rf:
                    line = line.replace('$appname', name) 
                    line = line.replace('$usermodules', usermodules)
                    line = line.replace('$userargs', usercmd)
                    line = line.replace('$tag', containerTag)
                    wf.write(line)

        cmdList = ["docker", "image", "ls", containerTag]
        process = subprocess.Popen(cmdList,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        stdout, stderr = process.communicate()

        if len(stderr) > 0:
            print(stderr)
            exit(1)

        tagExists = False
        if len(stdout) > 0 :
            stdoutLines = stdout.split()
            for line in stdoutLines:
                if line.find(containerTag) > -1:
                    tagExists = True
                    print('warning: container tag {} exists already, remove before rebuild'.format(containerTag))
                    break

        if not tagExists: 
            print('Build image for tag {}'.format(containerTag))

            cmdList = ["docker", "build", "-t", containerTag, "."]
            process = subprocess.Popen(cmdList,
                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
            stdout, stderr = process.communicate()

            if process.returncode != 0:
                print('Build image for tag {} failed'.format(containerTag))
                exit(1)

            print(stdout)
            print('Build image for tag {} succeeded'.format(containerTag))

        os.chdir("../") 

    # build the native simulation executive if requested
    if args.simexec: 
        os.chdir('./sim-dir/main')

        gofiles = glob.glob("*.go")
        compileList = ["go","build","-o", "./sim"]
        compileList.extend(gofiles)

        print("building simulation executable")
        process = subprocess.Popen(compileList,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        stdout, stderr = process.communicate()

        if process.returncode != 0:
            print("Error building ./sim")
            if len(stderr) > 0:
                print(stderr)
            exit(1)

        os.chdir('../../')

    if xpenv is not None:
        # the directory holding the script (build.py) should have subdirectory xlsx
        xlsxDir = os.path.join(scriptDir, 'xlsx')
        if not os.path.isdir(xlsxDir):
            print('error: expected subdirectory {}'.format(xlsxDir))
            exit(1)

        xlsxInputDir = os.path.join(xlsxDir, 'input') 
        if not os.path.isdir(xlsxInputDir):
            print('error: expected subdirectory {}'.format(xlsxInputDir))
            exit(1)

        inputFile = os.path.join(xlsxInputDir, xlsx_input_file)
        if not os.path.isfile(inputFile):
            print('error: expected xlsxPCES input file {}'.format(inputFile))
            exit(1)

        # ensure existence of convert where the environment variable indicated it would be
        convertDir = os.path.join(xpenv,'convert')
        if not os.path.isdir(convertDir):
            print('error: expected directory {}'.format(convertDir))
            exit(1)

        convertScript = os.path.join(xpenv,'runConvert.py')
        # ensure that the main conversion script 'runConvert.py' is present
        if not os.path.isfile(convertScript):
            print('error: expected script {}'.format(convertScript))
            exit(1)
     
        # now ensure that all the conversion scripts we need in convert are present
        # these are convert-cp.py, convert-exec.py, convert-experiments.py, convert-map.py, convert-netparams.py, convert-topo.py
        sheets = ('cp', 'exec', 'experiments', 'map', 'netparams', 'topo')
        scripts = []
        for sheet in sheets:
            scripts.append( os.path.join(convertDir, 'convert-{}.py'.format(sheet) ))

        errs = 0
        for script in scripts:
            if not os.path.isfile(script):
                print('error: expected script {}'.format(script))
                errs += 1

        if errs > 0:
            exit(1)
     
        buildxlsxArgs(name, xpenv, xlsxDir)
        xlsxArgsDir = os.path.join(xlsxDir, 'args')

        # create an input file for convert-xlsx.py
        convertArgs = os.path.join(xlsxDir, 'args-convert')

        with open(convertArgs, 'w') as wf:
            print('-working {}'.format(os.path.join(xlsxDir,'working')), file=wf)
            print('-name {}'.format(name), file=wf)
            print('-xlsx {}'.format(inputFile), file=wf)
            print('-csvDir {}'.format(os.path.join(xlsxDir,'csvDir')), file=wf)
            print('-yamlDir {}'.format(os.path.join(scriptDir, 'templates')), file=wf)
            print('-templateDir {}'.format(os.path.join(xlsxDir,'templateDir')), file=wf)
            print('-descDir {}'.format(os.path.join(xlsxDir,'descDir')), file=wf)
            print('-argsDir {}'.format(xlsxArgsDir), file=wf)
            print('-build', file=wf)

        # run the xlsxPCES conversion script 
        cmdLine = ["python3", convertScript , "-is", convertArgs]
        process = subprocess.Popen(cmdLine,
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        stdout, stderr = process.communicate()

        if process.returncode != 0:
            print("Error running xlsxPCES")
            if len(stderr) > 0:
                print(stderr)
            exit(1)
        elif len(stdout) > 0:
            print(stdout)


if __name__ == "__main__":
    main()


