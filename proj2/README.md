
# A simple url shortener

#### Jaime Danguillecourt

## Running
first run the backend:
`go run api/main.go <port>`
<port> is the port the backend will be hosted on (default 8000)

then run the front end:
`cd webapp && go run main.go`

optional parameters for frontend:
    -backend    address of backend
    -port       port frontend is hosted on

example:
    `go run main.go -backend=http://localhost:8000 -port=8080`
        *values in example are also the default values

## Description
The url shortener consists of two http servers. One is the front end (webapp) one is the backend (api).
The front end is an http server that hosts the html files and makes requests via http to the backend. The backend holds the data and has several endpoints users can hit for all CRUD functionability. 

Making the backend an http server makes it easy for any front end, regardless of the language it is written in,`:w
to use the backend. As long as the front end server and make and recieve http requests, it is compatible with the backend. It also makes it easy to add security, as we can easily add additional parameters in our json payloads (Request Struct) such as an access token. 
