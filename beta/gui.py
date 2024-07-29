#!/bin/python3

# script that uses tkinter library to create a GUI for the MrNesbits
# simulation running (only) on the alpha version demo architecture.

from tkinter import *
from tkinter import scrolledtext
from PIL import Image, ImageTk
import subprocess
import pdb
import json
import yaml
import argparse
import os
import time
import copy

dbdir = ""
archdir = ""

def buildParser():
    global args, workingdir, archdir, imagedir, plotdir, passthruFile, dbdir

    ap = argparse.ArgumentParser()
    # script uses a 'working directory' for all the files it writes,
    #   and through which it communicates with cntrl.py
    #   - saved menus
    #   - expCounter.txt (communicating progress with an executing set of experiments)
    #   - plot files 
    ap.add_argument("-working", type=str, required=True, dest="working")

    ap.add_argument("-db", type=str, required=True, dest="db")

    # name of a file which, if it exists in the working directory, is used to 
    # initialize menus that the gui has modified from the default
    ap.add_argument("-rm", type=str, required=False, dest="read_menu")

    # name a a file, which when specified, is used to save the states of
    # modifiable menus when the gui is shut down through the exit button
    ap.add_argument("-wm", type=str, required=False, dest="write_menu")

    # directory from which script reads some architectural details to populate menu selections
    ap.add_argument("-arch", type=str, required=True, dest="arch")

    # name of file in arch directory that holds model descriptions of cpus, switches, euds, and routers
    # that can be selected
    ap.add_argument("-devDesc", type=str, required=True, dest="devDesc")

    # name of file in arch directory that holds names of crypto algorithms that can be selected
    # for experimental runs
    ap.add_argument("-cryptoDesc", type=str, required=False, dest="cryptoDesc")

    # name of file in current directory that contains command line variable assignments.
    # If the file name is present, those assignments are appended to the arguments passed to
    # cntrl.py
    ap.add_argument("-passthru", type=str, required=False, dest="passthru")

    # more typically read command line parameters from a file, flagged with '-is' switch
    cmdline = sys.argv[1:]
    if len(sys.argv) == 3 and sys.argv[1] == "-is":
        cmdline = []
        with open(sys.argv[2],"r") as rf:
            for line in rf:
                line = line.strip()
                if len(line) == 0 or line.startswith('#'):
                    continue
                cmdline.extend(line.split()) 

    # wherever it came from, parse the command line
    args = ap.parse_args(cmdline)

    # get the directory of the file we're executing
    homedir = __file__

    # get the fully qualified directory name
    cwd = os.getcwd()

    workingdir = args.working.replace('./','')
    workingdir = os.path.join(cwd, workingdir) 

 
    # check that working points to a working directory
    if not os.path.isdir(workingdir):
        print("declared working directory {} does not exist".format(workingdir))
        exit(1)

    dbdir = args.db
    # check that db points to a working directory
    if not os.path.isdir(dbdir):
        print("declared database directory {} does not exist".format(dbdir))
        exit(1)

    archdir = args.arch.replace('./','') 
    archdir = os.path.join(cwd, args.arch)

    # check that arch points to a working directory
    if not os.path.isdir(archdir):
        print("declared arch directory {} does not exist".format(archdir))
        exit(1)

    if args.passthru != None:
        passthruFile = os.path.join(cwd, args.passthru)
        if not os.path.isfile(passthruFile):
            print("declared pass through argument file {} does not exist".format(passthruFile))
            exit(1)


legalKeyLengths = {}

def processCmdArgs():
    global read_menu, write_menu, devDescMap, allCrypto, dbdir, archdir
    global allKeyLengths, defaultKeyLengths
    global legalKeyLengths

    # if selected, ensure that saved menu files are accessible
    read_menu = args.read_menu
    write_menu = args.write_menu

    if read_menu:
        read_menu = os.path.join(workingdir, read_menu)
    if write_menu:
        write_menu = os.path.join(workingdir, write_menu)

    # check read menu file
    if read_menu:
        try:
            with open(read_menu, "r") as rf:
                menus = json.load(rf)
        except:
            print('Unable to open named read menu file', read_menu)
            exit(1)

    # structure assumes that subdirectroy bld-dir exists and holds two
    # go files used for model construction.   bld-dir/bld.go constructs
    # the model software and hardware architecture.  bld-dir/table.go
    # constructs dictionaryies devExec and cryptoDesc which are commonly
    # used by the gui and by the simulation.  This sharing ensures that the
    # gui offers only the devices and timings that the simulation can use.

    # if the tables application doesn't exist, first create it

    # check for existence of cryptoDesc.yaml and devDesc.yaml
    here = os.getcwd()
    os.chdir(archdir)

    needDesc = not os.path.isfile('./cryptoDesc.yaml') or not os.path.isfile('.devDesc.yaml')
    os.chdir(here) 
    os.chdir(dbdir)
    if not os.path.isfile('./cnvrtExec'):
        os.system('go build cnvrtExec.go')

    if not os.path.isfile('./cnvrtDesc'):
        os.system('go build cnvrtDesc.go')

    # build descriptor files if needed
    if needDesc:
        os.system('./cnvrtExec -is args-cnvrt')
        os.system('./cnvrtDesc -is args-cnvrt')

    os.chdir(here)

    # read in the device description map, making sure it exists and
    # is either yaml or json
    devDesc = args.devDesc
    devDescFile = os.path.join(archdir, devDesc)

    if not os.path.isfile(devDescFile):
        print("device description file {} does not exist".format(devDescFile))
        exit(1)

    if devDescFile.find('.yaml') > -1 or devDescFile.find('.yml') > -1:
        with open(devDescFile, 'r') as rf:
            devDescDict = yaml.safe_load(rf)

    elif devDescFile.find('.json') > -1:
            devDescDict = json.load(rf)
    else:
        print("device description file must be .yaml or .json")
        exit(1)

    devDescMap = devDescDict['DescMap']

    # read in the crypto description map, making sure it exists and
    # is either yaml or json
    cryptoDesc = args.cryptoDesc
    cryptoDescFile = os.path.join(archdir, cryptoDesc)

    if not os.path.isfile(cryptoDescFile):
        print("crypto description file {} does not exist".format(cryptoDescFile))
        exit(1)

    if cryptoDescFile.find('.yaml') > -1 or cryptoDescFile.find('.yml') > -1:
        with open(cryptoDescFile, 'r') as rf:
            cryptoDescDict = yaml.safe_load(rf)

    elif cryptoDescFile.find('.json') > -1:
            cryptoDescDict = json.load(rf)
    else:
        print("crypto description file must be .yaml or .json")
        exit(1)

    allCrypto = []
    legalkeyLengths = {}
    keyPresent = {}
    for algdict in cryptoDescDict['algs']:
        algName = algdict['name'] 
        keylens = algdict['keylen']

        allCrypto.append(algName)
        legalKeyLengths[algName] = keylens
        for k in keylens:
            keyPresent[k] = True

    allKeyLengths = []
    for k in keyPresent:
        allKeyLengths.append(str(k))
    allKeyLengths.sort(key=lambda v:int(v))
    defaultKeyLengths = allKeyLengths

