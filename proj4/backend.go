package main

import (
  "github.com/kataras/iris/v12"
  "flag"
  "sync"
  "fmt"
  "strconv"
  "time"
  "math/rand"
  "net/http"
  "encoding/json"
  "io/ioutil"
  "strings"
)

/*
TODO
make log include uncommitted entries
master will always have sequencial log of commits / precommits, no gaps or holes bcs lock req
to access log
master then has seperate thread checking if uncommited entry after last commited
will then try to commit them:
    ask everyone else to precommit
    others will add the precomit, (num, command, commit stautus)
    and reply precomit ready
    master will wait until quorum reached
        keep trying nodes that never responded
            needs to keep track which nodes responded
    once quorum will commit message
    master has seperate thread that will deal with making last commit changes
    master replies to client commit made
    master tells all nodes to commit
    nodes then commit, seperate thread (same as master) will deal with making commit changes
    if nodes missed a commit message but get a future one, commit all precommit before
    if nodes master changes all precomits must be erased
        while loop deleting all id's after last commit until len of map same as last commit
TODO
*/

var backends []string
var port int

// struct used when sending json data
type Response struct {
    Status int      // 0 on success else failure
    Data string
}

/*
struct containing data for our CRUD app
urls: the key is the name of shortened url
    the value is the url we wish to redirect to
lock: read write lock for thread safety
*/
type Data struct {
    urls map[string]string
    lock sync.RWMutex
}

var data = Data{}

// used to keep track of edits to our data
type Log struct {
    data map[int][]string
    ch chan bool
    lock sync.Mutex
}

var log = Log{}

type Raft struct {
    state int // 0 = follower, 1 = candidate, 2 = leader
    stateLock sync.Mutex
    term int // current term
    termLock sync.Mutex
    leader []string // [ current leader, leader's term ] 
    leaderLock sync.Mutex
    votes map[string]string // key: vote_ip, item: nil
    votesLock sync.Mutex
    candidate string // candidate we voted for this term
    candidateLock sync.Mutex
    lastHeartbeat int64 // last heartbeat recieved
    heartbeatTimeout int // how long we'll wait for a heartbeat
    heartbeatLock sync.Mutex
}

var raft = Raft{}

func getResponse(backend string, route string) Response {
    resp, err := http.Get(backend+route)
    if err != nil {
        return Response{Status: 1, Data: err.Error()}
    }

    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return Response{Status: 1, Data: err.Error()}
    }

    var response Response
    json.Unmarshal([]byte(body), &response)
    return response
}

/*
function for add endpoint (/add?shortUrl=<shortUrl>&redirect=<redirect>)
query param shortUrl: short url to add to map
query param redirect: redirect url to be associated w/ short url
return: json w/ success or fail message
*/
func add(ctx iris.Context) {
    shortUrl := ctx.URLParam("shortUrl")
    redirect := ctx.URLParam("redirect")
    var message string
    var status int
    if shortUrl == "" {
        message = "no short url provided"
        status = 1
    } else if redirect == "" {
        message = "no redirect url provided"
        status = 1
    } else {
        data.lock.RLock()
        if _, ok := data.urls[shortUrl]; ok {
            data.lock.RUnlock()
            message = "cannot add '" + shortUrl + "': already exists."
        } else {
        // add url
        data.lock.RUnlock()
        data.lock.Lock()
        data.urls[shortUrl] = redirect
        data.lock.Unlock()
        message = "succesfully added url. /" + shortUrl + " now redirects to " + redirect
        }
    }
    // send response
    response := Response{Status: status, Data: message}
    ctx.JSON(response)
}

/*
function for delete endpoint (/delete/{shortUrl})
return: json w/ success or fail message
*/
func del(ctx iris.Context) {
    var message string
    var status int
    shortUrl := ctx.Params().Get("shortUrl")
    // delete url
    data.lock.RLock()
    if _, ok := data.urls[shortUrl]; ok {
        data.lock.RUnlock()
        data.lock.Lock()
        delete(data.urls, shortUrl)
        data.lock.Unlock()
        message = "successfully deleted " + shortUrl
        status = 0
    } else {
        data.lock.RUnlock()
        // failed to delete, short url doesnt exists
        message = "failed to delete '" +shortUrl +"': not found."
        status = 1
    }
    // send response
    response := Response{Status: status, Data: message}
    ctx.JSON(response)
}

