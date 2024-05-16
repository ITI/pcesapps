#!/bin/python3

# script that uses tkinter library to create a GUI for the MrNesbits
# simulation running (only) on the alpha version demo architecture.
#
from tkinter import *
from tkinter import scrolledtext
import subprocess
import pdb

# list for device pull-down menu.  Placing here for access by selection initialization
allDevs = [
        "pcktSrc",
        "pvtSwitch",
        "pvtRouter",
        "ssl",
        "pubRouter",
        "pubSwitch",
        "eudDev"
        ]

# list for full selection of CPUs ("pcktSrc" and "ssl")
allCPUs = [
        "x86",
        "i3",
        "i7"
        ]

# list for selection of slower CPUs for EUD
slowCPUs = [
        "x86",
        "i3"
        ]

switchmodels = [
        "FS_S3900-48T6S-R",
        "FS_N8560-32C"
        ]

routermodels = [
        "Cisco_Catalyst_8200",
        "Juniper_MX_240"
        ]

# list for selection of fast linkspeeds
fastLS = [
        "1000",
        "10000"
        ]

slowLS = [
        "100",
        "1000"
        ]

allLS = [
        "100",
        "1000",
        "10000"
        ]

# list for crypto algorithm
cryptoAlgs = [
        "none",
        "rsa-3072",
        "aes"
        ]

nocryptoAlgs = [
        "none"
        ]


# selection holds the values for device CPU/Model, Link-speed, and crypto-algorithm (for the two
# platforms that might use it
#   GUI displays the contents of this table, and allows a user to update various values.
# When the simulation execution is requested the values of this table are coded into an input file
# to be used by a script that runs the simulation
#
selection = [["Device","CPU/Model","Link-speed","crypto"],
    ["pcktSrc", allCPUs[0], fastLS[0], "none"],
    ["pvtSwitch", switchmodels[0], fastLS[0],"none"],
    ["pvtRouter", routermodels[0], allLS[0],"none"],
    ["ssl", allCPUs[-1], fastLS[-1], "none"],
    ["pubRouter", routermodels[0], slowLS[0],"none"],
    ["pubSwitch", switchmodels[-1], slowLS[0],"none"],
    ["eudDev", slowCPUs[0], slowLS[0], "none"]]

# identify matrix entries that are always the same
equivalence = {(4,3):[(7,3)], (7,3):[(4,3)]}

def devRow(name):
    for rowIdx in range(1, len(selection)):
        if name==selection[rowIdx][0]:
            return rowIdx
    return -1

# choose new device to focus on. Replace the CPU/Model widget, the Linkspeed widget, and the crypto widget
# depending on what the device focus was, and has become
def updateFocus(m,n,x):
    devFocus = devName.get()
    # remove the previous widgets    
    # forget all the widgets in variable places
    for loc, (widget, variable) in activeWidget.items():
        widget.grid_remove()

    rowIdx = devRow(devFocus)

    # add the widgets corresponding to this device
    for (widget, variable, i, j) in updateDict[devFocus]:
        widget.grid(row=i,column=j)
        
        # set the menu variable
        if variable is not None:
            variable.set(selection[rowIdx][j])

        # set the active widget
        loc = (i,j)
        activeWidget[loc] = (widget, variable)


def refreshTable():
    for i in range(0, len(selection)):
        for j in range(0, len(selection[0])):
            value = selection[i][j]
            tableEntry[i][j].delete(0,END)
            tableEntry[i][j].insert(END, value)