def InitMenus():
    global allCPUs, allSSLCPU, allEUDCPU, allpvtSwitch, allpubSwitch, allRtrs, allCrypto
    global allEUDs, allPcktLen, allPcktBurst, allPcktMu 
    global allBurstMu, allCycleMu, allPvtNetBw, allPubNetBw, allKeyLengths
    global paramLists, defaultMenu

    # pass through every device name and save it in the appropriate menus
    for devName, devMap in devDescMap.items():
        devTypes = devMap['devtype']
        for dtype in devTypes:
            if dtype == 'host' and not devName in allCPUs:
                allCPUs.append(devName)
                defaultCPUs.append(devName)
            if dtype == 'server' and not devName in allCPUs:
                allCPUs.append(devName)
                defaultCPUs.append(devName)
            if dtype == 'ssl' and not devName in allSSLCPU:
                allSSLCPU.append(devName)
                defaultSSLCPU.append(devName)
            if dtype == 'eud' and not devName in allEUDCPU:
                allEUDCPU.append(devName)
                defaultEUDCPU.append(devName)
            if dtype == 'switch' and not devName in allpvtSwitch:
                allpvtSwitch.append(devName)
                defaultpvtSwitch.append(devName)
            if dtype == 'switch' and not devName in allpubSwitch:
                allpubSwitch.append(devName)
                defaultpubSwitch.append(devName)
            if dtype == 'router' and not devName in allRtrs:
                allRtrs.append(devName)
                defaultRtrs.append(devName)

    # check that we have at least one entry for each menu
    if len(allCPUs) == 0:
        print('device description must contain entries for {}'.format('allCPUs'))
        exit(1)

    if len(allSSLCPU) == 0:
        print('device description must contain entries for {}'.format('allSSLCPU'))
        exit(1)

    if len(allpvtSwitch) == 0:
        print('device description must contain entries for {}'.format('allpvtSwitch'))
        exit(1)

    if len(allpubSwitch) == 0:
        print('device description must contain entries for {}'.format('allpubSwitch'))
        exit(1)

    if len(allRtrs) == 0:
        print('device description must contain entries for {}'.format('allRtrs'))
        exit(1)

    # When read_menu is not None, there is a dictionary 'menus' that was
    # read in earlier, whose keys are the names of the modifiable menus
    # and whose value for a key is the save contents of that menu
    if read_menu: 
        # for each menu in the dictionary, replace the built-in dictionary
        for menu in menus:
            if menu == 'allEUDs':
                allEUDs = menus[menu]
            elif menu == 'allPcktLen':
                allPcktLen = menus[menu]
            elif menu == 'allPcktBurst':
                allPcktBurst = menus[menu]
            elif menu == 'allPcktMu':
                allPcktMu = menus[menu]
            elif menu == 'allPvtNetBw':
                allPvtNetBw = menus[menu]
            elif menu == 'allPubNetBw':
                allPubNetBw = menus[menu]
            elif menu == 'allKeyLengths':
                allKeyLengths = menus[menu]

    # remember what all we read in
    defaultMenu["EUDs"] = copy.deepcopy(allEUDs)
    defaultMenu["CryptoAlg"] = copy.deepcopy(allCrypto)
    defaultMenu["PcktSrc CPU"] = copy.deepcopy(allCPUs)
    defaultMenu["SSL CPU"] = copy.deepcopy(allSSLCPU)
    defaultMenu["EUD CPU"] = copy.deepcopy(allEUDCPU)
    defaultMenu["PvtNet Switch"] = copy.deepcopy(allpvtSwitch)
    defaultMenu["PubNet Switch"] = copy.deepcopy(allpubSwitch)
    defaultMenu["Router"] = copy.deepcopy(allRtrs)
    defaultMenu["PvtNet Mbps"] = copy.deepcopy(allPvtNetBw)
    defaultMenu["PubNet Mbps"] = copy.deepcopy(allPubNetBw)
    defaultMenu["PcktLen"] = copy.deepcopy(allPcktLen)
    defaultMenu["PcktBurst"] = copy.deepcopy(allPcktBurst)
    defaultMenu["InterPckt delay"] = copy.deepcopy(allPcktMu)
    defaultMenu["Key Lengths"] = copy.deepcopy(allKeyLengths)

    paramLists = {'None':[], 'Architecture': allArchs, 'EUDs': allEUDs,
           'CryptoAlg': allCrypto,  'PcktLen': allPcktLen, 'PcktBurst': allPcktBurst, 
           'InterPckt delay': allPcktMu, 'PcktSrc CPU': allCPUs, 'SSL CPU': allSSLCPU,
           'EUD CPU': allEUDCPU, 'PvtNet Switch': allpvtSwitch, 'PubNet Switch': allpubSwitch,
           'Router': allRtrs, 'PvtNet Mbps': allPvtNetBw, 'PubNet Mbps': allPubNetBw, 'Key Lengths':allKeyLengths}

# SaveMenus is called when the gui is shut down.  If the write_menu flag
# had been set, the contents of all the modifiable menus are saved for later recovery    
def SaveMenus():
    if not write_menu:
        return

    sm = {'allEUDs': allEUDs, 'allPcktLen': allPcktLen, 'allPcktBurst': allPcktBurst, 
            'allPcktMu': allPcktMu, 'allPvtNetBw': allPvtNetBw, 
            'allPubNetBw': allPubNetBw, 'allKeyLengths':allKeyLengths}

    try:
        with open(write_menu,'w') as wf: 
            json.dump(sm, wf, indent=4)
    except:
        print("unable to open menu save output file", write_menu)


# the outputText widget is used to convey status and error messages to the
# user.  As the addition always necessitates first a deletion of whatever
# is present, we make a method to do this.  If msg is empty just clear the widget
def setOutputText(msg):
    outputText.delete('1.0',END)
    if len(msg) > 0:
        outputText.insert(END,msg)
    return