/*
update route (/update/{shortUrl}?shortUrl=<newShortUrl>&redirect=<newRedirect>)
updates key and value in url map based on query parameters
query param shortUrl: new key in url map
query param redirect: new redirect value in url map
return: json w/ success or fail message
*/
func update(ctx iris.Context) {
    var message string
    var status int
    shortUrl := ctx.Params().Get("shortUrl")

    // update short url
    data.lock.RLock()
    if _, ok := data.urls[shortUrl]; ok {
        data.lock.RUnlock()
        newShortUrl := ctx.URLParam("shortUrl")
        newRedirect := ctx.URLParam("redirect")

        // change of key requires deleting old and creating new entry
        data.lock.Lock()
        if newShortUrl != shortUrl {
            delete(data.urls, shortUrl)
            data.urls[newShortUrl] = newRedirect
        } else {
            data.urls[shortUrl] = newRedirect
        }
        data.lock.Unlock()

        // render success message to client
        message = "succesfully updated '"+shortUrl+"'. short url: "+newShortUrl
        message += " redirect url: " + newRedirect
        status = 0
    } else {
        data.lock.RUnlock()
        // failed to update, short url doesnt exists
        message = "failed to update '" +shortUrl +"': not found."
        status = 1
    }
    // send response
    response := Response{Status: status, Data: message}
    ctx.JSON(response)
}

/*
handler for /fetch endpoint
return: json with all current data
*/
func fetch(ctx iris.Context) {
    message := ""
    data.lock.RLock()
    for key, value := range data.urls {
        message += key + "=" + value + " "
    }
    data.lock.RUnlock()
    status := 0
    // send response
    response := Response{Status: status, Data: message}
    ctx.JSON(response)
}

/*
handler for /{shortUrl}
return: Response obj w/ error or json containing the redirect url for the requested shortUrl
*/
func get(ctx iris.Context) {
    var message string
    var status int
    shortUrl := ctx.Params().Get("shortUrl")
    data.lock.RLock()
    if redirect, ok := data.urls[shortUrl]; ok {
        message = redirect
        status = 0
    } else {
        // failed to update, short url doesnt exists
        message = shortUrl +" not found."
        status = 1
    }
    data.lock.RUnlock()
    // send response
    response := Response{Status: status, Data: message}
    ctx.JSON(response)
}

/*
endpoint used for testing if server is alive 
return: response obj w/ status 0 and no data
*/
func ping(ctx iris.Context) {
    response := Response{Status: 0, Data: ""}
    ctx.JSON(response)
}



/*
endpoint for adding commands to our log
TODO use another thread to commit things from log our data, keeps track of last consecutive elem
request more data if notices it has a gap (len of log differnt to last index received)
use channel to tell log thread we just added something
func logAppend(ctx iris.Context) {
    index := ctx.URLParam("index")
    command := ctx.URLParam("command")
    i, err := strconv.Atoi(index)
    if err != nil {
        // return fail
        response := Response{Status: 1, Data: "failed to add command, invalid index"}
        ctx.JSON(response)
        return
    }
    log.lock.Lock()
    log.data[i] = command
    log.lock.Unlock()
    select {
        case log.ch <- true:
            // write to channel if we can
            // alerting something new in log
        default:
            // else drop message
    }

    // return success
    response := Response{Status: 0, Data: ""}
    ctx.JSON(response)
}

/*
this func runs in its own thread
it commits commands from the log to the data
it waits for a write to the logChannel to start processing elemts in the log
it then will handle as many consecutive elemtns in the log
if it detects theres elements missing, it will request the master for missing log elems
func logHandler() {
    lastCommit := -1
    for {
        // wait for something to be added to log
        <-log.ch

        // while consecutive log entries exist commit them all
        log.lock.Lock()
        logLen := len(log.data)
        flag := true
        for flag == true {
            if val, ok := log.data[lastCommit+1]; ok {
                // TODO
                // commit command to data

                lastCommit += 1
            } else {
                flag = false
            }
        }
        log.lock.Unlock()
        // if missing consequetive log entries, make a request
        if logLen-1 > lastCommit {
            // TODO
            // request lastCommit+1 from master
        }
    }

}
*/

