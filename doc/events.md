# Real-time Events

Learn how to stream data from server to clients using Server-Sent Events (SSE).

## Overview

go-webglue uses Server-Sent Events (SSE) for real-time server-to-client communication. Events are:

- **Unidirectional**: Server → Client only
- **Automatic Reconnection**: Built into SSE protocol
- **Simple HTTP**: No WebSocket upgrade needed
- **Multiple Parameters**: Send any JSON-serializable data

## Creating Events

### Define Events

```go
package main

import webglue "github.com/burgrp/go-webglue/pkg"

// Create events
var (
    userCreatedEvent = webglue.NewEvent("userCreated")
    dataUpdateEvent  = webglue.NewEvent("dataUpdated")
    progressEvent    = webglue.NewEvent("progress")
)
```

### Register with Module

```go
module := &webglue.Module{
    Name: "mymodule",
    Events: []*webglue.Event{
        userCreatedEvent,
        dataUpdateEvent,
        progressEvent,
    },
    Api: &MyApi{},
}
```

## Emitting Events

### Basic Emission

```go
// No parameters
event.Emit()

// Single parameter
event.Emit("Hello")

// Multiple parameters
event.Emit(userId, userName, userEmail)

// Complex data
event.Emit(User{
    ID:    42,
    Name:  "John",
    Email: "john@example.com",
})
```

### From API Methods

```go
type MyApi struct{}

func (api *MyApi) CreateUser(name, email string) (int, error) {
    user := &User{Name: name, Email: email}
    id := saveUser(user)

    // Emit event to all connected clients
    userCreatedEvent.Emit(id, name, email)

    return id, nil
}
```

### From Background Goroutines

```go
func main() {
    // ... setup handler ...

    // Background ticker
    go func() {
        ticker := time.NewTicker(1 * time.Second)
        for t := range ticker.C {
            tickEvent.Emit(t.Unix())
        }
    }()

    http.ListenAndServe(":8080", handler)
}
```

### From External Triggers

```go
// Database change notification
func watchDatabase(event *webglue.Event) {
    listener := pq.NewListener(dbURL, ...)
    listener.Listen("table_changes")

    for notification := range listener.Notify {
        event.Emit(notification.Extra)
    }
}

// File system watcher
func watchFiles(event *webglue.Event) {
    watcher, _ := fsnotify.NewWatcher()
    for evt := range watcher.Events {
        event.Emit(evt.Name, evt.Op.String())
    }
}
```

## Receiving Events

### Client-Side Handlers

JavaScript event handlers are automatically generated based on module and event names.

**Pattern**: `on{ModuleName}{EventName}`

```javascript
// Event "userCreated" in module "mymodule"
DIV().onMymoduleUserCreated((el, userId, userName, userEmail) => {
    el.append(
        DIV().text(`New user: ${userName} (${userEmail})`)
    );
})
```

### Event Parameters

Parameters are passed to the handler in the order they were emitted:

```go
// Go
event.Emit(42, "John", "john@example.com")
```

```javascript
// JavaScript
.onMymoduleEvent((el, id, name, email) => {
    console.log(id, name, email); // 42, "John", "john@example.com"
})
```

### Multiple Handlers

Multiple elements can listen to the same event:

```javascript
// User list updates when user is created
DIV("user-list").onMymoduleUserCreated((el, id, name, email) => {
    el.append(DIV().text(name));
})

// Counter updates when user is created
DIV("user-count").onMymoduleUserCreated((el) => {
    asy(async () => {
        let count = await api.mymodule.getUserCount();
        el.text(`Total: ${count}`);
    });
})

// Notification shows when user is created
DIV("notifications").onMymoduleUserCreated((el, id, name) => {
    let notification = DIV("notification")
        .text(`${name} joined!`)
        .fadeOut(3000, function() { $(this).remove(); });
    el.append(notification);
})
```

### Event Bubbling

Events bubble up the DOM tree. Only the element itself receives the event, not its children:

```javascript
DIV("container").onMymoduleEvent((el) => {
    // Only called if event triggered on container itself
    // Not called for child elements
})
```

## Use Cases

### Live Notifications

```go
// Server
notificationEvent := webglue.NewEvent("notification")

func (api *Api) SendNotification(userId int, message string) error {
    notificationEvent.Emit(userId, message, time.Now())
    return nil
}
```

