# API Reference

Complete reference for go-webglue types, functions, and interfaces.

## Go API

### Core Types

#### `Module`

Represents a logical component of your application.

```go
type Module struct {
    Name      string      // Module identifier (used in URLs and JS)
    Resources *embed.FS   // Embedded client-side files
    Events    []*Event    // Events this module can emit
    Api       any         // Struct with exported methods to expose
}
```

**Example**:
```go
//go:embed client/*
var clientResources embed.FS

module := &webglue.Module{
    Name:      "users",
    Resources: &clientResources,
    Events:    []*webglue.Event{userCreatedEvent},
    Api:       &UsersApi{},
}
```

#### `Options`

Configuration for creating the HTTP handler.

```go
type Options struct {
    Modules   []*Module  // Your application modules
    IndexHtml string     // Custom HTML template (optional)
}
```

**Default IndexHtml**: If not provided, uses `webglue.DefaultIndexHtml`

**Custom HTML Requirements**:
- Must include `{WEBGLUE}` placeholder
- go-webglue replaces it with stylesheet links and import map

**Example**:
```go
options := webglue.Options{
    Modules: []*webglue.Module{myModule},
    IndexHtml: `
<!DOCTYPE html>
<html>
  <head>
    <title>My App</title>
{WEBGLUE}
    <script type="module">
      import {start} from "webglue";
      $(document).ready(start);
    </script>
  </head>
  <body></body>
</html>
    `,
}
```

#### `Event`

Represents a server-to-client event stream.

```go
type Event struct {
    Module  string       // Auto-populated by framework
    Name    string       // Event name
    servers []*sse.Server // Internal
}
```

**Creating Events**:
```go
updateEvent := webglue.NewEvent("dataUpdated")
```

**Emitting Events**:
```go
event.Emit(param1, param2, ...) // Variadic parameters
```

### Functions

#### `NewHandler`

Creates the main HTTP handler for your application.

```go
func NewHandler(options Options) (*http.ServeMux, error)
```

**Returns**:
- `*http.ServeMux`: Handler ready to pass to `http.ListenAndServe`
- `error`: Error if initialization fails

**Example**:
```go
handler, err := webglue.NewHandler(webglue.Options{
    Modules: []*webglue.Module{myModule},
})
if err != nil {
    panic(err)
}

http.ListenAndServe(":8080", handler)
```

#### `NewEvent`

Creates a new event that can be emitted to clients.

```go
func NewEvent(name string) *Event
```

**Example**:
```go
tickEvent := webglue.NewEvent("tick")
tickEvent.Emit(time.Now().Unix())
```

### Interfaces

#### `CallChecker`

Implement this interface on your API struct to inject custom parameters or perform authentication.

```go
type CallChecker interface {
    CheckCall(request *http.Request, functionName string) ([]any, error)
}
```

**Parameters**:
- `request`: The HTTP request
- `functionName`: The Go method name being called (e.g., "GetUser")

**Returns**:
- `[]any`: Parameters to inject into the function call
- `error`: If non-nil, the API call fails with this error

**Example**:
```go
type MyApi struct {
    db *Database
}

func (api *MyApi) CheckCall(req *http.Request, funcName string) ([]any, error) {
    // Extract and validate auth token
    token := req.Header.Get("Authorization")
    user, err := api.db.ValidateToken(token)
    if err != nil {
        return nil, errors.New("unauthorized")
    }

    // Inject user into all API calls
    return []any{user}, nil
}

// Now all methods can receive User
func (api *MyApi) GetProfile(user *User) (*Profile, error) {
    return api.db.GetProfile(user.ID)
}
```

**Important**: The `CheckCall` method itself cannot be called via API.

### API Method Conventions

Your API methods are automatically exposed with these rules:

#### Naming Convention

- **Go**: `PascalCase` (e.g., `GetUser`)
- **JavaScript**: `camelCase` (e.g., `api.module.getUser`)

#### Parameter Types

Parameters can be:

1. **Injected Automatically** (not from JavaScript):
   - `context.Context` - from `request.Context()`
   - `*http.Request` - the HTTP request
   - Any type returned by `CallChecker`