def updateSelect():
    # need to update the values in table display and change selection

    # the (i,j) in activeWidget refer to the grid layout of the matrix of widgets.
    # This means that for the selection and tableEntry matrices we need to 
    # subtract one from j because the matrix has two more 'extra' columns on the left than
    # does selection
    devFocus = devName.get()
    rowIdx = devRow(devFocus)

    for (i,j), (widget, variable) in activeWidget.items():
        if variable is None:
            continue
    
        value = variable.get()

        updatePairs = []
        updatePairs.append((rowIdx, j))

        if (rowIdx,j) in equivalence:
            updatePairs.extend(equivalence[(rowIdx,j)])
        
        for (i,j) in updatePairs:
            selection[i][j] = value
            tableEntry[i][j].delete(0,END)
            tableEntry[i][j].insert(END, value)

    refreshTable()

# argDict helps translate the values in the selection table to the input file for the simulation run.
# Given the command flag for the simulation run, the associated tuple gives the location of the parameter
# being referenced
argDict = {"-srccpu":(1,1), "-srcls":(1,2),
           "-swpubm":(6,1),"-swpvtm":(2,1), "-swpvtls":(2,2), "-swpubls":(6,2),
           "-rtrpvtm":(3,1),"-rtrpubm":(5,1),"-rtrpvtls":(3,2), "-rtrpubls":(5,2),
           "-sslcpu":(4,1),"-sslls":(4,2),"-eudcpu":(7,1),"-eudls":(7,2),"-cryptoalg":(4,3)}

# stopExp is called when the button to exit the GUI is pushed
def stopExp():
    exit(1)

# runExp is called whent the button to run the simulation is pushed
def runExp():

    # auxilary function to tell whether a string is in floating point format
    def isfloat(v):
        return v.replace(".","").isnumeric()

    # auxilary function to tell whether the output from the simulation run indicates
    # a successful simulation and if so a message giving the execution time

    # execTime is given a line of words from the simulation output
    def execTime(line):

        # separate out the words
        vp = line.split()

        # look for a sequence where a word that is a floating point number is followed by the word "seconds"
        for i in range(0, len(vp)-1):
            if isfloat(vp[i]) and vp[i+1]=="seconds":
                # our answer
                return True, "Execution time is {} seconds".format(vp[i])
       
        # evidently was not successful
        return False, "Simulation run did not complete successfully. See terminal window"


    # create the simulation input file from the selections
    with open("./args-run","w") as f:
        for key, vtup in argDict.items():
            value = selection[vtup[0]][vtup[1]]
            f.write(key+" "+value+"\n")

    # run the experiment and get the output
    result = subprocess.run(['python3', './run.py', '-is', 'args-run'], stdout=subprocess.PIPE, text=True)

    # clear out the display
    expText.delete('1.0',END)

    # split the output into separate lines
    resultlines = result.stdout.splitlines()

    # for each line see whether an execution time was reported
    for line in resultlines:
        found, msg = execTime(line)
        if found:

            # found an execution time, display it and leave
            expText.insert(END,msg)
            return

    # no line held an execution time
    expText.insert(END, "Simulation run did not complete successfully. See terminal window")


# data structures for holding and displaying current state of parameter selections prior to execution
tableEntry = [] 

# selection is a global array of arrays, the Table construction makes an Entry widget of
# each of these.  The tableEntry array of arrays of Entry widgets is created to give the code access
# to the entries and their values for updates
class Table:
    def __init__(self,root):
        total_rows = len(selection)
        total_columns = len(selection[0])
         
        # code for creating table
        for i in range(total_rows):
            tableEntryRow = []
            for j in range(total_columns):
                self.e = Entry(root, width=20, fg='blue', bg='white',
                               font=('Arial',16,'bold'))
                tableEntryRow.append(self.e)
                self.e.grid(row=i, column=j)
                self.e.insert(END, selection[i][j])
            tableEntry.append(tableEntryRow)


# The GUI's root frame has 4 others, arranged by .grid in a single column
#  selectFrame holds the table of selected parameter values
#
#  hostFrame holds widgets for identifying a computing host, the CPU it should have, 
#    the line speed of its interfaces, and a button to cause the selection revealed in the widgets
#    to be copied into the selection matrix

