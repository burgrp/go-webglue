# go-webglue

The Web Framework designed for **small applications and embedded/IoT systems** where simplicity and efficiency matter:

- **Single Binary Deployment** - Everything compiles into one executable with embedded resources. Copy and run.
- **Efficient Resource Usage** - Go's performance and small memory footprint make it perfect for resource-constrained environments.
- **Simple Architecture** - No external dependencies, databases, or complex build pipelines required.
- **Rapid Development** - Build complete applications in hours, not days.

Perfect for control panels, monitoring tools, device interfaces, and internal utilities.


## Easy to use

A Go web framework that automatically bridges backend and frontend code. Expose Go struct methods as HTTP APIs, stream real-time events via SSE, and serve optimized static resources with zero configuration.

```go
// Backend: Define your API
type MyApi struct{}

func (api *MyApi) GetUser(id int) (User, error) {
    return findUser(id), nil
}
```

```javascript
// Frontend: Just call it
let user = await api.mymodule.getUser(42);
```

No route definitions. No API contracts. No build steps.

## Documentation

- **[Getting Started Guide](doc/getting-started.md)** - Build your first app step-by-step
- **[Architecture Overview](doc/architecture.md)** - How go-webglue works under the hood
- **[API Reference](doc/api-reference.md)** - Complete API documentation
- **[Frontend Guide](doc/frontend-guide.md)** - Building UIs with webglue.js
- **[Real-time Events](doc/events.md)** - Server-to-client event streaming
- **[Authentication](doc/authentication.md)** - Implementing auth with CallChecker
- **[Deployment](doc/deployment.md)** - Production best practices
- **[Examples](doc/examples.md)** - Code samples and patterns

## Features

- **Automatic API Bridging** - Go methods instantly become JavaScript functions
- **Real-time Events** - Built-in SSE support for live updates
- **Zero Build Required** - ES modules with automatic import maps
- **Smart Resource Handling** - Auto-minification, caching, and hot-reload
- **Type-Safe Parameters** - Automatic JSON marshaling with reflection
- **Context Injection** - Pass context.Context, custom types, or request data
- **Modular Architecture** - Organize code into reusable modules

## Quick Start

### Installation

```bash
go get github.com/burgrp/go-webglue/pkg
```

### Hello World

**Create your API** (`main.go`)

```go
package main

import (
    "embed"
    "net/http"
    webglue "github.com/burgrp/go-webglue/pkg"
)

//go:embed client/*
var clientResources embed.FS

type HelloApi struct{}

func (api *HelloApi) Greet(name string) string {
    return "Hello, " + name + "!"
}

func main() {
    handler, _ := webglue.NewHandler(webglue.Options{
        Modules: []*webglue.Module{{
            Name:      "hello",
            Resources: &clientResources,
            Api:       &HelloApi{},
        }},
    })

    http.ListenAndServe(":8080", handler)
}
```

**Create your UI** (`client/home.page.js`)

```javascript
import { api, tags } from "webglue";

let { DIV, BUTTON, TEXT } = tags;

export default {
    title: "Hello World",
    async render() {
        let input, output;
        return [
            DIV("container", [
                TEXT(el => input = el).val("World"),
                BUTTON().text("Greet").click(async () => {
                    let greeting = await api.hello.greet(input.val());
                    output.text(greeting);
                }),
                DIV(el => output = el)
            ])
        ];
    }
}
```

**Run it**

```bash
go run main.go
# Visit http://localhost:8080
```

## Key Concepts

### Automatic API Exposure

Go methods are automatically discovered and exposed:

```go
func (api *MyApi) CalculateSum(a, b int) int {
    return a + b  // Becomes: api.myApi.calculateSum(a, b)
}

func (api *MyApi) GetUserData(userId string) (User, error) {
    // JavaScript receives: promise that resolves to User
}
```

### Real-time Events

Stream data from server to clients:

```go
// Server
updateEvent := webglue.NewEvent("dataChanged")
updateEvent.Emit(newData)
```

```javascript
// Client
DIV().onMymoduleDataChanged((el, newData) => {
    el.text(JSON.stringify(newData));
})
```

### Development Mode

Hot-reload without rebuilding:

```bash
MYMODULE_DEV=/path/to/client go run main.go
# Edit JS/CSS files and refresh browser - changes appear instantly
```

## Project Structure

```
myproject/
├── main.go              # Backend entry point
├── go.mod
├── client/              # Frontend code (embedded)
│   ├── home.page.js     # Main page
│   ├── login.page.js    # Additional pages
│   └── styles.css
└── api.go               # API implementation
```

## Use Cases

- Internal dashboards and admin interfaces
- IoT control panels with real-time monitoring
- Data visualization with live updates
- API prototyping with quick UI feedback
- Single-binary deployable applications

## License

This project is open source. Check the repository for license details.

## Links

- **GitHub**: [github.com/burgrp/go-webglue](https://github.com/burgrp/go-webglue)
- **Issues**: [Report bugs or request features](https://github.com/burgrp/go-webglue/issues)
