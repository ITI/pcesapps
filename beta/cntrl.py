#!/usr/bin/python3
import matplotlib.pyplot as plt
from matplotlib.patches import Polygon
from matplotlib.patches import Rectangle
import numpy as np
import yaml
import pdb
import sys
import os
import subprocess
import copy
import time

# gui.py passes a single file, cntrl.yaml, on the command line to cntrl.py
# readExp reads the dictionary it contains and returns it to the caller
def readExp(file):
    with open(file,'r') as rf:
        expDesc = yaml.safe_load(rf)
    return expDesc


# buildExpDesc transforms the description of a particular experiment
# in variables expressed by the gui to variables understood by the simulator.
# The main points are to take out references to the SSL server when it is not
# selected, and to set the bandwidth assignments to SSL (when needed) and the private
# router, as these devices stradle pvtNet and pubNet (and the gui expresses bandwidth
# for interfaces in terms of the networks on which they are resident 
def buildExpDesc(expDesc):
    maxLS = str(max(int(expDesc['pvtNetBw']), int(expDesc['pubNetBw'])))
    codePairs = {'srcCPU':'srcCPU',
        'srcCPUBw':'pvtNetBw',
        'pvtNetBw':'pvtNetBw',
        'pubNetBw':'pubNetBw',
        'pvtSwitch':'pvtSwitch',
        'pubSwitch':'pubSwitch',
        'pvtSwitchBw':'pvtNetBw',
        'pubSwitchBw':'pubNetBw',
        'pvtRtr':'rtr',
        'pubRtr':'rtr',
        'pvtRtrBw':'pvtNetBw',
        'pubRtrBw':'pubNetBw', 
        'sslCPU':'sslCPU',
        'sslCPUBw':'pvtNetBw',
        'pcktlen':'pcktLen',
        'pcktburst':'pcktBurst',
        #'eudcycles':'eudCycles',
        'pcktMu':'pcktMu',
        #'burstMu':'burstMu',
        #'cycleMu':'cycleMu',
        'euds':'euds',
        'eudCPU':'eudCPU',
        'eudCPUBw':'pvtNetBw',
        'cryptoalg':'crypto', 
        'keylength':'keylength', 
    }

    baseExp = {}
    for key, value in codePairs.items():
        if key.find('ssl') == -1 or expDesc['arch'] == 'SSL':
            baseExp[key] = expDesc[value]
    if expDesc['arch'] == 'SSL':
        baseExp['sslCPUBw'] = maxLS
        baseExp["sslsrvr"] = "True"
    else:
        baseExp["sslsrvr"] = "False"

    baseExp['pvtRtrBw'] = maxLS
    return baseExp


numExp = 1


# a set of experiments goes through all combinations of the contents
# of a menu associated with a 'base' parameter, and the contents of
# a menu associated with an 'attrb' parameter.  Each individual
# simulation experiment selects one value from the base parameter menu for the base parameter,
# and one from the attrb parameter menu for the attribute parameter.
#  Therefore the description of each experiment is slightly different.
# buildCmdLineDicts builds a list of all those variants
#
def buildCmdLineDicts(expDesc):
    global numExp

    baseParam = expDesc['baseParam']
    attrbParam = expDesc['attrbParam']

    # set the number of base parameters, allowing for the possibility
    # that the gui did not select a base parameter, in which case
    # the base parameter is set to 'None' and the representation of 
    # its menu places one value in the menu, 'None'
    numBase = len(expDesc['baseParamList'])
    if numBase==0 or baseParam in ('None','none'): 
        expDesc['baseParamList'] = ['None']
        numBase = 1
        expDesc['baseParam'] = 'None'

    # set the number of attrb parameters, allowing for the possibility
    # that the gui did not select a attrb parameter, in which case
    # the attrb parameter is set to 'None' and the representation of 
    # its menu places one value in the menu, 'None'
    numAttrb = len(expDesc['attrbParamList'])
    if numAttrb==0 or attrbParam in ('None','none'): 
        expDesc['baseAttrbList'] = ['None']
        numAttrb=1
        expDesc['attrbParam'] = 'None'

    # loop through all combination of base and attrb parameters,
    # generating codes placed in experiment descriptor attributes
    # 'baseParam' and 'attrbParam' which indicate which variable experimental
    # parameter is set and to what value
    cmdLineDicts = []
    for baseValue in expDesc['baseParamList']:

        # make a clean copy of the basic experimental dictionary passed in as input
        cld = copy.deepcopy(expDesc)

        # if the base parameter value is specified, set its value
        # in the experiments parameter dictionary to the next value in the menu
        if baseValue not in ('None','none'):
            cld[baseParam] = baseValue
            baseParamCode = baseParam+'='+baseValue
        else:
            baseParamCode = 'None'

        # inner loop: given a base parameter value, iterate over the attrb menu values 
        for attrbValue in expDesc['attrbParamList']:

            # if the attrb value is specified, set its value in the experiment parameter
            # dictionary being constructed
            if attrbValue not in ('None','none'):
                cld[attrbParam] = attrbValue
                attrbParamCode = attrbParam+'='+attrbValue
            else:
                attrbParamCode = 'None'

            # add the descriptive codes for the values just set
            cmdLineDict = {'baseParam':baseParamCode, 'attrbParam':attrbParamCode}

            # transform experiment description from GUI dictionary key values to bld.go dictionary key values
            cmdLineDict['cmdDict'] = buildExpDesc(cld)        

            # add completed simulation model description to the list under construction
            cmdLineDicts.append(cmdLineDict)

    # remember how many unique experiments will be run
    numExp = numAttrb*numBase

    # return answer
    return cmdLineDicts