# addItem is called as a result of clicking a button to add an item to a menu.
# It looks up the menu being referenced and the text to add to the menu.
# There is probably a cleverer way to implement this than to enumerate over
# all the possible menus, except that adding an item might change the memory
# address of the menu, so some memory based trickery probably won't work.
# Perhaps some reflection based approach...but it was faster and easier
# to debug by copying and pasting code written for one menu, and using the editor's
# text replacement to change variable names as needed
def addItem():
    # get the name of the menu being modified.  The GUI has a menu 'modMenus' of labels
    # the GUI shows of all menus that can be modified by addItem (or deleteItem).
    # modMenu has a StringVar selectMenuName that holds the string currently selected
    # by modMenu.  Get that string
    smn = selectMenuName.get()

    # not all menus can have items added to them
    if smn not in addParamMenus:
        setOutputText('cannot add items to menu {}'.format(smn))
        return

    # the selectText Text widget has a string entered by the user giving the value to be added
    # to the menu identified by the value of selectMenuName.  Get the value and
    # clean out the selectText widget
    item = selectText.get(1.0,END)
    item = item.strip()
    selectText.delete(1.0,END)

    menu = paramLists[smn]

    # Check whether the user is asking to add an item that exists already in the selected menu
    # Variable smn holds a string identifier for the menu, the list 'paramLists' maps all possible
    # values smn may have to a menu list (paramList also maps to menus that smn cannot represent)

    # test every value in the menu with equality to what the user entered. Strip off any
    # white space at the front or back of this entry (although I'm pretty sure there won't be any)
    for entry in menu:
        if entry.strip() == item:
            # found a duplicate. Report to the outputText widget
            setOutputText("offered addition {} already exists in {} menu\n".format(item, smn)) 
            return

    setOutputText('')
 
    # go through every menu that might have an element added 
    if smn == 'EUDs':
        # validate that input item is positive integer no greater than 5000 
        item = item.strip()
        err = False
        if not item.isdigit():
           err = True
        else:
            if int(item) < 1 or int(item) > 5000:
                err = True
        
        if err:
            setOutputText("Number of EUDS must digit in [1,5000]")
            return

        # OK to proceded. Remove existing EUDS menu widget 
        (widget, variable) = menuWidget['EUDs']
        widget.grid_remove()    

        # add the new value to the list of all numbers of EUDs
        allEUDs.append(item)

        # make sure they are sorted (by integer value)
        allEUDs.sort(key=lambda v:int(v))

        # allEUDs list may have changed memory location as a result of
        # the addition, so reset paramLists to the potenitally new location
        paramLists['EUDs'] = allEUDs

        # create a new menu widget for the modification, using the same
        # previous name and new menu
        eudsMenu = OptionMenu(paramFrame, eudsName, *allEUDs)
        eudsFocus = item

        # variable from the menuWidget dictionary is eudsName
        variable.set(item)

        # we removed the old version of the menu widget, 
        # add the new one to the grid and save it in the dictionary we
        # maintain to look up widgets for removal
        eudsMenu.grid(row=0, column=4)
        menuWidget['EUDs'] = (eudsMenu, variable)
        setOutputText(item+" successfully added to EUDs menu")
        return


    elif smn == 'PcktLen':
        # validate that packet len is between 1 and 1500-36 (Ethernet Frame size minus IP/TCP headers)
        item = item.strip()
        err = False
        maxLen = 1500-36
        if not item.isdigit():
           err = True
        else:
            if int(item) < 1 or int(item) > maxLen:
                err = True
        
        if err:
            setOutputText("PcktLen must be in [1,{}]".format(maxLen))
            return

        # OK to procede. Remove existing PcktLenmenu and create new one
        (widget, variable) = menuWidget['PcktLen']
        widget.grid_remove()    

        # add the new value to the list of all numbers of EUDs
        allPcktLen.append(item)

        # make sure they are sorted (by integer value)
        allPcktLen.sort(key=lambda v:int(v))

        # allPcktLen list may have changed memory location as a result of
		# the addition, so reset paramLists to the potenitally new location
        paramLists['PcktLen'] = allPcktLen

        # create a new menu widget for the modification, using the same
		# previous name and new menu
        pcktLenMenu = OptionMenu(paramFrame, pcktLenName, *allPcktLen)
        pcktLenFocus = item

        # variable from the menuWidget dictionary is eudsName
        variable.set(item)

        # we removed the old version of the menu widget, 
        # add the new one to the grid and save it in the dictionary we
        # maintain to look up widgets for removal
        pcktLenMenu.grid(row=2, column=7)
        menuWidget['PcktLen'] = (pcktLenMenu, variable)

        setOutputText(item+" successfully added to PcktLen menu")
        return
        
    elif smn == 'PcktBurst':
        # validate that packet len is between 1 and 1500-36 (Ethernet Frame size minus IP/TCP headers)
        item = item.strip()
        err = False
        if not item.isdigit() or int(item) < 1:
            setOutputText("PcktBurst must be positive integer")
            return

        # OK to procede. Remove existing PcktBurstmenu and create new one
        (widget, variable) = menuWidget['PcktBurst']
        widget.grid_remove()    

        # add the new value to the list of all numbers of EUDs
        allPcktBurst.append(item)

        # make sure they are sorted (by integer value)
        allPcktBurst.sort(key=lambda v:int(v))

        # allPcktBurst list may have changed memory location as a result of
		# the addition, so reset paramLists to the potenitally new location
        paramLists['PcktBurst'] = allPcktBurst

        # create a new menu widget for the modification, using the same
		# previous name and new menu
        pcktBurstMenu = OptionMenu(paramFrame, pcktBurstName, *allPcktBurst)
        pcktBurstFocus = item

        # variable from the menuWidget dictionary is eudsName
        variable.set(item)

        # we removed the old version of the menu widget, 
        # add the new one to the grid and save it in the dictionary we
        # maintain to look up widgets for removal
        pcktBurstMenu.grid(row=3, column=7)
        menuWidget['PcktBurst'] = (pcktBurstMenu, variable)

        setOutputText(item+" successfully added to PcktBurst menu")
        return
        
    elif smn == 'InterPckt delay':
        # validate that packet inter-arrival mu is positive number
        item = item.strip()
        err = False
        if not item.replace('.','').isnumeric():
           err = True
        else:
            if 0.0 > float(item): 
                err = True
        
        if err:
            setOutputText("Packet inter-arrival must be non-negative")
            return

        # OK to proceded. Remove existing menu 
        (widget, variable) = menuWidget['PcktMu']
        widget.grid_remove()    

        # add new item 
        allPcktMu.append(item)

        # ensure sorted order by numeric value
        allPcktMu.sort(key=lambda v:float(v))

        # remember new menu
        paramLists['PcktMu'] = allPcktMu

        # create new menu widget
        pcktMuMenu = OptionMenu(paramFrame, pcktMuName, *allPcktMu)
        pcktMuFocus = item
        variable.set(item)

        # new widget in the grid 
        pcktMuMenu.grid(row=4, column=7)

        # remember widget and variable that holds the selected value 
        menuWidget['PcktMu'] = (pcktMuMenu, variable)
        setOutputText(item+" successfully added to PcktMu menu")
        return
      
    elif smn.find('Key Lengths') > -1 : 
        # validate that key length is an integer
        item = item.strip()
        err = False
        if not item.isdigit():
           err = True

        if err:
            setOutputText("Crypto key length must be integer")
            return

        # OK to proceded. Remove existing menu 
        # note that keys for menuWidget are compacted version of keys
        # for other menus
        (widget, variable) = menuWidget['KeyLengths']
        widget.grid_remove()    

        # add new item to menu
        allKeyLengths.append(item)

        # order by numeric value
        allKeyLengths.sort(key=lambda v:int(v))

        # remember modified menu
        paramLists['Key Lengths'] = allKeyLengths

        # create new widget
        keyLengthsMenu = OptionMenu(paramFrame, keyLengthsName, *allKeyLengths)
        keyLengthFocus = item
        variable.set(item)

        # place new widget
        keyLengthsMenu.grid(row=1, column=7)

        # remember widget and associated selected contents variable for later modification
        menuWidget['Key Lengths'] = (keyLengthsMenu, variable)

        setOutputText(item+" successfully added to Key Lengths menu")
        return
        
    elif smn.find('PvtNet') > -1:
        # validate that input is positive integer in [10, 10000] 
        item = item.strip()
        err = False
        if not item.isdigit():
           err = True
        else:
            if int(item) < 10 or int(item) > 10000:
                err = True
        
        if err:
            setOutputText("PvtNet Mbps must be in [10,10000]")
            return

        # OK to proceded. Remove existing menu and create new one
        (widget, variable) = menuWidget['PvtNet']
        widget.grid_remove()    
        allPvtNetBw.append(item)
        allPvtNetBw.sort(key=lambda v:int(v))
        paramLists['PvtNet Mbps'] = allPvtNetBw
        pvtNetBwMenu = OptionMenu(paramFrame, pvtNetBwName, *allPvtNetBw)
        pvtNetBwFocus = item
        variable.set(item)
        pvtNetBwMenu.grid(row=4, column=1)
        menuWidget['PvtNet'] = (pvtNetBwMenu, variable)
        setOutputText( item+" successfully added to PvtNet Mbps menu")
        return

    elif smn.find('PubNet') > -1:
        # validate that input is positive integer in [10, 10000] 
        item = item.strip()
        err = False
        if not item.isdigit():
           err = True
        else:
            if int(item) < 10 or int(item) > 10000:
                err = True
        
        if err:
            setOutputText("PubNet Mbps must be in [10,10000]")
            return

        # OK to proceded. Remove existing menu and create new one
        (widget, variable) = menuWidget['PubNet']
        widget.grid_remove()    
        allPubNetBw.append(item)
        allPubNetBw.sort(key=lambda v:int(v))
        paramLists['PubNet Mbps'] = allPubNetBw
        pubNetBwMenu = OptionMenu(paramFrame, pubNetBwName, *allPubNetBw)
        pubNetBwFocus = item
        variable.set(item)
        pubNetBwMenu.grid(row=4, column=4)
        menuWidget['PubNet'] = (pubNetBwMenu, variable)
        setOutputText( item+" successfully added to PubNet Mbps menu")
        return


