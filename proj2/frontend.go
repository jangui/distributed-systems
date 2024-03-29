package main

import (
    "github.com/kataras/iris/v12"
    "net/http"
    "encoding/json"
    "io/ioutil"
    "strings"
    "flag"
    "fmt"
)

// response struct used to decode json from backend
type Response struct {
    Status int
    Data string
}

// global var used to save backend address
var apiUrl string

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
route: route that gets hit on backend
return: response from backend
*/
func getResponse(route string) Response {
    resp, err := http.Get(apiUrl+route)
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
    response := getResponse("/fetch")
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
    response := getResponse(route)
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
    response := getResponse(route)
    ctx.ViewData("message", response.Data)
    ctx.View("message.html")
}

/*
function for edit route (/edit/{shortUrl})
return: renders edit html containing form to update info or error message
*/
func edit(ctx iris.Context) {
    shortUrl := ctx.Params().Get("shortUrl")
    response := getResponse("/"+shortUrl)

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
    response := getResponse(route)

    ctx.ViewData("message", response.Data)
    ctx.View("message.html")
}


/*
function for short url endpoints (/{shortUrl})
used for redirecting
*/
func redirect(ctx iris.Context) {
    shortUrl := ctx.Params().Get("shortUrl")
    response := getResponse("/"+shortUrl)
    if response.Status == 0 {
        // 307 response code stops redirects from being cached
        // 301 allows redirect caching
        // 301 would be better, 307 works better for demo'ing.
        ctx.Redirect(response.Data, 307)
    } else {
        ctx.ViewData("message", response.Data)
        ctx.View("message.html")
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
    apiAddr := flag.String("apiAddr", "localhost", "backend address")
    apiPort := flag.String("apiPort", "8000", "backend port")
    apiProtocol := flag.String("apiProtocol", "http", "backend protocol")
    port := flag.String("port", "8080", "frontend listening port")
    flag.Parse()

    apiUrl = *apiProtocol + "://" + *apiAddr + ":" + *apiPort
    fmt.Print("Frontend: ")
    app.Listen(":"+*port)
}