func raft_vote(ctx iris.Context) {
    voter := ctx.URLParam("voter")
    // if we're currently not a candidate, ignore the vote
    raft.stateLock.Lock()
    if raft.state != 1 {
        raft.stateLock.Unlock()
        response := Response{Status: 1, Data: "vote ignored"}
        ctx.JSON(response)
        return

    }
    raft.stateLock.Unlock()

    //fmt.Println("recieved vote from:", voter) // DEBUG

    raft.votesLock.Lock()
    // if voter in raft.votes, then he hasn't voted for us
    // we then remove him to signify he voted for us
    if _, ok := raft.votes[voter]; ok {
        delete(raft.votes, voter)
    }
    raft.votesLock.Unlock()
}

func raft_candidate(ctx iris.Context) {
    // if leader ignore candidate request
    raft.stateLock.Lock()
    if raft.state == 2 || raft.state == 1 { // TODO IDK
        raft.stateLock.Unlock()
        response := Response{Status: 1, Data: "candidate rejected"}
        ctx.JSON(response)
        return

    }
    raft.stateLock.Unlock()

    candidate := ctx.URLParam("candidate")
    term := ctx.URLParam("term")
    //fmt.Println("recieved candidate req from:", candidate) // DEBUG

    // vote for candidate again if same candidate
    // candidate might have never received our past votes
    raft.candidateLock.Lock()
    if raft.candidate == candidate {
        raft.candidateLock.Unlock()
        // send vote
        //fmt.Println("sent vote to:", candidate) // DEBUG
        portStr := strconv.Itoa(port)
        my_ip := "http://localhost" + ":" + portStr
        route := "/raft_vote" + "?voter=" + my_ip
        getResponse(candidate, route) // send vote, disregard response

        // reset election timout
        resetHeartbeat()

        response := Response{Status: 0, Data: ""}
        ctx.JSON(response)
        return
    }
    raft.candidateLock.Unlock()

    vote_term, err := strconv.Atoi(term)
    if err != nil {
        response := Response{Status: 1, Data: "invalid term"}
        ctx.JSON(response)
        return
    }

    // vote for candidate if higher term
    raft.termLock.Lock()
    if vote_term > raft.term {
        // update term signifing we voted
        raft.term = vote_term
        raft.termLock.Unlock()

        // send vote to candidate
        my_ip := "http://localhost" + ":" + strconv.Itoa(port)
        route := "/raft_vote" + "?vote=" + my_ip
        getResponse(candidate, route) // send vote

        // update current candidate incase vote never gets received
        raft.candidateLock.Lock()
        raft.candidate = candidate
        raft.candidateLock.Unlock()

        // reset election timer after voting
        resetHeartbeat()

    } else {
        raft.termLock.Unlock()
    }
    response := Response{Status: 0, Data: ""}
    ctx.JSON(response)
}

/*
sets last heart beat to time now in milliseconds
the new timeout will be between max and min, hardcoded currently
(500 - 750 millisecond timeout) before follower becomes candidate
*/
func resetHeartbeat() {
    max := 1000
    min := 750
    raft.heartbeatLock.Lock()
    raft.lastHeartbeat = int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)
    raft.heartbeatTimeout = rand.Intn(max-min) + min // randon int from range min to max
    raft.heartbeatLock.Unlock()
}

