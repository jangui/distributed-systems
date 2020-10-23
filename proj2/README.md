
# A simple url shortener

#### Jaime Danguillecourt

## Running
The easiest way to run is by running `make run` and `make stop` to stop

Otherwise you can do `make` to compile everything. Then run the binaries individually.

first run the backend:
`./api -port=8000`

flags:
    port
        port backend listens on
        optional, defaults to 8000

then run the front end:
`./frontend -port=8080 -apiAddr=localhost -apiPort=8000 -apiProtocol=http`

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

Because most modern languages have libraries for using http, I chose to make the backend an http server. This should make it really easy for backends in any language on any device talk to the backend. I also broke my wrist last week and wanted to do an implemetation that would save the most typing.