2. **From JavaScript** (JSON unmarshaled):
   - Primitives: `int`, `float64`, `string`, `bool`
   - Structs with JSON tags
   - Slices and maps
   - Nested structures

**Example**:
```go
type UserInput struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (api *MyApi) CreateUser(
    ctx context.Context,        // Injected: request context
    currentUser *User,          // Injected: from CallChecker
    input UserInput,            // From JS: unmarshaled from JSON body
) (string, error) {
    // Implementation
}
```

```javascript
// JavaScript call
let userId = await api.mymodule.createUser({
    name: "John",
    email: "john@example.com"
});
```

#### Return Types

Methods can return:

1. **Single Value**:
   ```go
   func (api *Api) GetCount() int { return 42 }
   ```
   ```javascript
   let count = await api.module.getCount(); // 42
   ```

2. **Value and Error**:
   ```go
   func (api *Api) GetUser(id int) (*User, error) { ... }
   ```
   ```javascript
   try {
       let user = await api.module.getUser(42);
   } catch (err) {
       console.error(err.message);
   }
   ```

3. **Multiple Values**:
   ```go
   func (api *Api) DivMod(a, b int) (int, int, error) {
       return a/b, a%b, nil
   }
   ```
   ```javascript
   let [quotient, remainder] = await api.module.divMod(10, 3);
   ```

4. **Multiple Values (no error)**:
   ```go
   func (api *Api) GetMinMax(nums []int) (int, int) { ... }
   ```
   ```javascript
   let [min, max] = await api.module.getMinMax([1, 5, 3]);
   ```

**Error Handling**:
- Any return value of type `error` that is non-nil stops processing
- Error is returned to JavaScript as rejected Promise
- Other return values are ignored if error is present

## JavaScript API

### Imports

```javascript
import { api, asy, error, goto, tags, start } from "webglue";
```

### Objects

#### `api`

Dynamic proxy object containing all discovered API methods.

**Structure**:
```javascript
api.moduleName.methodName(...args)
```

**Returns**: `Promise` that resolves to the result or rejects with error

**Example**:
```javascript
// Call Go method: func (api *MyApi) GetUser(id int) (*User, error)
let user = await api.mymodule.getUser(42);
```

#### `tags`

Object containing tag factory functions.

**Available Tags**:
- `DIV`, `SPAN`, `H1`, `H2`, `H3`
- `BUTTON`, `AHREF`, `LABEL`, `PAR`
- `IMG`, `ICON`, `SETOFF`
- `FORM`, `INPUT`, `TEXT`, `PASSWORD`, `NUMBER`
- `CHECKBOX`, `RADIO`, `SELECT`, `OPTION`, `TEXTAREA`
- `TABLE`, `TR`, `TD`, `TH`
- `FIELDSET`, `IFRAME`

**Usage**:
```javascript
let { DIV, BUTTON, TEXT } = tags;

DIV("css-class", [
    TEXT().val("Hello"),
    BUTTON().text("Click me")
])
```

**Arguments** (processed in order):
- `string`: Added as CSS class
- `object`: Passed to jQuery `.prop()`
- `array`: Elements appended as children
- `function`: Called with element, return value processed recursively

**Function Callbacks**:
```javascript
DIV(el => {
    el.addClass("dynamic");
    return "Content"; // Processed
})
```

### Functions

#### `start()`

Initializes the webglue application. Called automatically if using default HTML.

```javascript
import { start } from "webglue";
$(document).ready(start);
```

**Process**:
1. Discovers API endpoints
2. Sets up event stream
3. Initializes routing
4. Renders initial page

#### `goto(url, replace = false)`

Navigate to a different page.

```javascript
goto("/users?id=42")           // Add to history
goto("/home", true)            // Replace current history entry
```

**Parameters**:
- `url`: Path with optional query string
- `replace`: If true, replaces current history entry

#### `asy(asyncFunction)`

Wrapper for async functions that handles errors.

```javascript
BUTTON().click(() => {
    asy(async () => {
        let result = await api.mymodule.doSomething();
        console.log(result);
    });
});
```