# deleteItem removes a value the user asserts is in the selected menu, from the menu
def deleteItem():
    # find the menu being modified
    smn = selectMenuName.get()
    menu = paramLists[smn]

    item = selectText.get(1.0,END)

    # remove white space at endpoints
    item = item.strip()
    selectText.delete(1.0,END)

    # if item has the form '#int' then it is an index (starting at 0) of the 
    # menu item to be deleted
    if item.startswith('#') and item.replace('#','').isdigit():
        # check that the digit points to a menu item that exists
        pos = int(item.replace('#',''))
        if len(menu)-1 < pos:
            setOutputText('selected index of menu item is outside bounds of menu')
            return

        item = menu[pos]

    setOutputText('')

    # validate that string matches some entry in the named menu
    pList = paramLists[smn]
    found = False
    idx = 0
    for idx, entry in enumerate(pList):
        if entry.strip() == item:
            listEntry = entry
            found = True
            break

    if not found:
        setOutputText("entry {} not found in menu {}".format(item, smn))
        return

    # don't allow the user to completely empty a menu
    if len(pList) == 1: 
        setOutputText("not permitted to empty the menu with a deletion")
        return

    # remove the impacted menu widget
    (widget, variable, menu, rowv, colv) = menuNames[smn]
    widget.grid_remove()    

    pList = paramLists[smn]

    # remove the (now validated) entry from the menu
    pList.remove(entry)
    
    # the menu was sorted before, remains sorted by deletion

    # remember the modified menu
    paramLists[smn] = pList

    newMenu = OptionMenu(paramFrame, variable, *menu)

    # focus the modified menu on the first element
    variable.set(menu[0])

    # save location of modified menu for later removal
    menuWidget[smn] = (newMenu, variable)

    # place the modified menu on the grid
    newMenu.grid(row=rowv, column=colv)

    setOutputText( "successfully removed {} from {} menu".format(entry, smn))
    return


# save the state of the selected menu in a cache
def cacheMenu():
    smn = selectMenuName.get()
    menu = paramLists[smn]
    cachedMenu[smn] = copy.deepcopy(menu)
    setOutputText('cached state of menu {}'.format(smn))
    return

# recover the state of the selected menu from the cache, if present
def restoreCachedMenu():
    smn = selectMenuName.get()
    menu = paramLists[smn]
    if len(cachedMenu[smn]) == 0: 
        setOutputText('no cached state of menu {} is found'.format(smn))
        return

    cachedCopy = copy.deepcopy(cachedMenu[smn])

    (widget, variable, menu, rowv, colv) = menuNames[smn]
    widget.grid_remove()    

    # remember the modified menu
    paramLists[smn] = cachedCopy

    menu = cachedCopy
    newMenu = OptionMenu(paramFrame, variable, *menu)

    # focus the modified menu on the first element
    variable.set(menu[0])

    # save location of modified menu for later removal
    menuWidget[smn] = (newMenu, variable)

    # place the modified menu on the grid
    newMenu.grid(row=rowv, column=colv)

    setOutputText('successfully recovered cached state of menu {}'.format(smn))
    return

def restoreDefaultMenu():
    smn = selectMenuName.get()
    defaultmenu = copy.deepcopy(defaultMenu[smn])

    (widget, variable, menu, rowv, colv) = menuNames[smn]
    widget.grid_remove()    

    # remember the menu
    paramLists[smn] = defaultmenu

    newMenu = OptionMenu(paramFrame, variable, *defaultmenu)

    # focus the modified menu on the first element
    variable.set(defaultmenu[0])

    # save location of modified menu for later removal
    menuWidget[smn] = (newMenu, variable)

    # place the modified menu on the grid
    newMenu.grid(row=rowv, column=colv)

    setOutputText('successfully set default state of menu {}'.format(smn))
    return

# Functions called in response to actions on the GUI
#
# updateArchitecture called when the architecture is changed through its menu
# stopExp called when the button to exit the GUI is pushed
# executeExperiment called when the EXECUTE button is push to run an experiment
#

# updateArchitecture changes the current architecture image, and fusses with widgets
# that are architecturally dependent
def updateArchitecture(m,n,x):
    # m, n, and x are evidently required, but ignored
    archFocus = archName.get()

    # change to NoSSL removes widgets related to the crypto server, and changes architecture picture
    if archFocus == allArchs[1]:
        for loc, (widget, variable) in activeWidget.items():
            widget.grid_remove()  

        img.configure(image=renderNoCrypto)
        img.image=renderNoCrypto
       
    else:
        # change to SSL restores widgets related to the crypto serve
        for loc, (widget, variable) in activeWidget.items():
            widget.grid(row=loc[0], column=loc[1])

        img.configure(image=renderCrypto)
        img.image=renderCrypto

# stopExp is called when the button to exit the GUI is pushed
def stopExp():
    SaveMenus()
    exit(1)

