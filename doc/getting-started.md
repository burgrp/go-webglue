# Getting Started with go-webglue

This guide will walk you through creating your first go-webglue application from scratch.

## Prerequisites

- Go 1.20 or higher
- Basic knowledge of Go and JavaScript
- A text editor or IDE

## Installation

Add go-webglue to your project:

```bash
go get github.com/burgrp/go-webglue/pkg
```

## Your First Application

Let's build a simple calculator app that demonstrates the core concepts.

### Step 1: Project Setup

Create a new directory and initialize a Go module:

```bash
mkdir calculator-app
cd calculator-app
go mod init example.com/calculator
go get github.com/burgrp/go-webglue/pkg
```

Create the following structure:

```
calculator-app/
├── main.go
└── client/
    └── home.page.js
```

### Step 2: Create the Backend API

Create `main.go`:

```go
package main

import (
    "embed"
    "errors"
    "net/http"
    webglue "github.com/burgrp/go-webglue/pkg"
)

//go:embed client/*
var clientResources embed.FS

type CalculatorApi struct{}

func (api *CalculatorApi) Add(a, b float64) float64 {
    return a + b
}

func (api *CalculatorApi) Subtract(a, b float64) float64 {
    return a - b
}

func (api *CalculatorApi) Multiply(a, b float64) float64 {
    return a * b
}

func (api *CalculatorApi) Divide(a, b float64) (float64, error) {
    if b == 0 {
        return 0, errors.New("division by zero")
    }
    return a / b, nil
}

func main() {
    options := webglue.Options{
        Modules: []*webglue.Module{
            {
                Name:      "calculator",
                Resources: &clientResources,
                Api:       &CalculatorApi{},
            },
        },
    }

    handler, err := webglue.NewHandler(options)
    if err != nil {
        panic(err)
    }

    port := "8080"
    println("Calculator app running on http://localhost:" + port)
    err = http.ListenAndServe(":"+port, handler)
    if err != nil {
        panic(err)
    }
}
```

### Step 3: Create the Frontend

Create `client/home.page.js`:

```javascript
import { api, asy, tags } from "webglue";

let { DIV, BUTTON, NUMBER, H1 } = tags;

export default {
    title: "Calculator",

    async render() {
        let inputA, inputB, resultDiv;

        const calculate = (operation) => {
            asy(async () => {
                try {
                    const a = parseFloat(inputA.val());
                    const b = parseFloat(inputB.val());

                    let result;
                    switch(operation) {
                        case 'add':
                            result = await api.calculator.add(a, b);
                            break;
                        case 'subtract':
                            result = await api.calculator.subtract(a, b);
                            break;
                        case 'multiply':
                            result = await api.calculator.multiply(a, b);
                            break;
                        case 'divide':
                            result = await api.calculator.divide(a, b);
                            break;
                    }

                    resultDiv.text(`Result: ${result}`);
                } catch (e) {
                    resultDiv.text(`Error: ${e.message}`).css('color', 'red');
                }
            });
        };

        return [
            H1().text("Calculator"),
            DIV([
                NUMBER(el => inputA = el).val(10),
                NUMBER(el => inputB = el).val(5),
            ]),
            DIV([
                BUTTON().text("+").click(() => calculate('add')),
                BUTTON().text("-").click(() => calculate('subtract')),
                BUTTON().text("×").click(() => calculate('multiply')),
                BUTTON().text("÷").click(() => calculate('divide')),
            ]),
            DIV(el => resultDiv = el).text("Result: ")
        ];
    }
}
```

### Step 4: Run Your App

```bash
go run main.go
```

Visit http://localhost:8080 in your browser. You now have a working calculator!

## How It Works

### Backend Magic

1. **Module Definition**: You define a module with a name, embedded resources, and an API struct
2. **Automatic Exposure**: All exported methods on your API struct become HTTP endpoints
3. **Path Convention**: Method `Add` becomes available at `/api/calculator/add`

### Frontend Magic

1. **API Discovery**: On startup, the client calls `/api/webglue/discover` to find all available APIs
2. **Dynamic Proxy**: The `api` object is populated with methods matching your Go API
3. **Automatic Marshaling**: Parameters are JSON-encoded, results are decoded

### Error Handling

- Go errors are automatically caught and returned as `{"error": "message"}`
- JavaScript receives a rejected Promise with the error message
- Use try/catch or the `asy()` helper to handle errors

## Next Steps

Now that you have a basic app running:

1. **Add Styling**: Create `client/styles.css` - it will be automatically included
2. **Multiple Pages**: Add more `.page.js` files for different routes
3. **Real-time Updates**: Add events to push data from server to client ([Events Guide](events.md))
4. **Authentication**: Implement `CallChecker` interface ([Auth Guide](authentication.md))
5. **Complex Types**: Pass structs and arrays between Go and JavaScript

## Development Tips

### Hot Reload

For faster development, use development mode:

```bash
CALCULATOR_DEV=$(pwd)/client go run main.go
```

Now you can edit JS/CSS files and just refresh the browser - no rebuild needed!

### Debugging

Add logging to your API methods:

```go
func (api *CalculatorApi) Add(a, b float64) float64 {
    fmt.Printf("Add called: %f + %f\n", a, b)
    return a + b
}
```

Use browser DevTools to inspect network calls to `/api/calculator/add`

### Project Organization

As your app grows, organize it like this:

```
myapp/
├── main.go           # Entry point
├── api/
│   ├── users.go      # User API
│   └── products.go   # Product API
├── models/
│   └── types.go      # Shared types
└── client/
    ├── home.page.js
    ├── users.page.js
    └── styles.css
```

## Common Patterns

### Returning Multiple Values

```go
func (api *Api) DivMod(a, b int) (int, int, error) {
    if b == 0 {
        return 0, 0, errors.New("division by zero")
    }
    return a / b, a % b, nil
}
```

```javascript
let [quotient, remainder] = await api.mymodule.divMod(10, 3);
// quotient = 3, remainder = 1
```

### Complex Parameters

```go
type UserInput struct {
    FirstName string `json:"firstName"`
    LastName  string `json:"lastName"`
    Age       int    `json:"age"`
}

func (api *Api) CreateUser(input UserInput) (string, error) {
    // Create user...
    return "User created!", nil
}
```

```javascript
let message = await api.mymodule.createUser({
    firstName: "John",
    lastName: "Doe",
    age: 30
});
```

## Troubleshooting

### "Module not found" error

Make sure your embedded filesystem includes the `client` directory:

```go
//go:embed client/*
var clientResources embed.FS
```

### API methods not appearing in JavaScript

- Check that methods are exported (start with uppercase letter)
- Verify the module is added to `Options.Modules`
- Check browser console for discovery errors

### Changes not reflecting

- In production mode, restart the Go server
- In development mode, just refresh the browser
- Clear browser cache if needed

## What's Next?

- [Architecture Overview](architecture.md) - Understand how go-webglue works
- [Frontend Guide](frontend-guide.md) - Master the client-side framework
- [Examples](examples.md) - See more complete applications