# createBldArgs creates an input file for the bld.go application from the 
# experimental model description it is passed, writes it into file args-bld
# which is in bld-dir/args-bld (from the point of view of cntl.py)
def createBldArgs(cld, passthru):
    bldArgs = []

    if not os.path.isfile('./bld-dir/args-bld-base'):
        print('args-bld-base file not found', flush=True)
        exit(1)

    with open('./bld-dir/args-bld-base','r') as rf:
        bldArgs = rf.readlines()

    passthruArgs = []
    if len(passthru) > 0:
        with open(passthru,'r') as rf:
            passthruArgs = rf.readlines() 
    bldArgs.extend(passthruArgs)

    for key,value in cld['cmdDict'].items():
        bldArgs.append('-{} {}\n'.format(key,value))

    with open('./bld-dir/args-bld','w') as wf:
        for arg in bldArgs:
            wf.write(arg)

# The 'raw' list contains the results reported from the simulator.
# it transforms the reported min, 25quartile, mean, median, 75quartile and max
# into a set of values that, when analyzed statistically (by the pyplot program)
# yield those statistics
#
def extractBoxData(raw):
    # raw looks like
    # 'Comp Pattern class  has spread 0.000658, 0.000658, 0.00690, 0.000788, 0.000788, 0.000788'

    # extract the sequence of numbers
    raw = raw.replace(',','')
    words = raw.split()
    m = []
    for word in words:
        test = word.replace('.','')
        if test.isnumeric():
            m.append(float(word))

    # m[0] - least value
    # m[1] - 25 percentile
    # m[2] - mean
    # m[3] - median
    # m[4] - 75 percentile
    # m[5] - max

    # transform into data set whose box plot gives these same quartiles
    d = [0.0]*6
    d[0] = m[0]
    d[5] = m[5]
    d[1] = m[1]
    d[4] = m[4]

    x0 = 6*m[2]-(m[0]+m[1]+m[3]+m[4]+m[5])
    if x0 < m[3]:
        d[2] = x0
        d[3] = m[3]
    else:
        d[3] = x0
        d[2] = m[3]

    # turn into milliseconds
    for idx in range(len(d)):
        d[idx] = 1000*d[idx]

    return d 


# return the min and max values of the input list L
def extrema(L):
    minV = L[0]
    maxV = L[0]
    for i in range(len(L)):
        minV = min(minV,L[i])
        maxV = max(maxV,L[i])
    return minV, maxV