/*
endpoint for receieving heartbeats
heartbeats will be used to determine who is the leader
*/
func raft_heartbeat(ctx iris.Context) {
    new_leader := ctx.URLParam("leader")
    new_leader_term_str := ctx.URLParam("term")
    //fmt.Println("recieved heartbeat from:", leader, "term:", term) // DEBUG

    new_leader_term, err := strconv.Atoi(new_leader_term_str)
    if err != nil {
        response := Response{Status: 1, Data: "invalid term"}
        ctx.JSON(response)
        return
    }

    // check if leader with higher term sending heartbeat
    raft.leaderLock.Lock()
    old_leader := raft.leader[0] // used later more efficient to grab now
    old_leader_termStr := raft.leader[1]
    raft.leaderLock.Unlock()
    old_leader_term, _ := strconv.Atoi(old_leader_termStr)

    if new_leader_term > old_leader_term {

        // update term
        raft.termLock.Lock()
        raft.term = new_leader_term
        raft.termLock.Unlock()

        // update leader
        raft.leaderLock.Lock()
        raft.leader[0] = new_leader
        raft.leader[1] = new_leader_term_str
        raft.leaderLock.Unlock()
        fmt.Println("new leader:", new_leader, "term:", new_leader_term) // DEBUG

        // set last heartbeat to time now in unix milliseconds
        resetHeartbeat()
        // make sure we're follower
        raft.stateLock.Lock()
        raft.state = 0
        raft.stateLock.Unlock()

    } else if old_leader == new_leader {
    // recieved heartbeat from current leader
        // make sure our term to matches leaders
        raft.termLock.Lock()
        raft.term = new_leader_term
        raft.termLock.Unlock()
        fmt.Println("current leader:", new_leader, "term:", new_leader_term) // DEBUG

        // set last heartbeat to unix time now in milliseconds
        resetHeartbeat()
    }

    response := Response{Status: 0, Data: ""}
    ctx.JSON(response)

}

func raftFollower() int {
    state := 0

    // while follower
    for state == 0 {
        // check if haven't recieved heartbeat within timeout
        timenow := int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)
        raft.heartbeatLock.Lock()
        if (timenow - raft.lastHeartbeat > int64(raft.heartbeatTimeout)) {
            raft.heartbeatLock.Unlock()
            // become candidate
            raft.stateLock.Lock()
            raft.state = 1
            raft.stateLock.Unlock()
            return 1
        }
        raft.heartbeatLock.Unlock()

        // short sleep so we don't hoard the state lock
        time.Sleep(50 * time.Millisecond)

        // check if we're still follower
        raft.stateLock.Lock()
        state = raft.state
        raft.stateLock.Unlock()
    }

    return state
}

func raftCandidateSetup() (int, int, int64) {
    // just became candidate thus
    // increment term
    raft.termLock.Lock()
    raft.term += 1
    term := raft.term // save our current term as candidate
    raft.termLock.Unlock()

    // reset votes
    raft.votesLock.Lock()
    for _, raft_node := range backends {
        raft.votes[raft_node] = ""
    }
    raft.votesLock.Unlock()

    // set candidate timeout
    timeout := rand.Intn(750-500) + 500 // randon int from range 750 to 500
    // set timestamp of we became candidate
    candidateTimestamp := int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)
    return term, timeout, candidateTimestamp

}

func raftCandidate() int {
    term, timeout, candidateTimestamp := raftCandidateSetup()

    state := 1

    // while candidate
    for state == 1 {
        // if timeout become follower
        timenow := int64(time.Nanosecond) * time.Now().UnixNano() / int64(time.Millisecond)
        if (timenow - candidateTimestamp > int64(timeout) ) {
            // reset timer
            resetHeartbeat()
            // change state
            raft.stateLock.Lock()
            raft.state = 0
            raft.stateLock.Unlock()
            return 0
        }

        // send candidate reqs to all nodes
        for _, raft_node := range backends {
            // TODO my ip
            my_ip := "http://localhost" + ":" + strconv.Itoa(port)
            route := "/raft_candidate?candidate=" + my_ip + "&term=" + strconv.Itoa(term)
            getResponse(raft_node, route) // send candidate req, disregard response
        }

        // check if recieved quorum of votes
        raft.votesLock.Lock()
        votes := len(backends) + 1 - len(raft.votes)
        raft.votesLock.Unlock()

        // if quorum become leader
        if votes > (len(backends) + 1)/2 {
            raft.stateLock.Lock()
            raft.state = 2
            raft.stateLock.Unlock()
            raft.leaderLock.Lock()
            raft.leader[0] = ""
            raft.leader[1] = "-1"
            raft.leaderLock.Unlock()
        }

        // check if we're still candidate
        raft.stateLock.Lock()
        state = raft.state
        raft.stateLock.Unlock()
    }
    return state
}

