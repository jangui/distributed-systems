package main

import (
  "github.com/kataras/iris/v12"
  "flag"
  "sync"
  "fmt"
)

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

func main() {
    //hardcode some initial data
    data.urls = make(map[string]string)
    data.urls["tandon"] = "https://engineering.nyu.edu/"
    data.urls["classes"] = "https://classes.nyu.edu/"
    app := iris.New()

    // add all our routes
    app.Get("/fetch", fetch)
    app.Get("/add", add)
    app.Get("/update/{shortUrl}", update)
    app.Get("/delete/{shortUrl}", del)
    app.Get("/ping", ping)
    app.Get("/{shortUrl}", get)

    // parse args
    port := flag.String("port", "8000", "backend listening port")
    flag.Parse()

    // iris config
    config := iris.WithConfiguration(iris.Configuration {
        DisableStartupLog: true,
    })
    // start backend
    fmt.Println("BACKEND listening on " + *port)
    app.Listen(":"+*port, config)
}
