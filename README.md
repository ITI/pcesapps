[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)



#### PCES/MRNES Application Examples

The repository *github.com/iti/pcesapps* provides examples of **pces** applications for reference in model building.  The examples themselves are less about the applications themselvess than they are about illustrating various features, attributes available to a modeler. The examples are closely tied to the discussion in [User-Extensions](#https://github.com/ITI/pcesapps/blob/main/docs/User-Extensions.pdf) of ways a modeler can integrate their own code into **pces** models.   These examples are

- "Embedded",  of an HMI computer that offloads computational tasks to an embedded system.
- "Queueing" modifies the process for initiating measurements in the same system considered by "Embedded", showing in particular how a set of simulation runs can be set up to estimate end-to-end latency as a function of the offered load of requests to the system.
- "End User Devices (EUDs)" illustrates how a user can integrate custom code to direct messages to randomly selected destinations, each of which has a different processing overhead, and measure the average end-to-end latency depending on the end user device chosen.
- "SJF" constructs a completely different system from the first three, and is constructed to show how other ways that user-written pieces of the system model can be integrated into a **pces** model.  As with the EUDs application,  this capability is highlighted in anticipation of users needing to keep models and code separate from public repositories such as github.
- "Flows" introduces the concept of traffic flows, as a vehicle to represent heavy traffic loads efficiently.

#### The PCES/MRNES System

The Patterned Computation Evaluation System (PCES) and Multi-resolution Network Emulator and Simulator (MRNES) are software frameworks one may use to model computations running on distributed system with the focus on estimating its performance and use of system resources.

The PCES/MRNES System is written in the Go language.  We have written a number of GitHub repositories that support this system, described below.

- https://github.com/iti/evt/vrtime .  Defines data structures and methods used to describe virtual time.
- https://github.com/iti/rngstream .  Implements a random number generator.
- https://github.com/iti/evt/evtq . Implements the priority queue used for event management.
- https://github.com/iti/evt/evtm . Implements the event manager.
- https://github.com/iti/evt/mrnes . Library of data structures and methods for describing a computer network.
- https://github.com/iti/evt/pces .  Library of data structures and methods for modeling patterned computations and running discrete-event simulations of those models.
- https://github.com/iti/evt/pcesbld . Repository of tool **xlsxPCES** used for describing PCES/MRNES models and generating input files used to run simulation experiments.
- https://github.com/iti/evt/pcesapps . Repository of PCES/MRNES example models, and scripts to generate and run experiments on those models.

Copyright 2025 Board of Trustees of the University of Illinois.
See [the license](LICENSE) for details.