# buildPlot is called after all the experiments have completed,
# and their transformed data sets have been saved in 'boxPlot'
# It embeds the pyplot commands to build the plot.
# Input dictionary expDesc has some attributes that describe what is in boxPlot
#
def buildPlot(expDesc):
    numBase  = len(expDesc['baseParamList'])
    numAttrb = len(expDesc['attrbParamList'])
    invKeyConv = expDesc['invKeyConv']

    orgBaseKey  = invKeyConv[expDesc['baseParam']]
    orgAttrbKey = invKeyConv[expDesc['attrbParam']]

    # attrbSet will hold a list of the attribute values in the experiment,
    # to be used in axis labeling 
    attrbSet = []

    # data will be a list of lists, each interior list holding the data
    # points for an experiment.  len(data) is the number of experiments
    data = []

    # will hold the 'global' minimum and maximum y value in the entire data set
    gMinV = -1.0
    gMaxV = -1.0

    # boxes will get colors as a function of which attribute value 
    # contribute, we cycle through the attrb_colors below by increasing of attrib parameter index
    attrb_colors = ['b','g','r','y','m','c']


    # bands on the x-axis will have colors from base_colors, cycled through by increasing
    # base parameter index
    base_colors = ['lightblue','cornsilk','pink','salmon','goldenrod']


    # organize the data into lists that will be box-plotted.
    # gather the minimum and maximum values for the purposes of sizing the graph
    for baseValue in expDesc['baseParamList']:

        # attrb normally holds x-axis labels from the attrib parameter list, but if there
        # is none use the base value (because there will be no 'spread' across attrib values in an
        # adjacent set of columns)
        if orgAttrbKey == 'None':
            attrbSet.append(baseValue)

        for attrbValue in expDesc['attrbParamList']:

            # get the extrema for the list associated with one particular experiment
            minV, maxV = extrema(boxPlot[baseValue][attrbValue])

            # update global extrema
            if gMinV < 0.0 or minV < gMinV:
                gMinV = minV
            if gMaxV < 0.0 or maxV > gMaxV:
                gMaxV = maxV

            # add the list of the experiment's statistics as an element in the 'data' list of lists
            data.append(boxPlot[baseValue][attrbValue])
            if orgAttrbKey != 'None':
                attrbSet.append(attrbValue)

    # we don't to have box plots from different base parameters be immediately adjacent with 
    # the same spacing as plots with different attribute parameters but the same base value
    # We therefore put values in tickAxis to create spacing between those groups
    tickAxis = []
    counter = 1
    plotTick = []

    # compute the number of ticks between groups of attrb plot lines

    for pos, baseValue in enumerate(expDesc['baseParamList']):
        for tick in range(0, len(expDesc['attrbParamList'])):
            # within a group, each attribute parameter gets a new tick
            tickAxis.append(counter)
            plotTick.append(counter)
            counter += 1

    for idx in range(0,3):
        tickAxis.append(counter)
        counter += 1

    # each tick is given a label, some of them empty
    tickLabels = [' ']*len(plotTick)

    # the plot will be 4.5 x 4.5 inches, sized to fit within the gui window (of 5 x 5 inches_
    fig, ax = plt.subplots(figsize=(4.5,4.5))
    #fig.canvas.manager.set_window_title('pces Model Performance')
    fig.subplots_adjust(left=0.075, right=0.95, top=0.9, bottom=0.25)

    bx = len(expDesc['attrbParamList'])/2
    by = gMinV/2

    bottom = 0
    top = 0.025

    if orgBaseKey != 'None' and orgAttrbKey != 'None':  

        hgtDelta = 0.05
        #baseMenuHgt = 1.05*gMaxV/2
        baseMenuHgt = 0.5
        fig.text(0.825, baseMenuHgt, orgBaseKey, backgroundcolor='w', color='b',
            weight='roman', size = 'small')

        for idx in range(len(expDesc['baseParamList'])):

            left  = 1.0+ idx*(numAttrb)-0.25
            right = left+numAttrb

            ax.add_patch( Rectangle((left,bottom), numAttrb-0.5, height=1.05*gMaxV,
                angle=0.0, edgecolor=base_colors[idx], facecolor=base_colors[idx], alpha=1.0, linewidth=5))
            #ax.add_patch( Rectangle((left,bottom), (numAttrb-1), height=0.10, 
            #    angle=0.0, edgecolor=base_colors[idx], facecolor=base_colors[idx]))
          
            fig.text(0.825, baseMenuHgt-(idx+1)*hgtDelta, expDesc['baseParamList'][idx], backgroundcolor=base_colors[idx],
                color = 'black', weight='roman', size= 'x-small')

    

    # create a boxplot for every list contained in 'data'
    bp = ax.boxplot(data, notch=False, sym='+', vert=True, whis=1.5)

    # set graphical descriptions of those boxes
    plt.setp(bp['boxes'], color='black')
    plt.setp(bp['whiskers'], color='black')
    plt.setp(bp['fliers'], color='red', marker='+') 

    # put light grey grid lines to help indicate values the boxes represent,
    # and the attribute parameter values on the x-axis which which they are associated
    ax.yaxis.grid(True, linestyle='-', which='major', color='lightgrey',
               alpha=0.5)

    ax.xaxis.grid(True, linestyle='-', which='major', color='lightgrey',
               alpha=0.5)

    # create a title for the plot, depending on the base and attribute parameters
    gtitle = 'RTT as function of {} and {}'.format(orgBaseKey, orgAttrbKey)
    xl = orgAttrbKey
    if orgBaseKey == 'None' and orgAttrbKey == 'None':
        gtitle = 'RTT measurements'
        xl = ''
    elif orgBaseKey == 'None':
        gtitle = 'RTT as function of {}'.format(orgAttrbKey)
    elif orgAttrbKey == 'None':
        gtitle = 'RTT as function of {}'.format(orgBaseKey)
        xl = ''

    # plunk in titles and labels for x and y axes 
    ax.set(
        axisbelow=True,  # Hide the grid behind plot objects
        title=gtitle,
        xlabel= xl,
        ylabel='RTT (ms)',
    )

    # num_boxes is the number of box plotted

    num_boxes = len(data)

    #medians = np.empty(num_boxes)
    for i in range(num_boxes):

        # build a colored in rectangle
        box = bp['boxes'][i]
        box_x = []
        box_y = []
        for j in range(5):
            box_x.append(box.get_xdata()[j])
            box_y.append(box.get_ydata()[j])
        box_coords = np.column_stack([box_x, box_y])
        ax.add_patch(Polygon(box_coords, facecolor = attrb_colors[i%numAttrb]))

        med = bp['medians'][i]
        median_x = []
        median_y = []
        for j in range(2):
            median_x.append(med.get_xdata()[j])
            median_y.append(med.get_ydata()[j])
            ax.plot(median_x, median_y, 'k')
        #medians[i] = median_y[0]
        # Finally, overplot the sample averages, with horizontal alignment
        # in the center of each box
        ax.plot(np.average(med.get_xdata()), np.average(data[i]),
                 color='w', marker='*', markeredgecolor='k')

    # Set the axes ranges and axes labels
    # Add space on the right for base parameter labels

    ax.set_xlim(0.0, len(tickAxis))
    #ax.xaxis.set_ticks(plotTick)
 
    ax.set_ylim(0.0, 1.05*gMaxV)

    if orgAttrbKey != 'None' and orgBaseKey != 'None':
        ax.set_xticklabels(attrbSet, rotation=270, fontsize=10)
    else:
        ax.set_xticklabels(attrbSet, rotation=0, fontsize=10)


    
    base, ext = os.path.splitext(expDesc['plotFile'])
    try:
        plt.savefig(base+'.png', bbox_inches='tight')
        plt.savefig(base+'.pdf', bbox_inches='tight')
    except:
        print('error creating plots with base {}'.format(base))

    print('Plot files {} and {} created ...'.format(base+'.png', base+'.pdf'), flush=True)


