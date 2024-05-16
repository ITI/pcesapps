### 
### Alpha Version Demo
The objective of the alpha version of MrNesbits is to demonstrate the simulation of a topology and application running on that topology with performance parameters that are, or can in a short time frame, be measured in a lab.  

The figure below illustrates the alpha version we’ve developed.  The model is of an application ‘src’ that generates a packet and sends it to an application ‘crypto’ where the packet is encrypted and sent on to application ‘process’.   That application decrypts the message, processes it, encrypts a result, and sends it back to the ‘crypto’ application where it is decrypted and sent back to application ‘src’. Each of these applications reside on dedicated computing devices.   Adjacent pairs of devices are linked by a switch and a router.

![](./alpha-demo.png)

For the purposes of a demonstration we organize definitions of performance parameters in terms of the devices.  For each of the three computing devices we may choose individually one of two CPU types.   For each of the network devices we may choose individually one of two device models.   For every device (computing or network) we may choose the link-speed of the interfaces it hosts.   Note there is an assumption that all the interfaces on a device have the same link-speed.    Interfaces that connect two devices need not have the same link-speed, the simulation uses the minimum of the two values to ascribe latency to traffic in either direction.
Finally, we allow a user to choose between two crypto algorithms.   The same algorithm is used by applications ‘crypto’ and ‘process’.  All told then, there are 15 configuration parameters to select.   The table below enumerates them, and gives the command line flags we use to convey them to the simulator.

| Parameter            | Command Line Flag | Values                          |
|:--------------------:|:-----------------:|---------------------------------|
| pcktSrc CPU          | -scpu             | x86, i7                         |
| pcktSrc line-speed   | -sls              | any positive float (Mbytes/sec) |
| pvtSwitch Model      | -swpvtm           | Slow, Fast                      |
| pvtSwitch line-speed | -swpvtls          | any positive float (Mbytes/sec) |
| pvtRtr Model         | -rtrpvtm          | Slow, Fast                      |
| pvtRtr line-speed    | -rtrpvtls         | any positive float (Mbytes/sec) |
| ssl CPU              | -sslcpu           | x86, i7                         |
| ssl line-speed       | -sslls            | any positive float (Mbytes/sec) |
| pubRtr Model         | -rtrpubm          | Slow, Fast                      |
| pubRtr line-speed    | -rtrpubls         | any positive float (Mbytes/sec) |
| pubSwitch Model      | -swpubm           | Slow, Fast                      |
| pubSwitch line-speed | -swpubls          | any positive float (Mbytes/sec) |
| eudDev               | -eudcpu           | x86, i7                         |
| crypto algorithm     | -cryptoalg        | aes, rsa-3072                   |

### Running an experiment

alpha version demo simulation runs are conducted from within a directory ‘alpha-apps’.

One can run the python script run.py with a list of command-line parameters as described above, or put those parameters in a file (e.g., ‘args-run’) and call
               % python run.py -is args-run
The input file is comprised of lines listing the flags and defining the parameter values.  For example,  a working version of ‘args-run’  is
```
-srccpu i7
-srcls 10000
-swpvtm Fast
-swpvtls 10000
-rtrpvtm Fast
-rtrpvtls 20000
-swpubm Fast
-swpubls 10000
-rtrpubm Fast
-rtrpubls 20000
-sslcpu i7
-sslls 20000
-eudcpu i7
-eudls 200000
-cryptoalg rsa-3072

```

The script will follow through on the actions listed below under ‘Workflow’,  and print a round-trip time from the start of packet generation to the processing of a returned result, with an output line such as
         encryptPerf initiating function src measures 0.000495 seconds
The run also produces a trace file trace.yaml that logs the entry and exit of the packet from every network and application object on the path.    Eventually that file will need to be accessed and parsed, but it is not necessary for the alpha version demo.

#### Workflow
run.py takes on the command line performance-influencing parameters in the 
alpha version demo architecture for BITs.   It goes through a sequence of steps leading to a simulation run that incorporates the parameters. These steps are
1. Compile (if necessary) and run a program bld.go that builds a model of the alpha demo architecture, putting a number of configuration files in the subdirectory ./input.   These files will be read when the simulation run is executed.
2. Create an input file for program mdfy.go, compile mdfy.go if necessary, and run it.  The execution will create a file placed in ./input that formats the new parameter descriptions, and which is read when the simulation run is executed.
3. Run a sed command to replace the specification of the crypto algorithm in the appropriate configuration file to that included on the command line parameters given to run.py .
4. Compile (if necessary) and run sim.go, which runs the simulation experiment.  

The simulation experiment is engineered to execute exactly one round-trip communication from packet source, to SSL server, to EUD, and back.   The round-trip delay is printed out at the simulation’s end.  Running

    % python run.py -is args-run

produces output such as
      encryptPerf initiating function src measures 0.000495 seconds

The execution also produces an output file ./trace.yaml that contains very detailed trace information.   Parsing it will eventually be called for, but for the purposes of the alpha demo recovery of the output printed to standard out should suffice.