#  deviceFrame holds widgets for identifying a switch or router, the model it should have, 
#    the line speed of its interfaces, and a button to cause the selection revealed in the widgets
#    to be copied into the selection matrix

# algFrame holds widgets for selecting a crypto algorithm and updating it.

# expFrame holds a button to execute the simulation, a text window reporting the result
# and a button to cause the GUI to exit
#
# Each of the parameter selections is structured with a descriptive label directly above a pull-down menu
# that has all the supported options for that parameter.    To set the parameters for a given device 
# (host, eud, switch, or router) you select the name of the device using a pull-down menu, 
# select the parameters using their pull-down menus, and push the UPDATE button.
#
# Once the full set of parameters desired is shown by display, you can run the simulation pushing the
# EXECUTE button.

root = Tk()
root.title('MrNesbits alpha demo')

selectFrame = LabelFrame(root, text="Configuration Parameter Selection State", bg='white')
selectFrame.grid(row=0, column=0)
t = Table(selectFrame)

# create the frames
displayFrame = LabelFrame(root, text="Parameter Assignment", bg='white')
devFrame = LabelFrame(root, text="Parameter Selection", bg='white')
expFrame = LabelFrame(root, text="Experiment",bg='white')

# place them in root, stacked
displayFrame.grid(row=0,column=0)
devFrame.grid(row=1, column=0)
expFrame.grid(row=2, column=0)


#labels for menus in devFrame
devText = Entry(devFrame, width=14, bg='yellow', fg='black')
devText.insert(END, "Device Name")

cpuText = Entry(devFrame, width=12, bg='yellow', fg='black')
cpuText.insert(END, "CPU Type")

modelText = Entry(devFrame, width=15, bg='yellow', fg='black')
modelText.insert(END, "Device Model")

speedText = Entry(devFrame, width=20, bg='yellow', fg='black')
speedText.insert(END,"Link-speed (Mbytes/s)")

cryptoText = Entry(devFrame, width=10, bg='yellow', fg='black')
cryptoText.insert(END,"Crypto")



# Tktinker variable names for widgets
devName = StringVar()
cpuName = StringVar()
modelName = StringVar()
speedName = StringVar()
cryptoName = StringVar()

# widget for device selection
devMenu = OptionMenu(devFrame, devName, *allDevs)

# widget for choosing all cpus
allCPUMenu = OptionMenu(devFrame, cpuName, *allCPUs)

# widget for choosing slow cpus
slowCPUMenu = OptionMenu(devFrame, cpuName, *slowCPUs)

# widget for choosing switch model
switchModelMenu = OptionMenu(devFrame, modelName, *switchmodels)

# widget for choosing router model
routerModelMenu = OptionMenu(devFrame, modelName, *routermodels)

# widget for choosing slow link-speeds
slowSpeedMenu = OptionMenu(devFrame, speedName, *slowLS)

# widget for choosing fast link-speeds
fastSpeedMenu = OptionMenu(devFrame, speedName, *fastLS)

# widget for choosing fast link-speeds
allSpeedMenu = OptionMenu(devFrame, speedName, *allLS)

# widget for choosing crypto algorithm when we can
cryptoMenu = OptionMenu(devFrame, cryptoName, *cryptoAlgs)

# widget for choosing no crypto algorithm
nocryptoMenu = OptionMenu(devFrame, cryptoName, *nocryptoAlgs)

# set initial values for variables.  Assume that computing device pcktSrc is selected
devName.set("pcktSrc")  # pcktSrc
devFocus = "pcktSrc"

# call when devName changes
devName.trace("w", updateFocus)

devIdx = 0
while devIdx < len(selection):
    if selection[devIdx][0] == 'pcktSrc':
        break
    devIdx += 1

cpuName.set(selection[devIdx][1])      # available to all computing devices
modelName.set(switchmodels[0])  # won't be displayed
speedName.set(selection[devIdx][2])   # common to all CPU/EUDs
cryptoName.set(selection[devIdx][3])  # start with nothing