boxPlot = {}
def main():
    global boxPlot

    expDesc  = readExp(sys.argv[1])

    # make sure we can create a plot file
    try:
        if not 'plotFile' in expDesc:
            print('specify name of file to which experiment plot is written', '...', flush=True)
            exit(1)

        base, ext = os.path.splitext( expDesc['plotFile'] )
        base = base+'.png'

        with open(base,'w') as wf:
            wf.write('msg')
        os.remove(base)

    except:
        print('unable to write to plotFile', expDesc['plotFile'], '...', flush=True)
        exit(1)
 
    cmdLineDicts = buildCmdLineDicts(expDesc)
    expCount = 1

    os.chdir('./bld-dir')
    if not os.path.isfile("./bld"):
        cmd = "go build bld.go"
        os.system(cmd)
    os.chdir('../')

    for cld in cmdLineDicts:

        createBldArgs(cld, expDesc['passthru'])

        # run the build app
        os.chdir('./bld-dir')
        cmd = "./bld -is args-bld"
        os.system(cmd)
        os.chdir('../')

        if not os.path.isfile("./sim-dir/sim"):
            os.chdir('./sim-dir')
            cmd = "go build sim.go"
            os.system(cmd)
            os.chdir('../')

        with open(expDesc['expCounter'],'w') as wf:
            msg = 'running experiment {} of {} ...'.format(expCount,numExp)
            print(msg, flush=True) 
            wf.write(msg+"\n")
            expCount += 1

        os.chdir('./sim-dir')
        result = subprocess.run(
            ['./sim','-is','args-sim'],
            capture_output=True,
            text=True 
        )
        results = result.stdout

        dataSet = extractBoxData(results)

        baseCode  = cld['baseParam']
        attrbCode = cld['attrbParam']

        if baseCode == 'None':
            baseName = 'None'
            baseValue = 'None'
        else:
            (baseName, baseValue) = baseCode.split("=")

        if attrbCode == 'None':
            attrbName = 'None'
            attrbValue = 'None'
        else:
            (attrbName, attrbValue) = attrbCode.split("=")

        if baseValue not in boxPlot:
            boxPlot[baseValue] = {}
        boxPlot[baseValue][attrbValue] = dataSet
        os.chdir('../')

    print("creating plot ...", flush=True)
    buildPlot(expDesc)
    with open(expDesc['expCounter'],'w') as wf:
        msg = 'Done\n'
        wf.write(msg)
 
if __name__ =="__main__":
    main()


