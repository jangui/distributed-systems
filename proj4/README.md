
# A simple url shortener

#### Jaime Danguillecourt

## Running
The easiest way to run is by running `make run` and `make stop` to stop

Otherwise you can do `make` to compile everything. Then run the binaries individually.

first run the backend:
`./api -port=8000`
or
`make runApi`

flags:
    port
        port backend listens on
        optional, defaults to 8000

then run the front end:
`./frontend -port=8080 -apiAddr=localhost -apiPort=8000 -apiProtocol=http`
or
`make runFrontend`

flags (all optional with defaults currently shown):
    port
        port frontend listens on
    apiAddr
        address of backend
    apiPort
        port of backend
    apiProtocol
        protocal of backend (http or https)

## Description
The url shortener consists of two http servers. One is the front end one is the backend.
The front end is an http server that hosts the html files and makes requests via http to the backend. The backend holds the data and has several endpoints users can hit for all CRUD functionability. 

Because most modern languages have libraries for using http, I chose to make the backend an http server. This should make it really easy for backends in any language on any device talk to the backend.

Because both the frontend and backend are built using iris they can handle concurrent HTTP request. However, a read write lock is used by the backend to make sure not race conditions occur when read and writting the data.

A more granular approach could have been taken by giving each data entry a unique, unchangeable id as their key in the map. This allows for one read write lock for the entire map, used for reading and add / removing entries. Additionally, each entry would have its own read write lock for reading that entry and the write lock would be used when updating it. 

However, although this approach would be better, a read write lock for the whole map was used instead for simplicity.

## Testing
To test run `make run`. 

The backend can be stoped with `make stopApi`
The backend can be restarted with `make runApi`

To run a vegeta test: `make vegeta`

This will run a test of all the sites functionality except redirecting using shortened urls. This functionally was not part of the test as the actual redirecting adds extra time to the latency that doesn't reflect the preformance of the actual site. 

To test the site with the redirect functionallity an additional target file (target2.list) is provided. The test can be run with the following command:
`vegeta attack -workers 50 -duration=30s -targets=target2.list | tee results.bin | vegeta report`

To stop `make stop` & then `make clean`
