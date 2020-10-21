package main

import (
  "github.com/kataras/iris/v12"
  "os"
)

// struct used when sending json data
type Response struct {
    Status int      // 0 on success else failure
    Data string
}

/*
global map variable containing data for our CRUD app
the key is the name of shortened url
the value is the url we wish to redirect to
*/
var urls = make(map[string]string)

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
        if _, ok := urls[shortUrl]; ok {
            message = "cannot add '" + shortUrl + "': already exists."
        } else {
        // add url
        urls[shortUrl] = redirect
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
    if _, ok := urls[shortUrl]; ok {
        delete(urls, shortUrl)
        message = "successfully deleted " + shortUrl
        status = 0
    } else {
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
    if _, ok := urls[shortUrl]; ok {
        newShortUrl := ctx.URLParam("shortUrl")
        newRedirect := ctx.URLParam("redirect")

        // change of key requires deleting old and creating new entry
        if newShortUrl != shortUrl {
            delete(urls, shortUrl)
            urls[newShortUrl] = newRedirect
        } else {
            urls[shortUrl] = newRedirect
        }

        // render success message to client
        message = "succesfully updated '"+shortUrl+"'. short url: "+newShortUrl
        message += " redirect url: " + newRedirect
        status = 0
    } else {
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
    for key, value := range urls {
        message += key + "=" + value + " "
    }
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
    if redirect, ok := urls[shortUrl]; ok {
        message = redirect
        status = 0
    } else {
        // failed to update, short url doesnt exists
        message = shortUrl +" not found."
        status = 1
    }
    // send response
    response := Response{Status: status, Data: message}
    ctx.JSON(response)
}

func main() {
    //hardcode some initial data
    urls["tandon"] = "https://engineering.nyu.edu/"
    urls["classes"] = "https://classes.nyu.edu/"
    app := iris.New()

    // add all our routes
    app.Get("/fetch", fetch)
    app.Get("/add", add)
    app.Get("/update/{shortUrl}", update)
    app.Get("/delete/{shortUrl}", del)
    app.Get("/{shortUrl}", get)

    // listen on provided port, default 8080.
    if len(os.Args) == 1 {
        app.Listen(":8000")
    } else {
        app.Listen(":"+os.Args[1])
    }
}
