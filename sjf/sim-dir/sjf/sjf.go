package sjf

import (
	"github.com/iti/pces"
	"github.com/iti/mrnes"
	"github.com/iti/evt/evtm"
	"github.com/iti/evt/vrtime"
)

//---------- user extension code to provide a shortest-job-first-preemptive-resume
// service, and to have the completion of one execution thread initiate another

// SJF queue manages a list of tasks in service or waiting for service, 
// rspWrapper holds these.  Residual is remaining service time required
type rspWrapper struct {
    Residual float64
    Msg *pces.CmpPtnMsg
}

// rspOrder is the SJF queue.  In addition remembers the last modification time,
// and the eventID for completion of the job in service (needed for deleting an event)
type rspOrder struct {
    Modified float64
    EventID int 
    Queue []rspWrapper
}


// updateResidual is called when there is a change in state of the SJF queue,
// to subtract off the time elapsed since last modification from the job receiving service
func (rsp *rspOrder) updateResidual(now float64) {
    elapsed := now - rsp.Modified
    rsp.Modified = now 
    if len(rsp.Queue) > 0 { 
        rsp.Queue[0].Residual -= elapsed
    }   
}

// a constructor
func createRspOrder() *rspOrder {
    rsp := new(rspOrder)
    rsp.Queue = make([]rspWrapper,0)
    return rsp 
}

// SrvRspSJF is called when a new task arrives for service
func SrvRspSJF(evtMgr *evtm.EventManager, cpfi *pces.CmpPtnFuncInst, methodCode string, msg *pces.CmpPtnMsg) {

    // the next 8 lines are pretty much copies of the default srvRsp method in pces
    srvrc := cpfi.Cfg.(*pces.SrvRspCfg)
    srvrs := cpfi.State.(*pces.SrvRspState)
    srvrs.Calls += 1
    endptName := cpfi.Host
    endpt := mrnes.EndptDevByName[endptName]
    pces.AddCPTrace(pces.TraceMgr, cpfi.Trace, evtMgr.CurrentTime(), msg.ExecID,
        endpt.DevID(), pces.FullFuncName(cpfi, "srvRspSJF"), msg)


    // we create the bespoke state structures on demand, here, on the first
    // touch to this server
    if srvrs.Calls == 1 {
        rspOrd := createRspOrder()
        srvrs.Bespoke = rspOrd
    }

    // recover data structure governing SJF ordering
    rspOrd := srvrs.Bespoke.(*rspOrder)

    // look up the service requirement.  Use nil for message so that
    // lookup does not consider packet length 
    tcCode := srvrc.TimingCode[msg.MsgType]
    genTime := pces.HostFuncExecTime(cpfi, tcCode, nil)

    // create a wrapper around the newest task
    newTask := rspWrapper{Residual: genTime, Msg: msg}

    // if the SJF queue is empty put the arrival in service
    if len(rspOrd.Queue) == 0 {
        // add wrapped msg to queue
        rspOrd.Queue = append(rspOrd.Queue, newTask)

        // remember time of modification
        rspOrd.Modified = evtMgr.CurrentSeconds()

        // schedule service completion.  Remember the eventID for cancellation
        rspOrd.EventID,_ = evtMgr.Schedule(cpfi, nil, SJFComplete, vrtime.SecondsToTime(genTime))
        return
    }

    // queue is not empty.  Update residual times
    rspOrd.updateResidual(evtMgr.CurrentSeconds())

    // add the new task, determine whether a new event is needed
    // first see whether we need to lengthen the Queue

    for idx, wrapper := range rspOrd.Queue {

        // message at position idx has smaller residual, therefore higher priority
        if wrapper.Residual <= genTime {
            continue
        }

        // an existing job in the queue, rspOrd.Queue[idx], has a larger residual.
        // if that job is first we'll need to cancel the scheduled event, and schedule the new
        // shortest remaining job
        newQueue := make([]rspWrapper,0)
        if idx == 0 {
            evtMgr.RemoveEvent(rspOrd.EventID)
            newQueue = append(newQueue, newTask)
            rspOrd.Queue = append(newQueue, rspOrd.Queue...)
            rspOrd.EventID,_ = evtMgr.Schedule(cpfi, nil, SJFComplete, vrtime.SecondsToTime(genTime))
            return
        }
        // insert message at place other than end of list
        newQueue = append(newQueue, rspOrd.Queue[:idx]...)
        newQueue = append(newQueue, newTask)
        rspOrd.Queue = append(newQueue, rspOrd.Queue[idx:]...)
        return
    }

    // we get here if everything in the list has a smaller residual time, so 
    // all that is needed is to append to the end 
    rspOrd.Queue = append(rspOrd.Queue, newTask)
}
// SJFComplete handles the completion of a task in the SJF server
func SJFComplete(evtMgr *evtm.EventManager, context any, data any) any {
    cpfi := context.(*pces.CmpPtnFuncInst)
    srvrs := cpfi.State.(*pces.SrvRspState)
    rspOrd := srvrs.Bespoke.(*rspOrder)

    // get completed message
    msg := rspOrd.Queue[0].Msg

    endptName := cpfi.Host
    endpt := mrnes.EndptDevByName[endptName]
    pces.AddCPTrace(pces.TraceMgr, cpfi.Trace, evtMgr.CurrentTime(), msg.ExecID,
        endpt.DevID(), pces.FullFuncName(cpfi, "SJFComplete"), msg)

    // pop the rspOrd queue
    rspOrd.Queue = rspOrd.Queue[1:]
	rspOrd.Modified = evtMgr.CurrentSeconds()

    // if there is another task, put SJF task into service, schedule completion
    if len(rspOrd.Queue) > 0 {
        rspOrd.EventID,_ = evtMgr.Schedule(cpfi, msg, SJFComplete, vrtime.SecondsToTime(rspOrd.Queue[0].Residual))
    }

    // return response
    msg.CPID = msg.RtnCPID
    msg.Label = msg.RtnLabel
    msg.MsgType = msg.RtnMsgType

    // put where ExitFunc will find it
    cpfi.AddResponse(msg.ExecID, []*pces.CmpPtnMsg{msg})

    // schedule ExitFunc to finish up
    evtMgr.Schedule(cpfi, msg, pces.ExitFunc, vrtime.SecondsToTime(0.0))
    return nil
}