# executeExperiment is called when the button to run the simulation experiment is pushed.
# create a file for an experiment manager script to run, call manger script,
# poll for its completion and report that a plot is available.
def executeExperiment():
    global counterFile, numExp, rendered, plotdir

    # bldParams gathers selections from the GUI menus, to be reported
    # to cntrl.py
    bldParams = {}
    bldParams['arch'] = archName.get()
    bldParams['euds'] = eudsName.get()
    bldParams['crypto'] = cryptoName.get()
    bldParams['srcCPU'] = cpuName.get()
    bldParams['sslCPU'] = sslCPUName.get()
    bldParams['eudCPU'] = eudCPUName.get()
    bldParams['pvtSwitch'] = pvtSwitchName.get()
    bldParams['pubSwitch'] = pubSwitchName.get()
    bldParams['rtr'] = rtrName.get()
    bldParams['pvtNetBw'] = pvtNetBwName.get()
    bldParams['pubNetBw'] = pubNetBwName.get()
    bldParams['pcktLen'] = pcktLenName.get()
    bldParams['pcktBurst'] = pcktBurstName.get()
    bldParams['pcktMu'] = pcktMuName.get()
    bldParams['keylength'] = keyLengthsName.get()
    bldParams['workingdir'] = workingdir
    bldParams['archdir'] = archdir
    bldParams['passthru'] = passthruFile
    
    # validate that the selected key length is associated with
    # selected crypto algorithm
    lengthFound = False
    for keyL in legalKeyLengths[bldParams['crypto']]:
        if str(keyL) == bldParams['keylength']:
            lengthFound = True
            break
    if not lengthFound:
        setOutputText('no timing measurements for crypto algorithm {} with key length {}'.format(
            bldParams['crypto'], bldParams['keylength']))
        return 

    # include the location of a file where the state of the experiment
    # is continuously updated.  Have not been succesful in getting the GUI
    # to display its continuous updates, but at least here we tell cntrl.py
    # where to put the updates
    counterFile = os.path.join(workingdir, 'expCounter.txt')
    bldParams['expCounter'] = counterFile

    # the plotFileName Text widget holds the extension-free version of the
    # name of the plot file cntrl.py is to create and place in
    pfile = plotFileName.get(1.0,END)
    pfile = pfile.strip()
    if len(pfile) == 0:
        setOutputText("need to specify plot file name")
        return
    else:
        setOutputText('')

    plotFileName.delete(1.0,END)

    base, ext = os.path.splitext(pfile)
    plotdir = os.path.join(workingdir,'plots')
    bldParams['plotFile'] = os.path.join(plotdir, base+'.png')

    # make sure that we don't give more than four variables for base or attrb lists
    baseParam = baseName.get()
    if baseParam == 'None' or baseParam == 'none':
        setOutputText('Base parameter set must be chosen')
        return

    attrbParam = attrbName.get()

    if len(paramLists[baseParam])*len(paramLists[attrbParam]) > 16:
        setOutputText('product of the lengths of the base and attrb menus cannot exceed 16')
        return 

    if baseParam == attrbParam and baseParam != 'None':
        setOutputText('cannot choose same parameter as base and attribute experiment selection')
        return

    # the user may choose only one of the base or attribute lists,
    # which means the selection for the other is 'None', which is carried along
    # and looked for
    if baseParam != 'None' : 
        bldParams['baseParamList'] = paramLists[baseParam]
    else:
        bldParams['baseParamList'] = [ baseParam ]

    if attrbParam != 'None' : 
        bldParams['attrbParamList'] = paramLists[attrbParam]
    else:
        bldParams['attrbParamList'] = [ attrbParam ]


    testAlgs = []
    testKeys = []

    if baseParam == 'CryptoAlg':
        testAlgs.extend(paramLists[baseParam])
    elif attrbParam == 'CryptoAlg':
        testAlgs.extend(paramLists[attrbParam])

    if baseParam == 'Key Lengths':
        testKeys.extend(paramLists[baseParam])
    elif attrbParam == 'Key Lengths':
        testKeys.extend(paramLists[attrParam])

    # test x-product of testAlgs and testKeys
    for alg in testAlgs:
        for keyLen in testKeys:
            lengthFound = False
            for keyL in legalKeyLengths[alg]:
                if str(keyL) == keyLen:
                    lengthFound = True
                    break
            if not lengthFound:
                setOutputText('no timing measurements for crypto algorithm {} with key length {}'.format(
                    alg, keyLen))
                return 

    # the keys in the bldParams dictionary are related to but not necessarily identical to
    # strings used in other dictionaries to refer to those entities.  rangeKeyConv
    # and invKeyConv provide the mapping, to be used with communication with the user

    rangeKeyConv = {'EUDs':'euds', 'Architecture':'arch','PcktMu':'pcktMu',
        'CryptoAlg':'crypto', 'Key Lengths':'keylength', 'PcktLen': 'pcktLen', 'PcktBurst': 'pcktBurst', 
        'PubNet Switch':'pubSwitch', 'PvtNet Switch':'pvtSwitch',
        'PubNet Mbps':'pubNetBw', 'PvtNet Mbps':'pvtNetBw','Router':'rtr', 'None':'None'}

    invKeyConv = {}
    for k,v in rangeKeyConv.items():
        invKeyConv[v] = k

    bldParams['attrbParam'] = rangeKeyConv[attrbParam]
    bldParams['baseParam'] = rangeKeyConv[baseParam]
    bldParams['invKeyConv'] = invKeyConv

    

    # notify user of our intentions, at least try to...
    numExp = len(bldParams['baseParamList'])*len(bldParams['attrbParamList'])
    setOutputText('running {} simulation experiments'.format(numExp))

    # convert bldParams to a yaml format and write to file in the same directory.
    # Here we assume that gui.py and cntrl.py live in the same directory
    with open('./exp.yaml', 'w') as wf:
        wf.write(yaml.dump(bldParams))

    # bring up cntrl.py and await its completion
    cmd = 'python ./cntrl.py exp.yaml'

    snooze = 1.0 

    proc = subprocess.Popen(
        cmd,
        shell=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        )

    lastReport = ''
    while True:
        time.sleep(snooze)

        # this seems to have no effect on outputText, but
        # we can print to the gui stdout
        with open(counterFile,'r') as rf:
            msg = rf.readline()
            msg = msg.strip()
            setOutputText(msg)
            if msg != lastReport:
                print(msg)
                lastReport = msg

        result = proc.poll()
        if result is not None:
            # result is non-None when the subprocess completes
            break

    outputFileName = bldParams['plotFile']
    outputFileName = outputFileName.strip()
    
    expPlot = Image.open(outputFileName)
    renderExpPlot = ImageTk.PhotoImage(expPlot)

    img.configure(image=renderExpPlot)
    img.image=renderExpPlot
    setOutputText('rendered experiment plot in {}'.format(bldParams['plotFile']+'{.png,.pdf}'))


read_menu = None
write_menu = None
devDescMap = None
allCrypto = None

args = None
workingdir = ''
archdir = ''
imagedir = ''
plotdir = ''
passthruFile = ''

