package main

import (
  "github.com/kataras/iris/v12"
  "os"
)

/*
global map variable containing data for our CRUD app
the key is the name of shortened url
the value is the url we wish to redirect to
*/
var urls = make(map[string]string)

/*
function for index page (/)
returns: current short urls and mapped redirect url
         links to edit or delete
         form to add new short url
*/
func index(ctx iris.Context) {
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
  var message string
  if shortUrl == "" {
    message = "no short url provided"
  } else if redirect == "" {
    message = "no redirect url provided"
  } else {
    if _, ok := urls[shortUrl]; ok {
      message = "cannot add  '" + shortUrl + "': already exists."
    } else {
    // add url
    urls[shortUrl] = redirect
    message = "succesfully added url. /" + shortUrl + " now redirects to " + redirect
    }
  }
  ctx.ViewData("message", message)
  ctx.View("message.html")
}

/*
function for delete endpoint (/delete/{shortUrl})
return: renders success or fail message
*/
func del(ctx iris.Context) {
  var message string
  shortUrl := ctx.Params().Get("shortUrl")
  // delete url
  if _, ok := urls[shortUrl]; ok {
   delete(urls, shortUrl)
   message = "successfully deleted " + shortUrl
  } else {
    // failed to delete, short url doesnt exists
    message = "failed to delete '" +shortUrl +"': not found."
  }
  ctx.ViewData("message", message)
  ctx.View("message.html")
}

/*
function for edit route (/edit/{shortUrl})
return: renders edit html containing form to update info or error message
*/
func edit(ctx iris.Context) {
  shortUrl := ctx.Params().Get("shortUrl")
  if redirect, ok := urls[shortUrl]; ok {
    // render edit template
    ctx.ViewData("shortUrl", shortUrl)
    ctx.ViewData("redirect", redirect)
    ctx.View("edit.html")
  } else {
    // failed to edit, short url doesnt exists
    message := "failed to edit '" +shortUrl +"': not found."
    ctx.ViewData("message", message)
    ctx.View("message.html")
  }
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
  var message string
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
    ctx.ViewData("message", message)
    ctx.View("message.html")
  } else {
    // failed to update, short url doesnt exists
    message = "failed to update '" +shortUrl +"': not found."
    ctx.ViewData("message", message)
    ctx.View("message.html")
  }
}

/*
function for short url endpoints (/{shortUrl})
used for redirecting
*/
func redirect(ctx iris.Context) {
  shortUrl := ctx.Params().Get("shortUrl")
  if redirect, ok := urls[shortUrl]; ok {
    // 307 response code stops redirects from being cached
    // 301 allows redirect caching
    // 301 would be better, 307 works better for demo'ing.
    ctx.Redirect(redirect, 307)
  } else {
    ctx.ViewData("message", "short url does not exists")
    ctx.View("message.html")
  }
}

/*
main func sets up webapp and listens for incoming http connections
*/
func main() {
  //hardcode some initial data
  urls["tandon"] = "https://engineering.nyu.edu/"
  urls["classes"] = "https://classes.nyu.edu/"
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
  app.Get("/edit/{shortUrl}", edit)
  app.Get("/update/{shortUrl}", update)
  app.Get("/delete/{shortUrl}", del)
  app.Get("/{shortUrl}", redirect)

  // listen on provided port, default 8080.
  if len(os.Args) == 1 {
    app.Listen(":8080")
  } else {
    app.Listen(":"+os.Args[1])
  }
}