```javascript
// Client
DIV("notifications").onAppNotification((el, userId, message, timestamp) => {
    let notification = DIV("notification", [
        DIV("message").text(message),
        DIV("time").text(new Date(timestamp).toLocaleTimeString())
    ]).fadeOut(5000, function() { $(this).remove(); });

    el.prepend(notification);
})
```

### Progress Updates

```go
// Server
progressEvent := webglue.NewEvent("progress")

func (api *Api) ProcessLargeFile(fileId int) error {
    file := loadFile(fileId)
    total := len(file.Data)

    for i, chunk := range file.Chunks {
        processChunk(chunk)
        percent := (i + 1) * 100 / total
        progressEvent.Emit(fileId, percent)
    }

    return nil
}
```

```javascript
// Client
let progressBar;

DIV([
    DIV("progress-bar", el => {
        progressBar = el;
        progressBar.css("width", "0%");
    })
]).onAppProgress((el, fileId, percent) => {
    progressBar.css("width", percent + "%");
    if (percent >= 100) {
        setTimeout(() => el.fadeOut(), 1000);
    }
})
```

### Real-time Dashboard

```go
// Server
statsEvent := webglue.NewEvent("stats")

func updateStats() {
    ticker := time.NewTicker(5 * time.Second)
    for range ticker.C {
        stats := calculateStats()
        statsEvent.Emit(stats.Users, stats.Orders, stats.Revenue)
    }
}
```

```javascript
// Client
export default {
    async render() {
        let usersEl, ordersEl, revenueEl;

        const updateStats = (el, users, orders, revenue) => {
            usersEl.text(users);
            ordersEl.text(orders);
            revenueEl.text(`$${revenue}`);
        };

        return DIV("dashboard")
            .onAppStats(updateStats)
            .append([
                DIV("stat", [
                    DIV("label").text("Users"),
                    DIV("value", el => usersEl = el)
                ]),
                DIV("stat", [
                    DIV("label").text("Orders"),
                    DIV("value", el => ordersEl = el)
                ]),
                DIV("stat", [
                    DIV("label").text("Revenue"),
                    DIV("value", el => revenueEl = el)
                ])
            ]);
    }
}
```

### Live Chat

```go
// Server
messageEvent := webglue.NewEvent("message")

type ChatApi struct {
    messages []Message
}

func (api *ChatApi) SendMessage(userName, text string) error {
    msg := Message{
        User: userName,
        Text: text,
        Time: time.Now(),
    }
    api.messages = append(api.messages, msg)

    // Broadcast to all clients
    messageEvent.Emit(msg.User, msg.Text, msg.Time.Unix())

    return nil
}
```

```javascript
// Client
export default {
    async render() {
        let messagesDiv, inputEl;

        return DIV([
            DIV("messages", el => {
                messagesDiv = el;
                el.onChatMessage((container, user, text, timestamp) => {
                    container.append(
                        DIV("message", [
                            DIV("user").text(user),
                            DIV("text").text(text),
                            DIV("time").text(new Date(timestamp * 1000).toLocaleTimeString())
                        ])
                    );
                    // Auto-scroll
                    container.scrollTop(container[0].scrollHeight);
                });
            }),

            DIV("input-area", [
                TEXT(el => inputEl = el),
                BUTTON().text("Send").click(() => {
                    asy(async () => {
                        await api.chat.sendMessage("YourName", inputEl.val());
                        inputEl.val("");
                    });
                })
            ])
        ]);
    }
}
```

### Live Data Feed

```go
// Server - IoT device data
sensorEvent := webglue.NewEvent("sensorData")

func streamSensorData() {
    for data := range sensorChannel {
        sensorEvent.Emit(
            data.DeviceID,
            data.Temperature,
            data.Humidity,
            data.Timestamp,
        )
    }
}
```

```javascript
// Client - Live graph
DIV("sensor-display").onIotSensorData((el, deviceId, temp, humidity, time) => {
    // Update chart/graph with new data point
    updateChart(deviceId, temp, humidity, time);

    // Show latest reading
    el.find(`[data-device="${deviceId}"]`)
      .find(".temp").text(temp + "°C")
      .find(".humidity").text(humidity + "%");
})
```

## Advanced Patterns

### Event Filtering

Only handle events for specific items:

```javascript
// Server emits: event.Emit(orderId, status)
DIV({ "data-order-id": "123" })
    .onOrdersStatusChanged((el, orderId, status) => {
        if (orderId === el.data("order-id")) {
            el.find(".status").text(status);
        }
    })
```

