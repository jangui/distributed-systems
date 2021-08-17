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


var backends []string
var port int
var my_addr string

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
type Urls struct {
    data map[string]string
    lock sync.RWMutex
}

var urls = Urls{}

// used to keep track of edits to our data
type Log struct {
    data map[int][]string
    lastCommit int
    nextCommit int
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

func getResponse(host string, route string) Response {
    resp, err := http.Get(host+route)
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
returns what curernt state we're in
0: follower
1: candidate
2: leader
*/
func getState() int {
    raft.stateLock.Lock()
    state := raft.state
    raft.stateLock.Unlock()
    return state
}

/*
function for add endpoint (/add?shortUrl=<shortUrl>&redirect=<redirect>)
query param shortUrl: short url to add to map
query param redirect: redirect url to be associated w/ short url
return: json w/ success or fail message
*/
func addEndpoint(ctx iris.Context) {
    shortUrl := ctx.URLParam("shortUrl")
    redirect := ctx.URLParam("redirect")

    // if not leader tell client they have wrong leader
    // client will then find new leader
    if getState() != 2 {
        response := Response{Status: 2, Data: "not leader"}
        ctx.JSON(response)
        return
    }

    // check to see if add is valid
    status, message := checkAdd(shortUrl, redirect)
    // if cant add tell client
    if status == 1 {
        response := Response{Status: status, Data: message}
        ctx.JSON(response)
        return
    }

    if logReplicate("add", []string{shortUrl, redirect}) {
        status = 0
        message = "succesfully added url. /" + shortUrl + " now redirects to " + redirect
    } else {
        status = 1
        message = "add rejected"
    }

    response := Response{Status: status, Data: message}
    ctx.JSON(response)
}

/*
checks if add is valid
return: status (int; 0 = success, 1 = error) message (string)
*/
func checkAdd(shortUrl string, redirect string) (int, string) {
    status := 0
    var message string

    if shortUrl == "" {
        message = "no short url provided"
        status = 1
    } else if redirect == "" {
        message = "no redirect url provided"
        status = 1
    } else {
        urls.lock.RLock()
        // cant add if already exists
        if _, ok := urls.data[shortUrl]; ok {
            urls.lock.RUnlock()
            message = "cannot add '" + shortUrl + "': already exists."
            status = 1
        }
        urls.lock.RUnlock()
    }

    return status, message
}

/*
do the actual add to our data
shortUrl: short url to add
redirect: where url redirects to
return: status (int) and message (string) to send back to client
*/
func add(shortUrl string, redirect string) {
    urls.lock.Lock()
    urls.data[shortUrl] = redirect
    urls.lock.Unlock()
}

/*
function for delete endpoint (/delete/{shortUrl})
return: json w/ success or fail message
*/
func delEndpoint(ctx iris.Context) {
    shortUrl := ctx.Params().Get("shortUrl")

    // if not leader tell client they have wrong leader
    // client will then find new leader
    if getState() != 2 {
        response := Response{Status: 2, Data: "not leader"}
        ctx.JSON(response)
        return
    }

    // check to see if delete is valid
    status, message := checkDel(shortUrl)
    // if cant delete tell client
    if status == 1 {
        response := Response{Status: status, Data: message}
        ctx.JSON(response)
        return
    }

    if logReplicate("del", []string{shortUrl}) {
        status = 0
        message = "successfully deleted"
    } else {
        status = 1
        message = "delete rejected"
    }

    // send response
    response := Response{Status: status, Data: message}
    ctx.JSON(response)
}

/*
check if delete valid
shortUrl: short url to delete
return: status (int; 0 = success, 1 = error) message (string)
*/
func checkDel(shortUrl string) (int, string) {
    var message string
    // check if short url exists
    urls.lock.RLock()
    if _, ok := urls.data[shortUrl]; !(ok) {
        urls.lock.RUnlock()
        // short url doesnt exists
        message := "failed to delete '" +shortUrl +"': not found."
        return 1, message
    }
    urls.lock.RUnlock()
    return 0, message
}

/*
do the deletion on the data
shortUrl: short url to delete
return: status (int; 0 = success, 1 = error) message (string)
*/
func del(shortUrl string) {
    // delete url
    urls.lock.Lock()
    delete(urls.data, shortUrl)
    urls.lock.Unlock()
}

/*
update route (/update/{shortUrl}?shortUrl=<newShortUrl>&redirect=<newRedirect>)
updates key and value in url map based on query parameters
query param shortUrl: new key in url map
query param redirect: new redirect value in url map
return: json w/ success or fail message
*/
func updateEndpoint(ctx iris.Context) {
    var status int
    var message string
    shortUrl := ctx.Params().Get("shortUrl") // shortUrl to update
    newShortUrl := ctx.URLParam("shortUrl")  // new name
    newRedirect := ctx.URLParam("redirect")  // new redirect

    // if not leader tell client they have wrong leader
    // client will then find new leader
    if getState() != 2 {
        response := Response{Status: 2, Data: "not leader"}
        ctx.JSON(response)
        return
    }

    // check to see if update is valid
    status, message = checkUpdate(shortUrl, newShortUrl, newRedirect)
    // if cant update tell client
    if status == 1 {
        response := Response{Status: status, Data: message}
        ctx.JSON(response)
        return
    }

    if logReplicate("update", []string{shortUrl, newShortUrl, newRedirect}) {
        status = 0
        message = "succesfully updated '"+shortUrl+"'. short url: "+newShortUrl
        message += " redirect url: " + newRedirect
    } else {
        status = 1
        message = "update rejected"
    }

    response := Response{Status: status, Data: message}
    ctx.JSON(response)

}

/*
check if update is valid
shortUrl: short url to update
newShortUrl: new short url name
newRedirect: new url to redirect to
return: status (int; 0 = success, 1 = error) message (string)
*/
func checkUpdate(shortUrl string, newShortUrl string, newRedirectUrl string) (int, string) {
    var message string

    // check if shortUrl exists to update
    urls.lock.RLock()
    if _, ok := urls.data[shortUrl]; !(ok) {
        urls.lock.RUnlock()
        // failed to update, short url doesnt exists
        message = "failed to update '" +shortUrl +"': not found."
        return 1, message
    }
    urls.lock.RUnlock()
    return 0, message
}

/*
do update
shortUrl: short url to update
newShortUrl: new short url name
newRedirect: new url to redirect to
return: status (int; 0 = success, 1 = error) message (string)
*/
func update(shortUrl string, newShortUrl string, newRedirect string) {
    urls.lock.Lock()
    if newShortUrl != shortUrl {
        // change of key requires deleting old and creating new entry
        delete(urls.data, shortUrl)
        urls.data[newShortUrl] = newRedirect
    } else {
        urls.data[shortUrl] = newRedirect
    }
    urls.lock.Unlock()
}

/*
handler for /fetch endpoint
return: json with all current data
*/
func fetchEndpoint(ctx iris.Context) {
    // if not leader tell client they have wrong leader
    // client will then find new leader
    if getState() != 2 {
        response := Response{Status: 2, Data: "not leader"}
        ctx.JSON(response)
        return
    }

    message := ""
    urls.lock.RLock()
    for key, value := range urls.data {
        message += key + "=" + value + " "
    }
    urls.lock.RUnlock()
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
    // if not leader tell client they have wrong leader
    // client will then find new leader
    if getState() != 2 {
        response := Response{Status: 2, Data: "not leader"}
        ctx.JSON(response)
        return
    }

    var message string
    var status int
    shortUrl := ctx.Params().Get("shortUrl")
    urls.lock.RLock()
    if redirect, ok := urls.data[shortUrl]; ok {
        message = redirect
        status = 0
    } else {
        // failed to update, short url doesnt exists
        message = shortUrl +" not found."
        status = 1
    }
    urls.lock.RUnlock()
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
endpoint for asking who is current leader
*/
func getLeader(ctx iris.Context) {
    // if we're leader response saying we are
     if getState() == 2 {
        response := Response{Status: 0, Data: my_addr}
        ctx.JSON(response)
        return
    }

    // put leader into response data
    raft.leaderLock.Lock()
    response := Response{Status: 0, Data: raft.leader[0]}
    raft.leaderLock.Unlock()

    // if no leader return error response code
    if response.Data == "" {
        response.Status = 1
    }

    ctx.JSON(response)
}

func logReplicate(command string, data []string) bool {

    // precommit
    log.lock.Lock()
    index := log.lastCommit + 1
    log.data[index] = []string{"false", command}
    for _, command_data := range data {
        log.data[index] = append(log.data[index], command_data)
    }

    // send precommit
    route := getRoute(command, data, index, 0)
    replies := 0
    for _, backend := range backends {
        response := getResponse(backend, route)
        if response.Status == 0 {
            replies += 1
        }
    }

    // if less than quorum reject change
    if replies < len(backends) / 2 {
        log.lock.Unlock()
        return false
    }

    // commit
    if log.nextCommit < index {
        log.nextCommit = index
    }
    log.lock.Unlock()
    // committing thread will handle actual modification of data
    // and handle sending commits to others
    // at this point the leader has committed and will tell the client success
    return true
}

/*
func for getting which route to hit when wanting to do log replication
command: command we wish to run
data: data needed for command
index: index in log to commit / precommit
flag: 0 if precommit, 1 if commit
return: string w/ route for appropriate command for log replication
*/
func getRoute(command string, data []string, index int, flag int) string {
    route := "/commit/"
    switch command {
        case "add":
            route += "add?shortUrl=" + data[0] + "&redirect=" + data[1] + "&index=" + strconv.Itoa(index)
        case "del":
            route += "del?shortUrl=" + data[0] + "&index=" + strconv.Itoa(index)
        case "update":
            route += "update?shortUrl=" + data[0] + "&newShortUrl=" + data[1]
            route += "&newRedirect=" + data[1] + "&index=" + strconv.Itoa(index)
    }
    if flag == 0 {
        route += "&flag=precommit"
    } else if flag == 1 {
        route += "&flag=commit"
    }
    return route
}

/*
end point for precommit / commit
/commit/{command}?urlParameters
command: what command to precommit
url parameters:
    data for command
    index to pick into log
    flag (precom or commit)
*/
func commitEndpoint(ctx iris.Context) {
    command := ctx.Params().Get("command")
    shortUrl := ctx.URLParam("shortUrl")
    indexStr := ctx.URLParam("index")
    flag := ctx.URLParam("flag")
    var data []string
    switch command {
        case "add":
            redirect := ctx.URLParam("redirect")
            data = []string{shortUrl, redirect}
        case "del":
            data = []string{shortUrl}
        case "update":
            newShortUrl := ctx.URLParam("newShortUrl")
            newRedirect := ctx.URLParam("newRedirect")
            data = []string{shortUrl, newShortUrl, newRedirect}
    }
    index, _ := strconv.Atoi(indexStr)

    // add to log
    log.lock.Lock()
    status := "false"
    if flag == "commit" {
        status = "true"
    }
    log.data[index] = []string{status, command}
    for _, command_data := range data {
        log.data[index] = append(log.data[index], command_data)
    }
    log.lock.Unlock()
    // commit thread will handle actual modification of data

    response := Response{Status: 0, Data: ""}
    ctx.JSON(response)
}

/*
function used for requesting a commit
*/
func reqCommit(ctx iris.Context) {
    status := 1
    var message string

    indexStr := ctx.URLParam("index")
    requester := ctx.URLParam("requester")
    index, _ := strconv.Atoi(indexStr)
    log.lock.Lock()
    // if we have the commit and it is commited, send it over
    if _, ok := log.data[index]; ok {
        entry := log.data[index]
        log.lock.Unlock()
        if entry[0] == "true" {
            // send over commit
            route := getRoute(entry[1], entry[2:len(entry)], index, 1)
            getResponse(requester, route)
        }
    } else {
        log.lock.Unlock()
        status = 1
        message = "missing commit"

    }
    response := Response{Status: status, Data: message}
    ctx.JSON(response)
}

func vote(ctx iris.Context) {
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

    raft.votesLock.Lock()
    // if voter in raft.votes, then he hasn't voted for us
    // we then remove him to signify he voted for us
    if _, ok := raft.votes[voter]; ok {
        delete(raft.votes, voter)
    }
    raft.votesLock.Unlock()
}

func candidateReq(ctx iris.Context) {
    // if leader ignore candidate request
    raft.stateLock.Lock()
    if raft.state == 2 || raft.state == 1 {
        raft.stateLock.Unlock()
        response := Response{Status: 1, Data: "candidate rejected"}
        ctx.JSON(response)
        return

    }
    raft.stateLock.Unlock()

    // if our last commit greater than candidates, ignore request
    candLastCommitStr := ctx.URLParam("last_commit")
    candLastCommit, err := strconv.Atoi(candLastCommitStr)
    if err != nil {
        response := Response{Status: 1, Data: "invalid last commit"}
        ctx.JSON(response)
        return
    }
    log.lock.Lock()
    last_commit := log.lastCommit
    log.lock.Unlock()
    if last_commit > candLastCommit {
        response := Response{Status: 1, Data: "candidate rejected"}
        ctx.JSON(response)
        return
    }

    candidate := ctx.URLParam("candidate")
    term := ctx.URLParam("term")

    // vote for candidate again if same candidate
    // candidate might have never received our past votes
    raft.candidateLock.Lock()
    if raft.candidate == candidate {
        raft.candidateLock.Unlock()
        // send vote
        route := "/vote" + "?voter=" + my_addr
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
        route := "/vote" + "?vote=" + my_addr
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
func raftHeartbeat(ctx iris.Context) {
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

        log.lock.Lock()
        last_commit := log.lastCommit
        log.lock.Unlock()
        // send candidate reqs to all nodes
        for _, raft_node := range backends {
            route := "/candidate_req?candidate=" + my_addr + "&term=" + strconv.Itoa(term)
            route += "&last_commit=" + strconv.Itoa(last_commit)
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
                    route := "/raft_heartbeat" + "?leader=" + my_addr + "&term=" + strconv.Itoa(term)
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

func commitHandler() {
    for {
        log.lock.Lock()
        // check if we commited last entry
        if log.nextCommit == log.lastCommit && log.lastCommit != -1 {
            entry := log.data[log.lastCommit]
            index := log.lastCommit
            log.lock.Unlock()
            // send over commit
            for _, backend := range backends {
                route := getRoute(entry[1], entry[2:len(entry)], index, 1)
                getResponse(backend, route)
            }
            time.Sleep(150 * time.Millisecond)
            continue
        }

        // commit next entry if we can
        if _, ok := log.data[log.lastCommit+1]; ok {
            if log.data[log.lastCommit+1][0] == "false" {
                log.data[log.lastCommit+1][0] = "true"
                doCommit(log.data[log.lastCommit+1])
                log.lastCommit += 1
                log.lock.Unlock()
                continue
            }
        }

        // check if missing entries
        lastCom := log.lastCommit
        if log.nextCommit - log.lastCommit > 1 {
            log.lock.Unlock()

            // ask nodes for lastCommit + 1
            for _, backend := range backends {
                route := "/requestCommit?index=" + strconv.Itoa(lastCom)
                route += "&requester=" + my_addr
                response := getResponse(backend, route)
                if response.Status == 0 {
                    // node will send us back commit to our commit endpoint
                    break
                }
            }

            log.lock.Lock()
        }

        log.lock.Unlock()
    }
}

func doCommit(data []string) {
    command := data[1]
    switch command {
        case "add":
            add(data[2], data[3])
        case "del":
            del(data[2])
        case "update":
            update(data[2], data[3], data[4])
    }
}

func main() {
    //hardcode some initial data
    urls.data = make(map[string]string)
    urls.data["tandon"] = "https://engineering.nyu.edu/"
    urls.data["classes"] = "https://classes.nyu.edu/"

    log.data = make(map[int][]string)

    raft.state = 0
    raft.term = 0
    raft.votes = make(map[string]string)
    raft.leader = []string{"", "-1"}

    log.lastCommit = -1
    log.nextCommit = -1

    app := iris.New()

    // add all our routes
    app.Get("/fetch", fetchEndpoint)
    app.Get("/add", addEndpoint)
    app.Get("/update/{shortUrl}", updateEndpoint)
    app.Get("/delete/{shortUrl}", delEndpoint)
    app.Get("/commit/{command}", commitEndpoint)
    app.Get("/ping", ping)
    app.Get("/candidate_req", candidateReq)
    app.Get("/vote", vote)
    app.Get("/raft_heartbeat", raftHeartbeat)
    app.Get("/get_leader", getLeader)
    app.Get("/{shortUrl}", get)


    // parse args
    portStr := flag.String("listen", "8000", "backend listening port")
    backendStr := flag.String("backends", "", "address of backends (comma seperated)")
    hostname := flag.String("hostname", "http://localhost", "address of computer this is running on")
    flag.Parse()
    my_addr = *hostname + ":" + *portStr

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

    // start commit handler
    go commitHandler()

    // iris config
    config := iris.WithConfiguration(iris.Configuration {
        DisableStartupLog: true,
    })
    // start backend
    fmt.Println("BACKEND listening on " + *portStr)
    app.Listen(":"+*portStr, config)
}
