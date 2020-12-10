package main

import (
    "github.com/kataras/iris/v12"
    "net/http"
    "encoding/json"
    "io/ioutil"
    "strings"
    "flag"
    "fmt"
    "time"
)

// response struct used to decode json from backend
type Response struct {
    Status int
    Data string
}

// global var used to save backend addresses
var backends string[]

/*
function used for putting json data from backend into map
response: json data from backend
return: map where key = short url, value = redirect url
*/
func processResponse(response string) map[string]string {
    var urls = make(map[string]string)
    keyVals := strings.Split(response, " ")
    if len(keyVals) == 0 {
        return urls
    }
    for i, keyVal := range keyVals {
        if i == len(keyVals) - 1 {
            // last element is always empty string
            break
        }
        keyValList := strings.Split(keyVal,"=")
        urls[keyValList[0]] = keyValList[1]
    }
    return urls
}

/*
gets response from backend for given route
backend: address of backend (e.g. http://localhost:8080)
route: route that gets hit on backend
return: response from backend or error
*/
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
function for index page (/)
returns: current short urls and mapped redirect url
         links to edit or delete
         form to add new short url
*/
func index(ctx iris.Context) {
    response := getResponse(backendUrl, "/fetch")
    if response.Status != 0 {
        ctx.ViewData("message", response.Data)
        ctx.View("message.html")
        return
    }
    urls := processResponse(response.Data)

    // Bind: {{.urls}} with url list
    ctx.ViewData("urls", urls)
    // Render template file: ./views/index.html
    ctx.View("index.html")
}

/*
function for add endpoint (/add/{shortUrl}?shortUrl=&redirect=)
this endpoint is ususally hit by the form in the index page
query param shortUrl: short url to add to map
query param redirect: redirect url to be associated w/ short url
return: renders success / fail message
*/
func add(ctx iris.Context) {
    shortUrl := ctx.URLParam("shortUrl")
    redirect := ctx.URLParam("redirect")

    // check if input is valid
    if checkInput(shortUrl) == -1 || checkInput(redirect) == -1 {
        ctx.ViewData("message", "invalid input. space (' ') or equals ('=') not allowed")
        ctx.View("message.html")
        return
    }

    route := "/add?shortUrl=" + shortUrl + "&redirect=" + redirect
    response := getResponse(backendUrl, route)
    ctx.ViewData("message", response.Data)
    ctx.View("message.html")
}

/*
function for delete endpoint (/delete/{shortUrl})
asks backend to delete given shortUrl and displays response from backend
return: renders success or fail message
*/
func del(ctx iris.Context) {
    shortUrl := ctx.Params().Get("shortUrl")
    route := "/delete/" + shortUrl
    response := getResponse(backendUrl, route)
    ctx.ViewData("message", response.Data)
    ctx.View("message.html")
}

/*
function for edit route (/edit/{shortUrl})
return: renders edit html containing form to update info or error message
*/
func edit(ctx iris.Context) {
    shortUrl := ctx.Params().Get("shortUrl")
    response := getResponse(backendUrl, "/"+shortUrl)

    if response.Status == 0 {
        // render edit template
        ctx.ViewData("shortUrl", shortUrl)
        ctx.ViewData("redirect", response.Data)
        ctx.View("edit.html")
    } else {
        // failed to edit, short url doesnt exists
        ctx.ViewData("message", response.Data)
        ctx.View("message.html")
    }
}

/*
check if valid input
input: string we want to save in backend
return: 0 if valid, -1 if not valid
*/
func checkInput(input string) int {
    if strings.Contains(input, "=") {
        return -1
    }
    if strings.Contains(input, " ") {
        return -1
    }
    return 0
}

/*
update route (/update/{shortUrl}?shortUrl=&redirect=)
updates key and value in url map based on query parameters
this route is usually hit from the form in the edit endpoint
query param shortUrl: new key in url map
query param redirect: new redirect value in url map
return: renders success or fail message
*/
func update(ctx iris.Context) {
    shortUrl := ctx.Params().Get("shortUrl")
    newShortUrl := ctx.URLParam("shortUrl")
    newRedirect := ctx.URLParam("redirect")

    // check if edits are valid
    if checkInput(newShortUrl) == -1 || checkInput(newRedirect) == -1 {
        ctx.ViewData("message", "invalid input. space (' ') or equals ('=') not allowed")
        ctx.View("message.html")
        return
    }

    route := "/update/"+shortUrl+"?shortUrl="+newShortUrl+"&redirect="+newRedirect
    response := getResponse(backendUrl, route)

    ctx.ViewData("message", response.Data)
    ctx.View("message.html")
}


/*
function for short url endpoints (/{shortUrl})
used for redirecting
*/
func redirect(ctx iris.Context) {
    shortUrl := ctx.Params().Get("shortUrl")
    response := getResponse(backendUrl, "/"+shortUrl)
    if response.Status == 0 {
        ctx.Redirect(response.Data, 301) // use 307 instead of 301 to avoid browser redirect caching
    } else {
        ctx.ViewData("message", response.Data)
        ctx.View("message.html")
    }
}

/*
function used to check if backend is alive
this function should be run in its own thread
backendAddr: backendAddr we want to ping
pingPeriod: how often should backend be ping'd
return: nothing, prints failure if no response from backend
*/
func pingBackend(backendAddr string, pingPeriod time.Duration) {
    for {
        // sleep
        time.Sleep(pingPeriod * time.Second)

        // ping
        response := getResponse(backendAddr, "/ping")
        if response.Status != 0 {
            timeNow := time.Now()
            err := "Detected Faliure on " + backendAddr + " at "
            err += timeNow.Format("2006-01-02 15:04:05 2609") + " UTC"
            fmt.Println(err)
        }
    }
}


/*
main func sets up webapp and listens for incoming http connections
*/
func main() {
    app := iris.New()

    tmpl := iris.HTML("./views", ".html")

    // Enable re-build on local template files changes.
    //tmpl.Reload(true)

    // Register the view engine to the views,
    // this will load the templates.
    app.RegisterView(tmpl)

    // add all our routes
    app.Get("/", index)
    app.Get("/add", add)
    app.Get("/delete/{shortUrl}", del)
    app.Get("/edit/{shortUrl}", edit)
    app.Get("/update/{shortUrl}", update)
    app.Get("/{shortUrl}", redirect)

    // parse args
    // listening port
    port := flag.String("listen", "8080", "frontend listening port")
    // address of backends
    backendStr := flag.String("backends", "", "address of backends (comma seperated)")
    flag.Parse()
    backends = strings.Split(*backendStr, ",")
    // add localhost if missing hostname
    for i, backend := range backends {
        if backend[0] == ':' {
            backends[i] = "http://localhost" + backend
        }
    }

    //apiUrl = *apiProtocol + "://" + *apiAddr + ":" + *apiPort

    // check if backend is alive every 5 secconds
    //go pingBackend(apiUrl, 5)

    // iris config
    config := iris.WithConfiguration(iris.Configuration {
        DisableStartupLog: true,
    })

    // start frontend
    fmt.Println("FRONTEND listening on " + *port)
    app.Listen(":"+*port, config)
}