# non-Tktinker variables that reflect these, don't change immediately with changes in menu
devNameBckup = devName.get()
cpuNameBckup = cpuName.get()
modelNameBckup = modelName.get()
speedNameBckup = speedName.get()
cryptoNameBckup = cryptoName.get()


# buttons for selecting and updating parameters
updateButton = Button(devFrame, command=updateSelect, padx=3, width=8, text="UPDATE", fg='red')

# placement of widgets
#        0               1               2                3                   4        
#  0 | devText | cpuText or modelText | speedText       |    cryptoText  | empty
#    --------------------------------------------------------------------------------------------------
#    | devMenu | one of (allCPUMenu,  | one of          | one of         |
#  1 |         |  slowCPUMenu,        | (slowSpeedMenu, | (cryptoMenu,   | updateButton
#    |         |  switchModelMenu,    | fastSpeedMenu,  | nocryptoMenu)  |
#    |         |  routerModelMenu)    | allSpeedMenu)   |                |
#    ---------------------------------------------------------------------------------------------------
#


devText.grid(row=0,column=0)
cpuText.grid(row=0,column=1)
speedText.grid(row=0,column=2)
cryptoText.grid(row=0,column=3)
devMenu.grid(row=1,column=0)
allCPUMenu.grid(row=1,column=1)
fastSpeedMenu.grid(row=1,column=2)
nocryptoMenu.grid(row=1,column=3)
updateButton.grid(row=1,column=4)

activeWidget = {}
activeWidget[(0,1)] = (cpuText, None)
activeWidget[(1,1)] = (allCPUMenu, cpuName)
activeWidget[(1,2)] = (fastSpeedMenu, speedName)
activeWidget[(1,3)] = (nocryptoMenu, cryptoName)

updateDict = {'pcktSrc': [(cpuText, None, 0,1), (allCPUMenu, cpuName, 1,1),(fastSpeedMenu, speedName, 1,2),(nocryptoMenu, cryptoName, 1,3)],
    'pvtSwitch':[(modelText, None, 0,1), (switchModelMenu, modelName, 1, 1), (fastSpeedMenu, speedName, 1, 2), (nocryptoMenu, cryptoName, 1,3)],
    'pvtRouter':[(modelText, None, 0,1), (routerModelMenu, modelName, 1, 1), (allSpeedMenu, speedName, 1, 2), (nocryptoMenu, cryptoName, 1,3)],
    'ssl': [(cpuText, None, 0,1), (allCPUMenu, cpuName, 1,1),(fastSpeedMenu, speedName, 1,2),(cryptoMenu, cryptoName, 1,3)],
    'pubSwitch':[(modelText, None, 0,1), (switchModelMenu, modelName, 1, 1), (slowSpeedMenu, speedName, 1, 2), (nocryptoMenu, cryptoName, 1,3)],
    'pubRouter':[(modelText, None, 0,1), (routerModelMenu, modelName, 1, 1), (slowSpeedMenu, speedName, 1, 2), (nocryptoMenu, cryptoName, 1,3)],
    'eudDev': [(cpuText, None, 0,1), (slowCPUMenu, cpuName, 1,1),(slowSpeedMenu, speedName, 1,2),(cryptoMenu, cryptoName, 1,3)]}

expButton = Button(expFrame, command=runExp, padx=3, width=8, text="EXECUTE", fg='blue')
expButton.grid(row=0, column=0)

expText = scrolledtext.ScrolledText(expFrame, width=100, height=2, wrap=WORD,
                font=("Times New Roman",20))

expText.insert(END,"simulation run output appears here")
expText.grid(row=1,column=0)

stopButton = Button(expFrame, command=stopExp, padx=3, width=12, text="EXIT GUI", fg='red')
stopButton.grid(row=3, column=0)

root.mainloop()