# initialize the menus that cannot be modified, saved, and recovered.
# Loading them from the devDesc map
allCPUs = []
allSSLCPU = []
allEUDCPU = []
allpvtSwitch = []
allpubSwitch = []
allRtrs = []
allpvtNetBw = []
allpubNetBw = []
allPcktLen = []
allPcktBurst = []
allPcktMu = []
allKeyLengths = []
allEUDs = []
allCrypto = []

defaultCPUs = []
defaultSSLCPU = []
defaultEUDCPU = []
defaultpvtSwitch = []
defaultpubSwitch = []
defaultRtrs = []

# allArch is not modifiable but does need to be initialized
allArchs = [
    "SSL",
    "NoSSL"
]

# the menus below can be modified, saved, and recovered here at start-up
defaultEUDs = [
    "1",
    "10",
    "100",
    "1000"
]
defaultPcktLen = [
    "64",
    "128"
    "256",
    "512",
    "1024"
]
defaultPcktBurst = [
    "1",
    "10"
    "100",
]
defaultPcktMu = [
    "0",
    "0.001",
    "1e-3",
    "0.01",
    "1e-2",
    "0.1",
    "1e-1",
    "1",
    "1e+0",
]
defaultKeyLengths = [
    "256"
]
defaultPvtNetBw = [
    "1",
    "10",
    "100",
    "1000",
    "10000",
]
defaultPubNetBw = [
    "1",
    "10",
    "100",
    "1000",
    "10000",
]
# the menus below can be modified, saved, and recovered here at start-up
allEUDs = [
    "1",
    "10",
    "100",
    "1000"
]
allPcktLen = [
    "64",
    "128",
    "256",
    "512",
    "1024"
]
allPcktBurst = [
    "1",
    "10",
    "100",
]
allPcktMu = [
    "0",
    "0.001",
    "1e-3",
    "0.01",
    "1e-2",
    "0.1",
    "1e-1",
    "1",
    "1e+0",
]
allKeyLengths = [
    "256"
]
allPvtNetBw = [
    "1",
    "10",
    "100",
    "1000",
    "10000",
]
allPubNetBw = [
    "1",
    "10",
    "100",
    "1000",
    "10000",
]
allParams = [
    "Architecture",
    "EUDs",
    "CryptoAlg",
    "PcktLen",
    "PcktBurst",
    "PcktMu",
    "Src CPU",
    "SSL CPU",
    "EUD CPU",
    "PvtNet Switch",
    "PubNet Switch",
    "Router",
    "PvtNet Mbps",
    "PubNet Mbps"
]

addParamMenus = [
    'EUDs',
    'PcktLen',
    'PcktBurst',
    'InterPckt delay',
    'Key Lengths',
    'PvtNet Mbps',
    'PubNet Mbps',
]

modMenus = [
    'PcktSrc CPU',
    'Router',
    'PvtNet Switch',
    'PvtNet Mbps',
    'EUDs',
    'SSL CPU',
    'EUD CPU',
    'PubNet Switch',
    'PubNet Mbps',
    'Router',
    'Key Lengths',
    'CryptoAlg',
    'PcktLen',
    'PcktBurst',
    'InterPckt delay'
]
allMenus = ['Architecture']
allMenus.extend(copy.deepcopy(modMenus))

attrbParams = [
    "None",
    "Architecture",
    "EUDs",
    "CryptoAlg",
    "PcktLen",
    "PcktBurst",
    "PcktMu",
]
baseParams = [
    "Architecture",
    "EUDs",
    "CryptoAlg",
    "PcktLen",
    "PcktBurst",
    "PcktMu",
]

defaultMenu = {}
cachedMenu = {}
cachedMenu["EUDs"] = []
cachedMenu["CryptoAlg"] = []
cachedMenu["PcktSrc CPU"] = []
cachedMenu["SSL CPU"] = []
cachedMenu["EUD CPU"] = []
cachedMenu["PvtNet Switch"] = []
cachedMenu["PubNet Switch"] = []
cachedMenu["Router"] = []
cachedMenu["PvtNet Mbps"] = []
cachedMenu["PubNet Mbps"] = []
cachedMenu["PcktLen"] = []
cachedMenu["PcktBurst"] = []
cachedMenu["InterPckt delay"] = []
cachedMenu["Key Lengths"] = []


# on occasion we need to look up a particular menu as a function of some string name,
# which not incoincidently is the text label in the gui referring to that menu
paramLists = {}



counterFile = ''
lc = 0
numExp = 0

# script execution starts here.  Need to get the initialization done before building 
# the graphical stuff

buildParser()
processCmdArgs()
InitMenus()
 
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
root.title('MrNesbits beta demo')

# create the frames

imageFrame = LabelFrame(root, text="Architecture")
imageDir = os.path.join('./','images')
imageFile = os.path.join(imageDir, 'nocryptoSrvr.png')
loadNoCrypto = Image.open(imageFile)
renderNoCrypto = ImageTk.PhotoImage(loadNoCrypto)

imageFile = os.path.join(imageDir, 'cryptoSrvr.png')
loadCrypto = Image.open(imageFile)
renderCrypto = ImageTk.PhotoImage(loadCrypto)

img = Label(imageFrame, image=renderCrypto, width=500, height=500)
img.image = renderCrypto
img.pack(side="bottom", fill="both", expand="yes")
imageFrame.pack(fill=BOTH, expand=True)

paramFrame = LabelFrame(root, text="Experiment Params", bg='white', width=1200)
paramFrame.pack(fill=BOTH, expand=True)

modifyFrame = LabelFrame(root, text="Modify Menus", bg='white', width=1200)
modifyFrame.pack(fill=BOTH, expand=True)

expFrame = LabelFrame(root, text="Experiment",bg='white', width=1200)
expFrame.pack(fill=BOTH, expand=True)

outputFrame = LabelFrame(root, text="Output", bg='white', width=1000)
outputFrame.pack(fill=BOTH, expand=True)

controlFrame = LabelFrame(root, text="Control", bg='white', width=1200)
controlFrame.pack(fill=BOTH, expand=True)

selectMenuText = Entry(modifyFrame, bg='black', fg='white', justify=CENTER)
selectMenuText.insert(END,"Modify Menu")
selectMenuName = StringVar()
selectMenuMenu = OptionMenu(modifyFrame, selectMenuName, *modMenus)
selectMenuFocus = modMenus[0]
selectMenuName.set(selectMenuFocus)
selectMenuNameBackup = selectMenuName.get()
selectMenuText.grid(row=0,column=0)
selectMenuMenu.grid(row=0,column=1)

selectMenufiller = Entry(expFrame, bg='white', width=2, bd=0, highlightthickness=0) 
selectMenufiller.grid(row=0, column=2)

selectText = Text(modifyFrame, height=1, width=10)

addItemButton = Button(modifyFrame, text="AddItem", command=addItem)
deleteItemButton = Button(modifyFrame, text="DeleteItem", command=deleteItem)
cacheMenuButton = Button(modifyFrame, text="CacheMenu", command=cacheMenu)
restoreMenuButton = Button(modifyFrame, text="RestoreCachedMenu", command=restoreCachedMenu)
defaultMenuButton = Button(modifyFrame, text="RestoreDefaultMenu", command=restoreDefaultMenu)