**Behavior**:
- Catches errors and calls `error(e)` or `page.error(e)`
- Prevents unhandled promise rejections

#### `error(e)`

Default error handler.

```javascript
error(new Error("Something went wrong"));
```

**Behavior**:
- Calls `page.error(e)` if defined
- Otherwise shows `alert(e)`

### Page Module Format

Each page is a JavaScript module exporting a default object.

**Location**: `client/{pagename}.page.js`

**Required Exports**:

```javascript
export default {
    title: "Page Title",              // Browser title
    render: async (url, params) => {  // Returns array of jQuery elements
        return [
            DIV().text("Content")
        ];
    },

    // Optional:
    error: (e) => { ... },            // Custom error handler
    check: async (url, params) => {   // Pre-render check
        // Return URL string to redirect, or falsy to continue
    }
}
```

**Example**:
```javascript
// client/users.page.js
import { api, tags } from "webglue";

let { DIV, H1 } = tags;

export default {
    title: "Users",

    async render(url, params) {
        let users = await api.mymodule.listUsers();

        return [
            H1().text("Users"),
            DIV(users.map(user =>
                DIV().text(user.name)
            ))
        ];
    }
}
```

### Event Handling

#### Module Events

Events are auto-registered based on discovery:

```javascript
// For event named "updated" in module "users"
$(element).onUsersUpdated((el, ...params) => {
    // Handle event
});
```

**Naming Convention**:
- `on` + `ModuleName` (capitalized) + `EventName` (capitalized)
- Example: `onUsersDataChanged`, `onCoreReady`

#### Built-in Events

**webglue.tick**

Fires every second on all elements:

```javascript
DIV().onWebglueTick(handler) // or
DIV().on("webglue.tick", handler)
```

### Local Storage Headers

Set headers to be sent with all API requests:

```javascript
// Set in localStorage (persists across sessions)
localStorage.setItem("webglue.headers.Authorization", "Bearer token123");

// Set in sessionStorage (cleared on browser close)
sessionStorage.setItem("webglue.headers.X-Custom", "value");
```

**Prefix**: `webglue.headers.`

**Example Use Case**: Authentication tokens
```javascript
// After login
localStorage.setItem("webglue.headers.Authorization", `Bearer ${token}`);

// All subsequent API calls include: Authorization: Bearer token123

// Logout
localStorage.removeItem("webglue.headers.Authorization");
```

## Environment Variables

### Development Mode

Enable file-system serving for a module:

```bash
MODULENAME_DEV=/path/to/client go run main.go
```

**Module Name**: Uppercase version of module name
**Example**: Module "myApp" → `MYAPP_DEV=/path/to/files`

**Benefits**:
- No rebuild needed for JS/CSS changes
- No minification (easier debugging)
- Instant feedback

## Constants

### Go Constants

```go
webglue.ContentTypeHeader   = "Content-Type"
webglue.ContentTypeJson     = "application/json"
webglue.ContentLengthHeader = "Content-Length"
webglue.WebgluePlaceholder  = "{WEBGLUE}"
webglue.DefaultIndexHtml    = "..." // Default HTML template
webglue.EventStreamName     = "webglue"
```

## Error Handling

### Go Errors

```go
func (api *Api) GetUser(id int) (*User, error) {
    if id < 0 {
        return nil, errors.New("invalid user ID")
    }
    // ...
}
```

### JavaScript Errors

```javascript
try {
    await api.module.getUser(-1);
} catch (err) {
    console.error(err.message); // "invalid user ID"
}
```

## Type Mappings

| Go Type | JavaScript Type | Notes |
|---------|----------------|-------|
| `int`, `int64`, `float64` | `number` | |
| `string` | `string` | |
| `bool` | `boolean` | |
| `struct` | `object` | Uses JSON tags |
| `[]T` | `Array` | |
| `map[string]T` | `object` | |
| `time.Time` | `string` | ISO 8601 format |
| `nil` | `null` | |
| `error` | Exception | Becomes rejected Promise |

## Next Steps

- [Frontend Guide](frontend-guide.md) - Build UIs with webglue
- [Events Guide](events.md) - Real-time communication
- [Examples](examples.md) - Code samples