### Batching Events

```go
// Server
type EventBatcher struct {
    event   *webglue.Event
    items   []Item
    mu      sync.Mutex
    ticker  *time.Ticker
}

func NewEventBatcher(event *webglue.Event) *EventBatcher {
    b := &EventBatcher{
        event:  event,
        ticker: time.NewTicker(1 * time.Second),
    }
    go b.flush()
    return b
}

func (b *EventBatcher) Add(item Item) {
    b.mu.Lock()
    b.items = append(b.items, item)
    b.mu.Unlock()
}

func (b *EventBatcher) flush() {
    for range b.ticker.C {
        b.mu.Lock()
        if len(b.items) > 0 {
            b.event.Emit(b.items)
            b.items = nil
        }
        b.mu.Unlock()
    }
}
```

### Targeted Events

Send events only to specific users (requires external pub/sub):

```go
// This requires Redis or similar for multi-instance setups
type TargetedEventSystem struct {
    events map[string]*webglue.Event
    pubsub *redis.PubSub
}

func (t *TargetedEventSystem) EmitToUser(userId string, data any) {
    // In single-instance app, could maintain user connections
    // In multi-instance, use Redis pub/sub
}
```

## Built-in Tick Event

go-webglue provides a built-in tick event that fires every second:

```javascript
DIV().onWebglueTick((el) => {
    // Called every second
    el.text(new Date().toLocaleTimeString());
})

// Or using raw event name
DIV().on("webglue.tick", (e) => {
    $(e.target).text(new Date().toLocaleTimeString());
})
```

## Debugging Events

### Server-Side

```go
func (api *Api) SendMessage(text string) error {
    fmt.Printf("Emitting message event: %s\n", text)
    messageEvent.Emit(text)
    return nil
}
```

### Client-Side

```javascript
// Log all events
$("body").on("*", (e) => {
    if (e.type.startsWith("webglue-")) {
        console.log("Event:", e.type, e);
    }
});

// Monitor SSE connection
// Open DevTools → Network → events → EventStream tab
```

### Connection Status

```javascript
// The EventSource is managed internally but you can monitor it
// Check browser console for connection messages:
// "Event source closed, reopening." = connection lost
```

## SSE vs WebSockets

### When to use SSE (go-webglue default):
- Server-to-client updates only
- Simple pub/sub patterns
- Automatic reconnection
- Works over HTTP/2

### When to consider WebSockets:
- Need client-to-server streaming
- Binary data transfer
- Very low latency requirements
- Custom protocol needed

For most dashboards, notifications, and live updates, SSE (go-webglue's approach) is simpler and sufficient.

## Performance Considerations

### Connection Limits

- Browsers limit SSE connections (typically 6 per domain)
- All events share one connection in go-webglue (no limit)
- Each client maintains one persistent connection

### Scalability

**Single Instance**: Works great for thousands of concurrent connections

**Multiple Instances**: Each instance has its own SSE connections. To broadcast across instances:

```go
// Use Redis pub/sub
func broadcastEvent(event *webglue.Event, data any) {
    // Publish to Redis
    redisClient.Publish("events", data)

    // Also emit locally
    event.Emit(data)
}

// Subscribe to Redis
func subscribeEvents(event *webglue.Event) {
    pubsub := redisClient.Subscribe("events")
    for msg := range pubsub.Channel() {
        event.Emit(msg.Payload)
    }
}
```

### Event Frequency

High-frequency events (>10 per second):
- Consider batching on server
- Throttle on client
- Use progressive updates

```javascript
let lastUpdate = 0;
DIV().onHighFreqEvent((el, data) => {
    let now = Date.now();
    if (now - lastUpdate > 100) { // Max 10 updates/sec
        updateDisplay(data);
        lastUpdate = now;
    }
})
```

## Error Handling

### Connection Errors

SSE automatically reconnects on failure. Monitor in console:

```javascript
// Automatic reconnection happens internally
// Check browser console for connection status
```

### Event Handler Errors

```javascript
DIV().onMymoduleEvent((el, data) => {
    try {
        // Process data
        processData(data);
    } catch (err) {
        console.error("Event handler error:", err);
    }
})
```

## Next Steps

- [Frontend Guide](frontend-guide.md) - Build interactive UIs
- [Authentication](authentication.md) - Secure events
- [Deployment](deployment.md) - Production setup
- [Examples](examples.md) - Complete applications