func raftLeader() int {
    state := 2

    // start heartbeat timer
    heartbeatTimer := time.NewTimer(50 * time.Millisecond)

    for state == 2 {
        select {
            case <-heartbeatTimer.C:
                // send heartbeat when timer finishes
                raft.termLock.Lock()
                term := raft.term
                raft.termLock.Unlock()
                for _, raft_node := range backends {
                    // TODO my ip
                    portStr := strconv.Itoa(port)
                    my_ip := "http://localhost" + ":" + portStr
                    route := "/raft_heartbeat" + "?leader=" + my_ip + "&term=" + strconv.Itoa(term)
                    getResponse(raft_node, route) // send heart beat, disregard response
                }

                // reset timer
                heartbeatTimer = time.NewTimer(50 * time.Millisecond)
            default:
                // short sleep better than burning cpu cycles
                time.Sleep(10 * time.Millisecond)
        }

        // check if we're still leader
        raft.stateLock.Lock()
        state = raft.state
        raft.stateLock.Unlock()
    }
    return state
}

func raftNode() {
    // get inital state
    // should always start as follower
    raft.stateLock.Lock()
    state := raft.state
    raft.stateLock.Unlock()

    // set up for timeout
    resetHeartbeat()

    for {
        switch state {
            case 0: // follower
                fmt.Println("I AM FOLLOWER, leader:", raft.leader[0]) // DEBUG

                state = raftFollower()

            case 1: // candidate
                fmt.Println("I AM CANDIDATE") // DEBUG

                state = raftCandidate()

            case 2: // leader
                fmt.Println("I AM LEADER, term :", raft.term) // DEBUG

                state = raftLeader()

        }
    }
}

func main() {
    //hardcode some initial data
    data.urls = make(map[string]string)
    data.urls["tandon"] = "https://engineering.nyu.edu/"
    data.urls["classes"] = "https://classes.nyu.edu/"

    log.data = make(map[int][]string)
    log.ch = make(chan bool, 1)

    raft.state = 0
    raft.term = 0
    raft.votes = make(map[string]string)
    raft.leader = []string{"", "-1"}

    app := iris.New()

    // add all our routes
    app.Get("/fetch", fetch)
    app.Get("/add", add)
    app.Get("/update/{shortUrl}", update)
    app.Get("/delete/{shortUrl}", del)
    app.Get("/ping", ping)
    app.Get("/raft_candidate", raft_candidate)
    app.Get("/raft_vote", raft_vote)
    app.Get("/raft_heartbeat", raft_heartbeat)
    app.Get("/{shortUrl}", get)

    // parse args
    portStr := flag.String("listen", "8000", "backend listening port")
    backendStr := flag.String("backends", "", "address of backends (comma seperated)")
    flag.Parse()

    var err error
    port, err = strconv.Atoi(*portStr)
    if err != nil {
        fmt.Println("invalid port provided:", portStr)
        return
    }
    // set global var

    backends = strings.Split(*backendStr, ",")
    // add localhost if missing hostname
    for i, backend := range backends {
        if backend[0] == ':' {
            backends[i] = "http://localhost" + backend
        }
    }
    // start raft
    go raftNode()

    // iris config
    config := iris.WithConfiguration(iris.Configuration {
        DisableStartupLog: true,
    })
    // start backend
    fmt.Println("BACKEND listening on " + *portStr)
    app.Listen(":"+*portStr, config)
}
