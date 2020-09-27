package main

import "github.com/kataras/iris/v12"

func main() {
    app := iris.New()

    tmpl := iris.HTML("./views", ".html")

    // Enable re-build on local template files changes.
    tmpl.Reload(true)

    // Register a custom template func:
    tmpl.AddFunc("greet", func(s string) string {
        return "Greetings " + s + "!"
    })

    // Register the view engine to the views,
    // this will load the templates.
    app.RegisterView(tmpl)

    // Method:    GET
    // Resource:  http://localhost:8080
    app.Get("/", func(ctx iris.Context) {
        // Bind: {{.message}} with "Hello world!"
        ctx.ViewData("message", "Hello world!")
        // Render template file: ./views/hi.html
        ctx.View("hi.html")
    })

    app.Listen(":8080")
}
