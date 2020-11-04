package mapreduce

import "fmt"

type WorkerInfo struct {
	address string
}

// Clean up all workers by sending a Shutdown RPC to each one of them Collect
// the number of jobs each work has performed.
func (mr *MapReduce) KillWorkers() []int {
	l := make([]int,0)
	for _, w := range mr.Workers {
		DPrintf("DoWork: shutdown %s\n", w.address)
		args := &ShutdownArgs{}
		var reply ShutdownReply
		ok := call(w.address, "Worker.Shutdown", args, &reply)
		if ok == false || reply.OK == false {
			fmt.Printf("DoWork: RPC %s shutdown error\n", w.address)
		} else {
			l=append(l,reply.Njobs)
		}
	}
	return l
}

/*
function used by main thread for calling reducers
does RPC call and writes to done channel when done
also writes to worker channel when done to signal
worker is available
this function is inteded it be its own thread
*/
func (mr *MapReduce) DoMap(worker string, i int) {
    var reply DoJobReply
    args := &DoJobArgs{mr.file, Map, i, mr.nReduce}
    ok := call(worker, "Worker.DoJob", args, &reply)
    if ok {
        mr.DoneChannel <- true
        mr.registerChannel <- worker
    }
}

/*
function used by main thread for calling reducers
does RPC call and writes to done channel when done
also writes to worker channel when done to signal
worker is available
this function is inteded it be its own thread
*/
func (mr *MapReduce) DoReduce(worker string, i int) {
    var reply DoJobReply
    args := &DoJobArgs{mr.file, Reduce, i, mr.nMap}
    ok := call(worker, "Worker.DoJob", args, &reply)
    if ok {
        mr.DoneChannel <- true
        mr.registerChannel <- worker
    }

}


/*
main thread that creates map and reduce threads
this thread will use channels to make sure all map jobs
finish before starting reducer jobs
it also makes sure not to start more map jobs than need / have workers for
*/
func (mr *MapReduce) RunMaster() []int {
    var worker string
    var finishedJobs int
    var i int
    // run map jobs
    for {
        select {
            // check if all map jobs finished
            case <-mr.DoneChannel:
                finishedJobs++

            // worker can work
            case worker = <-mr.registerChannel:
                // if enough map jobs started, dont start more
                if i >= mr.nMap {
                    continue
                }
                // work
                go mr.DoMap(worker, i)
                i++
        }
        if finishedJobs < mr.nMap {
            break
        }
    }
    finishedJobs = 0
    i = 0
    // run reduce jobs
    for {
        select {
            // check if all jobs finished
            case <-mr.DoneChannel:
                finishedJobs++

            // worker can work
            case worker = <-mr.registerChannel:
                // if enough reduce jobs started, dont start more
                if i >= mr.nReduce {
                    continue
                }
                // work
                go mr.DoReduce(worker, i)
                i++
        }
        if finishedJobs < mr.nMap {
            break
        }
    }
    return mr.KillWorkers()
}