selectText.grid(row=0, column=3)
addItemButton.grid(row=0, column=7)
deleteItemButton.grid(row=0, column=8)
cacheMenuButton.grid(row=0, column=9)
restoreMenuButton.grid(row=0, column=10)
defaultMenuButton.grid(row=0, column=11)


baseText = Entry(expFrame, bg='blue', fg='white', justify=CENTER)
baseText.insert(END,"Base Parameter")
baseName = StringVar()
#baseMenu = OptionMenu(expFrame, baseName, *baseParams)
baseMenu = OptionMenu(expFrame, baseName, *allMenus)
baseFocus = baseParams[0]
baseName.set(baseFocus)
baseNameBackup = baseName.get()
baseText.grid(row=0,column=0)
baseMenu.grid(row=0,column=1)
basefiller = Entry(expFrame, bg='white', width=10, bd=0, highlightthickness=0) 
basefiller.grid(row=0,column=2)

attrbText = Entry(expFrame, bg='blue', fg='white', justify=CENTER)
attrbText.insert(END,"Attribute Parameter")
attrbName = StringVar()
#attrbMenu = OptionMenu(expFrame, attrbName, *attrbParams)
attrbMenu = OptionMenu(expFrame, attrbName, *allMenus)
attrbFocus = attrbParams[0]
attrbName.set(attrbFocus)
attrbNameBackup = attrbName.get()
attrbText.grid(row=0,column=3)
attrbMenu.grid(row=0,column=4)
attrbfiller = Entry(expFrame, bg='white', width=10, bd=0, highlightthickness=0) 
attrbfiller.grid(row=0,column=5)

plotText = Entry(expFrame, bg='blue', fg='white', justify=CENTER)
plotText.insert(END,"Plot File Name")
plotText.grid(row=0,column=6)

plotFileName = Text(expFrame, height=1, width=12)
plotFileName.delete(1.0,END)
plotFileName.grid(row=0, column=7)

plotfiller = Entry(expFrame, bg='white', width=10, bd=0, highlightthickness=0) 
plotfiller.grid(row=0,column=8)

#labels for menus in paramFrame
filler = Entry(paramFrame, bg='white', width=2, bd=0, highlightthickness=0) 
archText = Entry(paramFrame, bg='yellow', fg='black', justify=CENTER)
archText.insert(END,"Architecture")
archName = StringVar()
archMenu = OptionMenu(paramFrame, archName, *allArchs)
archFocus = allArchs[0]
archName.set(archFocus)
archNameBack = archName.get()
archText.grid(row=0, column=0)
archMenu.grid(row=0, column=1)
filler.grid(row=0, column=2)
archName.trace("w", updateArchitecture)

eudsText = Entry(paramFrame, bg='yellow', fg='black', justify=CENTER)
eudsText.insert(END,"EUDs")
eudsName = StringVar()

eudsMenu = OptionMenu(paramFrame, eudsName, *allEUDs)
eudsFocus = allEUDs[1]
eudsName.set(eudsFocus)
eudsNameBackup = eudsName.get()
eudsText.grid(row=0, column=3)
eudsMenu.grid(row=0, column=4)
eudsFiller = Entry(paramFrame, bg='white', width=2, bd=0, highlightthickness=0)
eudsFiller.grid(row=0, column=5)

cryptoText = Entry(paramFrame, bg='olivedrab', fg='white', justify=CENTER)
cryptoText.insert(END, "CryptoAlg")
cryptoName = StringVar()
cryptoFocus = allCrypto[0]
cryptoName.set(cryptoFocus)
cryptoMenu = OptionMenu(paramFrame, cryptoName, *allCrypto)
cryptoNameBackup = cryptoName.get()
cryptoText.grid(row=0, column=6)
cryptoMenu.grid(row=0, column=7)

cpuText = Entry(paramFrame, bg='yellow', fg='black', justify=CENTER)
cpuText.insert(END, "PcktSrc CPU")
cpuName = StringVar()
cpuMenu = OptionMenu(paramFrame, cpuName, *allCPUs)
cpuFocus = allCPUs[2]
cpuName.set(cpuFocus)
cpuNameBackup = cpuName.get()
cpuText.grid(row=1, column=0)
cpuMenu.grid(row=1, column=1)
cpufiller = Entry(paramFrame, bg='white', width=2, bd=0, highlightthickness=0) 
cpufiller.grid(row=1, column=2)

sslCPUText = Entry(paramFrame, bg='yellow', fg='black', justify=CENTER)
sslCPUText.insert(END,"SSL CPU")
sslCPUName = StringVar()
sslCPUMenu = OptionMenu(paramFrame, sslCPUName, *allSSLCPU)
sslCPUFocus = allSSLCPU[0]
sslCPUName.set(sslCPUFocus)
sslCPUNameBackup = sslCPUName.get()
sslCPUText.grid(row=1, column=3)
sslCPUMenu.grid(row=1, column=4)
sslCPUfiller = Entry(paramFrame, bg='white', width=2, bd=0, highlightthickness=0) 
sslCPUfiller.grid(row=1, column=5)

eudCPUText = Entry(paramFrame, bg='yellow', fg='black', justify=CENTER)
eudCPUText.insert(END, "EUD CPU")
eudCPUName = StringVar()
eudCPUMenu = OptionMenu(paramFrame, eudCPUName, *allEUDCPU)
eudCPUFocus = allEUDCPU[0]
eudCPUName.set(eudCPUFocus)
eudCPUNameBackup = eudCPUName.get()
eudCPUText.grid(row=2, column=3)
eudCPUMenu.grid(row=2, column=4)

pvtSwitchText = Entry(paramFrame, bg='yellow', fg='black', justify=CENTER)
pvtSwitchText.insert(END, "PvtNet Switch")
pvtSwitchName = StringVar()
pvtSwitchMenu = OptionMenu(paramFrame, pvtSwitchName, *allpvtSwitch)
pvtSwitchFocus = allpvtSwitch[1]
pvtSwitchName.set(pvtSwitchFocus)
pvtSwitchNameBackup = pvtSwitchName.get()
pvtSwitchText.grid(row=3, column=0)
pvtSwitchMenu.grid(row=3, column=1)
pvtsfiller = Entry(paramFrame, bg='white', width=2, bd=0, highlightthickness=0) 
pvtsfiller.grid(row=3, column=2)

pubSwitchText = Entry(paramFrame, bg='yellow', fg='black', justify=CENTER)
pubSwitchText.insert(END, "PubNet Switch")
pubSwitchName = StringVar()
pubSwitchMenu = OptionMenu(paramFrame, pubSwitchName, *allpubSwitch)
 
pubSwitchName.set(allpubSwitch[0])
pubSwitchNameBackup = pubSwitchName.get()
pubSwitchText.grid(row=3, column=3)
pubSwitchMenu.grid(row=3, column=4)
pubsfiller = Entry(paramFrame, bg='white', width=2, bd=0, highlightthickness=0) 
pubsfiller.grid(row=3, column=5)

