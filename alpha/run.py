#!/usr/bin/env python3

# setup.py takes on the command line descriptions of variables in the 
# alpha-case architecture for BITs.  From these arguments it creates
# an 'experimental parameters update' file which is included with other
# model parameters when the simulation is run.

# The workflow for running an experiment is as follows.
# 1. The bld program is run to create the model topology and baseline
#    experiment parameters.  This need only be done once, by
#
#         % bld -is args-bld
#
#    args-bld is a list of command-line parameters for bld that give names
#    and locations to files used as input when running an experiment.
#    In this distribution all those files are written within the 'input' subdirectory
#
# 2. For each experimental run a set of parameters particular to the alpha-case
#    arechitecture are selected somehow (e.g. from the GUI).   Those parameters
#    are ultimately represented in a file input/expx.yaml which is input to the 
#    simulation run.  An included program xexpcfg generates 
#    input/expx.yaml to reflect selected parameters, via
#
#         % xexpcfg -is args-xexp
#
#    args-xexp is a list of command-line arguments that quantify all
#    the parameters we're defining for the alpha-case throw-away GUI.
#    
#    This script (setup.py) accepts those parameters (plus another, about which more anon)
#    on the command-line and generates args-xexp.   The flag names for setup.py are considerably
#    shorter than the corresponding flag names in args-xexp because the former can be expressed line-by-line
#    in a file, and we anticipate that setup.py will be called from within a script.
#    
#    The parameters for the alpha-case demo are as follows (citing both flag representations)
#    
#    parameter                                  xexpcfg        setup.py     selectable values
#    cpu type for src device                    -srcCPU         -scpu         {i7, x86}
#    link-speed for interface at src device     -srcLinkSpeed   -sls          positive floating point number
#    model type for Switch devices              -SwitchModel    -swm          {Slow, Fast}
#        


import argparse
import pdb
import os
import sys
import tempfile
import shutil
from enum import Enum

ap = argparse.ArgumentParser()

class argType(Enum):
    INT = 1
    STRING = 2
    FLOAT = 3


# list of the expected command line argument flags.  Each flags a value that follows it.
argStrs = ("-srccpu","-srcls","-swpubm","-swpvtm", "-swpvtls", "-swpubls", "-rtrpvtm", "-rtrpubm", "-rtrpvtls", "-rtrpubls", "-sslcpu","-sslls","-eudcpu","-eudls","-cryptoalg")

# identity of types needed for the loop that sets the command line arguments
argTypes = {"-srccpu":argType.STRING, 
    "-srcls":argType.FLOAT,
    "-swpvtm":argType.STRING, "-swpvtls":argType.FLOAT,   
    "-swpubm":argType.STRING, "-swpubls":argType.FLOAT,   
    "-rtrpvtm":argType.STRING, "-rtrpvtls":argType.FLOAT,   
    "-rtrpubm":argType.STRING, "-rtrpubls":argType.FLOAT,   
    "-sslcpu":argType.STRING, "-sslls":argType.FLOAT,
    "-eudcpu":argType.STRING, "-eudls":argType.FLOAT,
    "-cryptoalg":argType.STRING
}

def cmdLineWords(line):
    # separate by white space and coallese pieces between command flags
    pieces = line.split()
    words = []
   
    idx = 0
    while idx < len(pieces):
        words.append(pieces[idx])   # a flag 
        nonflag = []
        idx += 1
        while idx< len(pieces) and not pieces[idx].startswith('-'):
            nonflag.append(pieces[idx])
            idx += 1

        # replace white space with '_'
        if len(nonflag) > 0:
            if len(nonflag) > 1:
                words.append("'"+"_".join(nonflag)+"'")
            else:
                words.append(nonflag[0])

    return words


# add_argument for the parser tells it what to look for on the command line.
# This loop gives the name, sets the type, and indicates that the flag is required
for arg in argStrs:
    if argTypes[arg] == argType.INT:
        ap.add_argument(arg, type=int, required=True)
    elif argTypes[arg] == argType.FLOAT:
        ap.add_argument(arg, type=float, required=True)
    else: 
	    ap.add_argument(arg, required=True)

# the command line parameters can be spread out on the command line as per usual,
# or be listed in a file.  In the former case the command given is % python exp.py -is args-exp
# where ./args-exp is a file that holds the command line arguments.  In typical use
# I put one flag and value per line.
#  The loop below creates a list 'cmdline' which is either straight from sys.argv,
# or, if '-is' is present, opens the file named by '-is' and builds a cmdline list
# as though the arguments had been presented on the command line
cmdline = []
if len(sys.argv) < 3:
    print("command line parameters required")
    sys.exit(1)

if sys.argv[1] == '-is':
    with open(os.path.abspath(sys.argv[2]),'r') as cf:
        for line in cf.readlines():
            line = line.strip()

            # vect = line.split()
            vect = cmdLineWords(line)
            for idx in range(0, len(vect)):
                cmdline.append( vect[idx].strip() )
else:   
    cmdline = sys.argv[1:]

# parse the command line, the results are accessible through variable args
args = ap.parse_args(cmdline)

# make sure the model is built
# print("building model")
if not os.path.isfile("./bld-dir/bld"):
    os.chdir('./bld-dir') 
    cmd = "go build ./bld.go"
    os.system(cmd)
    os.chdir('../')

os.chdir('./bld-dir') 
cmd = "./bld -is args-bld"
os.system(cmd)
os.chdir('../')

# set the crypto algorithm by replacing whatever bld.go said it would be with
# the algorithm named on the exp.py command line.  This is accomplished by
# making a copy of ./input/cpInit.yaml (which has the line which defines the algorithm)
# to a scratch file, then run sed on that copy to make the replacement and write the copy
# back to ./input/cpInit.yaml. (N.B. for those practiced with sed...for reasons unknown in the
# development environment we used, the interactive switch -i which would normally do the editting
# in place didn't work.)

# create a name for a scratch file
tmp_dir = tempfile.mkdtemp()
tmp_file = os.path.join(tmp_dir,"scratch")

# create the name for input/cpInit.yaml
cpInitfile = os.path.join('./input', 'cpInit.yaml')

# copy cpInit.yaml to the scratch file
shutil.copy(cpInitfile, tmp_file)

# print("setting crypto algorithm in input/cpInit.yaml")

# create a string with the sed command that performs the replacement
cmd = "sed 's/algorithm: [a-z0-9-]*/algorithm: {}/' {} > {}".format(args.cryptoalg, tmp_file, cpInitfile)

# perform the replacement
os.system(cmd)


# print("creating input/mdfy.yaml")
if not os.path.isfile("./mdfy-dir/mdfy"):
    os.chdir('./mdfy-dir')
    cmd = "go build mdfy.go"
    os.system(cmd)
    os.chdir('../')

cmd = "./mdfy -is ../args-run"
os.chdir('./mdfy-dir')
os.system(cmd)
os.chdir('../')

# print("running model")
if not os.path.isfile("./sim-dir/sim"):
    os.chdir('./sim-dir')
    cmd = "go build sim.go"
    os.system(cmd)
    os.chdir('../')

cmd = "./sim -is args-sim"
os.chdir('./sim-dir')
os.system(cmd)
os.chdir('../')

