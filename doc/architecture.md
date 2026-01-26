# Architecture Overview

This document explains how go-webglue works under the hood.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      Browser                            │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Page Modules (.page.js)                          │  │
│  │  ├─ home.page.js                                  │  │
│  │  ├─ users.page.js                                 │  │
│  │  └─ ...                                           │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │  webglue.js (Core Client Library)                 │  │
│  │  ├─ API Proxy (api.module.method)                 │  │
│  │  ├─ Event Handlers (SSE)                          │  │
│  │  ├─ SPA Router                                    │  │
│  │  └─ Tag Factories (DIV, BUTTON, etc.)             │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                          │
                    HTTP/SSE
                          │
┌─────────────────────────────────────────────────────────┐
│                    Go Server                            │
│  ┌───────────────────────────────────────────────────┐  │
│  │  http.ServeMux                                    │  │
│  │  ├─ /           → StaticHandler                   │  │
│  │  ├─ /api/*      → ApiHandler                      │  │
│  │  └─ /events     → EventHandler (SSE)              │  │
│  └───────────────────────────────────────────────────┘  │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Modules                                          │  │
│  │  ├─ webglue (core)                                │  │
│  │  ├─ your-module-1                                 │  │
│  │  │   ├─ API Struct                                │  │
│  │  │   ├─ Events                                    │  │
│  │  │   └─ Resources (embed.FS)                      │  │
│  │  └─ your-module-2                                 │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Module System

**Location**: `mux_handler.go`

A Module is the fundamental building block:

```go
type Module struct {
    Name      string      // Used in URLs and JS API paths
    Resources *embed.FS   // Client-side files (JS, CSS, HTML)
    Events    []*Event    // Server-to-client event streams
    Api       any         // Struct with exported methods
}
```

**Key Points**:
- Each module has its own namespace (e.g., `api.mymodule.*`)
- Resources are embedded at compile time using `//go:embed`
- The core "webglue" module is always included automatically

### 2. Handler Chain

**Location**: `mux_handler.go:20-48`

When you call `webglue.NewHandler()`, three HTTP handlers are created:

#### StaticHandler (`/`)

**Location**: `static_handler.go`

Responsibilities:
- Serves the main `index.html` (with auto-generated import maps)
- Serves embedded CSS/JS/images
- Minifies resources in production mode
- Supports file-system fallback in dev mode

**Process**:
1. Check if path matches dev mode file → serve from filesystem
2. Check if path matches cached resource → serve from memory
3. Otherwise → serve `index.html` (SPA fallback)

**Import Map Generation**:
```javascript
// Auto-generated in HTML:
<script type="importmap">
{
  "imports": {
    "webglue": "./webglue.js",
    "mymodule": "./mymodule.js"
  }
}
</script>
```

#### ApiHandler (`/api/`)

**Location**: `api_handler.go`

Responsibilities:
- Routes HTTP calls to Go methods
- Uses reflection to invoke methods dynamically
- Handles parameter marshaling/unmarshaling
- Returns results as JSON

**Process**:
1. Parse URL: `/api/{module}/{function}`
2. Find module by name
3. Capitalize function name (JS `camelCase` → Go `PascalCase`)
4. Use reflection to find method
5. Build parameter list:
   - Inject typed params (context, custom types via CallChecker)
   - Unmarshal remaining params from JSON body
6. Invoke method with `reflect.Value.Call()`
7. Process return values:
   - If any return is error type and non-nil → return error
   - Single non-error return → `{"result": value}`
   - Multiple returns → `{"result": [val1, val2, ...]}`

**Reflection Magic**:

```go
// Example method
func (api *MyApi) GetUser(ctx context.Context, token string, id int) (User, error)

// Parameter resolution:
// - ctx: Injected from request.Context()
// - token: Injected via CallChecker (if implemented)
// - id: Unmarshaled from JSON body [42]
```

#### EventHandler (`/events`)

**Location**: `event_handler.go`

Responsibilities:
- Establishes Server-Sent Events (SSE) connection
- Streams events to connected clients
- Manages automatic reconnection

**Process**:
1. Client connects to `/events?stream=webglue`
2. Server keeps connection open
3. When `event.Emit()` is called, data is broadcast to all connections
4. Client receives event and triggers jQuery custom events

### 3. Client Library

**Location**: `pkg/client/webglue.js`

#### Startup Sequence

```javascript
start() → startAsync()
  ↓
  1. Discover APIs (fetch /api/webglue/discover)
  2. Build api.* proxy objects
  3. Register event handlers
  4. Set up SPA routing
  5. Connect to SSE stream
  6. Render initial page
```

#### API Discovery

On startup:
```javascript
GET /api/webglue/discover
→ {
    "mymodule": {
        "functions": ["getUser", "createUser"],
        "events": ["userCreated"]
    }
}
```

Creates:
```javascript
api.mymodule.getUser = (...params) =>
    fetch('/api/mymodule/getUser', {
        method: 'POST',
        body: JSON.stringify(params)
    })
```

#### SPA Routing

**URL Format**: `/{pageName}?param=value`

**Process**:
1. User navigates to `/users?id=42`
2. Router extracts `pageName = "users"`, `params = {id: "42"}`
3. Import `users.page.js` dynamically
4. Call `page.render(url, params)`
5. Replace body content with returned elements

**History API**:
- `goto(url)` → `pushState` (add to history)
- `goto(url, true)` → `replaceState` (replace current)
- Browser back/forward triggers `onpopstate` → re-render

#### Tag Factories

Helper functions to create DOM elements with jQuery:

```javascript
DIV("class-name", [         // String = CSS class
    { id: "myDiv" },        // Object = jQuery .prop()
    el => console.log(el),  // Function = callback with element
    "Text content"          // Primitives = appended
])
```

## Request Flow Examples

### API Call Flow

```
Client                          Server
  │                               │
  │  POST /api/users/getUser      │
  │  Body: [42]                   │
  ├──────────────────────────────>│
  │                               │ ApiHandler.ServeHTTP()
  │                               │   ├─ Parse: module="users", func="GetUser"
  │                               │   ├─ Find Module & Method (reflection)
  │                               │   ├─ Build params: [ctx, 42]
  │                               │   ├─ Call: GetUser(ctx, 42)
  │                               │   └─ Marshal result
  │  {"result": {...}}            │
  │<──────────────────────────────│
  │                               │
```

### Event Flow

```
Server                          Client
  │                               │
  │ event.Emit(data)              │
  ├──────────────────────────────>│ EventSource.onmessage
  │                               │   ├─ Parse JSON
  │                               │   ├─ Build event name: "webglue-users-updated"
  │                               │   └─ Trigger: $("*").trigger(eventName, data)
  │                               │
  │                               │ jQuery event bubbles
  │                               │   └─ Element.onUsersUpdated() called
```

### Page Navigation Flow

```
User Action                     webglue.js
  │                               │
  │ Click <a href="/users">      │
  ├──────────────────────────────>│ Event listener
  │                               │   ├─ preventDefault()
  │                               │   ├─ goto("/users")
  │                               │   ├─ history.pushState()
  │                               │   ├─ import("./users.page.js")
  │                               │   ├─ page.render()
  │                               │   └─ $("body").append(elements)
  │                               │
  │ Page updated                  │
  │<──────────────────────────────│
```

## Design Patterns

### Reflection-Based Routing

Instead of manually registering routes, go-webglue uses reflection to discover methods:

```go
apiType := reflect.TypeOf(module.Api)
for i := 0; i < apiType.NumMethod(); i++ {
    method := apiType.Method(i)
    // method.Name becomes available as API endpoint
}
```

**Pros**:
- Zero boilerplate
- Automatic API discovery
- Type-safe parameters

**Cons**:
- Slightly slower than direct calls (negligible for most apps)
- All exported methods are exposed (use CallChecker for access control)

### Embedded Resources

Resources are compiled into the binary:

```go
//go:embed client/*
var clientResources embed.FS
```

**Benefits**:
- Single binary deployment
- No asset server needed
- Cache resources in memory with minification

**Development Override**:
Set `MODULENAME_DEV=/path` environment variable to serve from filesystem instead.

### SSE vs WebSockets

go-webglue uses Server-Sent Events (SSE) for server-to-client communication:

**Why SSE?**
- Simpler than WebSockets (unidirectional)
- Automatic reconnection built-in
- Works over HTTP (no upgrade needed)
- Better for pub/sub patterns

**When to use WebSockets instead?**
- Need bidirectional streaming
- Binary data transfer
- Lower latency requirements

### jQuery-Based UI

The framework includes jQuery for DOM manipulation:

**Why jQuery?**
- Familiar API for many developers
- No build step required
- Lightweight for small apps
- Good enough for admin UIs and dashboards

**Alternatives**:
You can use React/Vue/Svelte by:
1. Not using the default `index.html`
2. Providing custom HTML with your framework
3. Using `/api/*` endpoints directly

## Performance Considerations

### Memory Usage

- Embedded resources are loaded once at startup
- Minified resources cached in memory
- Each SSE connection holds one goroutine

### Caching Strategy

**Production Mode**:
- All resources minified and cached at startup
- No disk I/O during requests
- Serve index.html for unknown paths

**Development Mode**:
- Files read from disk on each request
- No minification
- Instant updates without rebuild

### Scalability

**Single Instance**:
- Handles thousands of concurrent SSE connections
- API calls are stateless (except your API struct state)

**Multiple Instances**:
- Need external pub/sub for SSE (Redis, etc.)
- Stateless API works fine behind load balancer

## Security Considerations

### API Exposure

All exported methods are exposed by default:
- Use `CallChecker` interface for authentication
- Validate inputs in your methods
- Return errors for unauthorized access

### XSS Protection

- Client-side HTML is user-controlled (your JS)
- Be careful with user input in jQuery `.html()`
- Use `.text()` for untrusted content

### CORS

No CORS headers by default:
- Add CORS middleware if needed
- Or use a reverse proxy

## Extension Points

### Custom Index HTML

Replace the default template:

```go
webglue.Options{
    IndexHtml: myCustomHTML, // Must include {WEBGLUE} placeholder
}
```

### CallChecker Interface

Inject custom parameters into API calls:

```go
func (api *MyApi) CheckCall(req *http.Request, funcName string) ([]any, error) {
    // Authentication, rate limiting, logging, etc.
    return []any{customParam}, nil
}
```

### Multiple Modules

Organize large apps:

```go
webglue.Options{
    Modules: []*webglue.Module{
        userModule,
        productModule,
        adminModule,
    },
}
```

Each module has its own namespace, resources, and events.

## What's Next?

- [API Reference](api-reference.md) - Detailed API documentation
- [Frontend Guide](frontend-guide.md) - Client-side development
- [Examples](examples.md) - Real-world patterns