rtrText = Entry(paramFrame, bg='yellow', fg='black', justify=CENTER)
rtrText.insert(END, "Router")
rtrName = StringVar()
rtrMenu = OptionMenu(paramFrame, rtrName, *allRtrs)
rtrFocus = allRtrs[1]
rtrName.set(rtrFocus)
rtrNameBackup = rtrName.get()
rtrText.grid(row=2, column=0)
rtrMenu.grid(row=2, column=1)

pvtNetBwText = Entry(paramFrame, width=20, bg='grey', fg='white', justify=CENTER)
pvtNetBwText.insert(END,"PvtNet Mbps")
pvtNetBwName = StringVar()
pvtNetBwMenu = OptionMenu(paramFrame, pvtNetBwName, *allPvtNetBw)
pvtNetBwFocus = allPvtNetBw[1]
pvtNetBwName.set(pvtNetBwFocus)
pvtNetBwNameBackup = pvtNetBwName.get()
pvtNetBwText.grid(row=4, column=0)
pvtNetBwMenu.grid(row=4, column=1)
pubsfiller = Entry(paramFrame, bg='white', width=2, bd=0, highlightthickness=0) 
pubsfiller.grid(row=4, column=3)

pubNetBwText = Entry(paramFrame, bg='grey', fg='white', justify=CENTER)
pubNetBwText.insert(END,"PubNet Mbps")
pubNetBwName = StringVar()
pubNetBwMenu = OptionMenu(paramFrame, pubNetBwName, *allPubNetBw)
pubNetBwFocus = allPubNetBw[1]
pubNetBwName.set(pubNetBwFocus)
pubNetBwNameBackup = pubNetBwName.get()
pubNetBwText.grid(row=4, column=3)
pubNetBwMenu.grid(row=4, column=4)

activeWidget = {}
activeWidget[(1,3)] = (sslCPUText, None)
activeWidget[(1,4)] = (sslCPUMenu, sslCPUName)

pcktLenText = Entry(paramFrame, bg='olivedrab', fg='white', justify=CENTER)
pcktLenText.insert(END,"PcktLen (bytes)")
pcktLenName = StringVar()
pcktLenMenu = OptionMenu(paramFrame, pcktLenName, *allPcktLen)
pcktLenFocus = allPcktLen[1]
pcktLenName.set(pcktLenFocus)
pcktLenNameBackup = pcktLenName.get()
pcktLenText.grid(row=2, column=6)
pcktLenMenu.grid(row=2, column=7)

pcktBurstText = Entry(paramFrame, bg='skyblue', fg='black', justify=CENTER)
pcktBurstText.insert(END,"PcktBurst")
pcktBurstName = StringVar()
pcktBurstMenu = OptionMenu(paramFrame, pcktBurstName, *allPcktBurst)
pcktBurstFocus = allPcktBurst[1]
pcktBurstName.set(pcktBurstFocus)
pcktBurstNameBackup = pcktBurstName.get()
pcktBurstText.grid(row=3, column=6)
pcktBurstMenu.grid(row=3, column=7)

pcktMuText = Entry(paramFrame, bg='skyblue', fg='black', justify=CENTER)
pcktMuText.insert(END,"InterPckt delay (ms)")
pcktMuName = StringVar()
pcktMuMenu = OptionMenu(paramFrame, pcktMuName, *allPcktMu)
pcktMuFocus = allPcktMu[1]
pcktMuName.set(pcktMuFocus)
pcktMuNameBackup = pcktMuName.get()
pcktMuText.grid(row=4, column=6)
pcktMuMenu.grid(row=4, column=7)
pcktMuMenufiller = Entry(expFrame, bg='white', width=2, bd=0, highlightthickness=0) 

keyLengthsText = Entry(paramFrame, bg='olivedrab', fg='white', justify=CENTER)
keyLengthsText.insert(END,"Key Lengths")
keyLengthsName = StringVar()
keyLengthsMenu = OptionMenu(paramFrame, keyLengthsName, *allKeyLengths)
keyLengthsFocus = allKeyLengths[0]
keyLengthsName.set(keyLengthsFocus)
keyLengthsNameBackup = keyLengthsName.get()
keyLengthsText.grid(row=1, column=6)
keyLengthsMenu.grid(row=1, column=7)

menuWidget = {}
menuWidget['EUDs']    = (eudsMenu, eudsName)
menuWidget['PcktLen'] = (pcktLenMenu, pcktLenName)
menuWidget['PcktBurst'] = (pcktBurstMenu, pcktBurstName)
menuWidget['PcktMu'] = (pcktMuMenu, pcktMuName)
menuWidget['PvtNet']  = (pvtNetBwMenu, pvtNetBwName)
menuWidget['PubNet']  = (pubNetBwMenu, pubNetBwName)
menuWidget['KeyLengths']  = (keyLengthsMenu, keyLengthsName)

menuNames = {}
menuNames["EUDs"] = (eudsMenu, eudsName, allEUDs,0,4)
menuNames["CryptoAlg"] = (cryptoMenu, cryptoName, allCrypto,0,7) 
menuNames["PcktSrc CPU"] =  (cpuMenu, cpuName, allCPUs,1,1)
menuNames["SSL CPU"] = (sslCPUMenu, sslCPUName, allSSLCPU,1,4)
menuNames["EUD CPU"] = (eudCPUMenu, eudCPUName, *allEUDCPU,2,4)
menuNames["PvtNet Switch"] = (pvtSwitchMenu, pvtSwitchName, allpvtSwitch,3,1)
menuNames["PubNet Switch"] = (pubSwitchMenu, pubSwitchName, allpubSwitch,3,4)
menuNames["Router"] = (rtrMenu, rtrName, allRtrs,2,1)
menuNames["PvtNet Mbps"] = (pvtNetBwMenu, pvtNetBwName, allPvtNetBw,4,1)
menuNames["PubNet Mbps"] = (pubNetBwMenu, pubNetBwName, allPubNetBw,4,4)
menuNames["PcktLen"] = (pcktLenMenu, pcktLenName, allPcktLen, 2,7)
menuNames["PcktBurst"] = (pcktBurstMenu, pcktBurstName, allPcktBurst, 3,7)
menuNames["InterPckt delay"] = (pcktMuMenu, pcktMuName, allPcktMu,4,7)
menuNames["Key Lengths"] = (keyLengthsMenu, keyLengthsName, allKeyLengths,1,7)









expButton = Button(expFrame, command=executeExperiment, padx=3, width=8, text="EXECUTE", fg='blue')
expButton.grid(row=0, column=9)

outputText = scrolledtext.ScrolledText(outputFrame, width=100, height=2, wrap=WORD,
                font=("Times New Roman",20))

setOutputText('')
outputText.grid(row=1,column=0)

stopButton = Button(controlFrame, command=stopExp, padx=3, width=12, text="EXIT GUI", fg='red')
stopButton.grid(row=0, column=0)

root.mainloop()
