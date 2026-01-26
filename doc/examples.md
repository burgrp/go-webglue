# Examples

Complete examples of go-webglue applications.

## Table of Contents

- [Todo List](#todo-list)
- [Real-time Chat](#real-time-chat)
- [Dashboard with Live Stats](#dashboard-with-live-stats)
- [File Upload Manager](#file-upload-manager)
- [User Authentication System](#user-authentication-system)

## Todo List

A simple todo list application demonstrating CRUD operations.

### Server (`main.go`)

```go
package main

import (
    "embed"
    "errors"
    "net/http"
    "sync"
    webglue "github.com/burgrp/go-webglue/pkg"
)

//go:embed client/*
var clientResources embed.FS

type Todo struct {
    ID        int    `json:"id"`
    Text      string `json:"text"`
    Completed bool   `json:"completed"`
}

type TodoApi struct {
    todos   []Todo
    nextID  int
    mu      sync.RWMutex
    updated *webglue.Event
}

func (api *TodoApi) List() []Todo {
    api.mu.RLock()
    defer api.mu.RUnlock()
    return api.todos
}

func (api *TodoApi) Add(text string) (Todo, error) {
    if text == "" {
        return Todo{}, errors.New("text cannot be empty")
    }

    api.mu.Lock()
    defer api.mu.Unlock()

    todo := Todo{
        ID:   api.nextID,
        Text: text,
    }
    api.nextID++
    api.todos = append(api.todos, todo)

    api.updated.Emit("added", todo)

    return todo, nil
}

func (api *TodoApi) Toggle(id int) error {
    api.mu.Lock()
    defer api.mu.Unlock()

    for i := range api.todos {
        if api.todos[i].ID == id {
            api.todos[i].Completed = !api.todos[i].Completed
            api.updated.Emit("toggled", api.todos[i])
            return nil
        }
    }

    return errors.New("todo not found")
}

func (api *TodoApi) Delete(id int) error {
    api.mu.Lock()
    defer api.mu.Unlock()

    for i, todo := range api.todos {
        if todo.ID == id {
            api.todos = append(api.todos[:i], api.todos[i+1:]...)
            api.updated.Emit("deleted", id)
            return nil
        }
    }

    return errors.New("todo not found")
}

func main() {
    updatedEvent := webglue.NewEvent("updated")

    handler, _ := webglue.NewHandler(webglue.Options{
        Modules: []*webglue.Module{{
            Name:      "todo",
            Resources: &clientResources,
            Events:    []*webglue.Event{updatedEvent},
            Api: &TodoApi{
                updated: updatedEvent,
            },
        }},
    })

    http.ListenAndServe(":8080", handler)
}
```

### Client (`client/home.page.js`)

```javascript
import { api, asy, tags } from "webglue";

let { DIV, H1, INPUT, BUTTON, CHECKBOX } = tags;

export default {
    title: "Todo List",

    async render() {
        let inputEl, listEl;

        const renderList = async () => {
            let todos = await api.todo.list();
            listEl.empty();

            todos.forEach(todo => {
                listEl.append(
                    DIV("todo-item", [
                        CHECKBOX(el => {
                            el.prop("checked", todo.completed);
                            el.change(() => {
                                asy(async () => {
                                    await api.todo.toggle(todo.id);
                                });
                            });
                        }),
                        DIV("todo-text", el => {
                            el.text(todo.text);
                            if (todo.completed) {
                                el.css("text-decoration", "line-through");
                            }
                        }),
                        BUTTON().text("Delete").click(() => {
                            asy(async () => {
                                await api.todo.delete(todo.id);
                            });
                        })
                    ])
                );
            });
        };

        const addTodo = () => {
            asy(async () => {
                let text = inputEl.val();
                if (text) {
                    await api.todo.add(text);
                    inputEl.val("");
                }
            });
        };

        return DIV("container", [
            H1().text("Todo List"),

            DIV("input-area", [
                INPUT({ type: "text", placeholder: "New todo" }, el => {
                    inputEl = el;
                    el.keypress(e => {
                        if (e.key === "Enter") addTodo();
                    });
                }),
                BUTTON().text("Add").click(addTodo)
            ]),

            DIV("todo-list", el => {
                listEl = el;
                listEl.onTodoUpdated(() => renderList());
                renderList();
            })
        ]);
    }
}
```

## Real-time Chat

Chat application with live message updates.

### Server

```go
package main

import (
    "embed"
    "net/http"
    "sync"
    "time"
    webglue "github.com/burgrp/go-webglue/pkg"
)

//go:embed client/*
var clientResources embed.FS

type Message struct {
    User      string `json:"user"`
    Text      string `json:"text"`
    Timestamp int64  `json:"timestamp"`
}

type ChatApi struct {
    messages      []Message
    mu            sync.RWMutex
    messageEvent  *webglue.Event
}

func (api *ChatApi) GetMessages() []Message {
    api.mu.RLock()
    defer api.mu.RUnlock()
    return api.messages
}

func (api *ChatApi) Send(user, text string) error {
    msg := Message{
        User:      user,
        Text:      text,
        Timestamp: time.Now().Unix(),
    }

    api.mu.Lock()
    api.messages = append(api.messages, msg)
    api.mu.Unlock()

    api.messageEvent.Emit(msg)

    return nil
}

func main() {
    messageEvent := webglue.NewEvent("message")

    handler, _ := webglue.NewHandler(webglue.Options{
        Modules: []*webglue.Module{{
            Name:      "chat",
            Resources: &clientResources,
            Events:    []*webglue.Event{messageEvent},
            Api: &ChatApi{
                messageEvent: messageEvent,
            },
        }},
    })

    http.ListenAndServe(":8080", handler)
}
```

### Client

```javascript
import { api, asy, tags } from "webglue";

let { DIV, H1, TEXT, BUTTON } = tags;

export default {
    title: "Chat",

    async render() {
        let usernameEl, messageEl, messagesEl;
        let username = localStorage.getItem("chat-username") || "Anonymous";

        const renderMessages = async () => {
            let messages = await api.chat.getMessages();
            messagesEl.empty();

            messages.forEach(msg => {
                messagesEl.append(
                    DIV("message", [
                        DIV("message-user").text(msg.user),
                        DIV("message-text").text(msg.text),
                        DIV("message-time").text(
                            new Date(msg.timestamp * 1000).toLocaleTimeString()
                        )
                    ])
                );
            });

            // Auto-scroll
            messagesEl.scrollTop(messagesEl[0].scrollHeight);
        };

        const sendMessage = () => {
            asy(async () => {
                let text = messageEl.val();
                if (text) {
                    await api.chat.send(username, text);
                    messageEl.val("");
                }
            });
        };

        return DIV("container", [
            H1().text("Chat Room"),

            DIV("username-area", [
                TEXT(el => {
                    usernameEl = el;
                    el.val(username);
                    el.attr("placeholder", "Username");
                    el.change(() => {
                        username = usernameEl.val();
                        localStorage.setItem("chat-username", username);
                    });
                })
            ]),

            DIV("messages", el => {
                messagesEl = el;
                el.onChatMessage((container, msg) => {
                    container.append(
                        DIV("message", [
                            DIV("message-user").text(msg.user),
                            DIV("message-text").text(msg.text),
                            DIV("message-time").text(
                                new Date(msg.timestamp * 1000).toLocaleTimeString()
                            )
                        ])
                    );
                    container.scrollTop(container[0].scrollHeight);
                });
                renderMessages();
            }),

            DIV("input-area", [
                TEXT(el => {
                    messageEl = el;
                    el.attr("placeholder", "Type a message");
                    el.keypress(e => {
                        if (e.key === "Enter") sendMessage();
                    });
                }),
                BUTTON().text("Send").click(sendMessage)
            ])
        ]);
    }
}
```

## Dashboard with Live Stats

Real-time statistics dashboard.

### Server

```go
package main

import (
    "embed"
    "math/rand"
    "net/http"
    "time"
    webglue "github.com/burgrp/go-webglue/pkg"
)

//go:embed client/*
var clientResources embed.FS

type Stats struct {
    Users    int     `json:"users"`
    Revenue  float64 `json:"revenue"`
    Orders   int     `json:"orders"`
}

type DashboardApi struct {
    stats       Stats
    statsEvent  *webglue.Event
}

func (api *DashboardApi) GetStats() Stats {
    return api.stats
}

func (api *DashboardApi) updateStats() {
    ticker := time.NewTicker(2 * time.Second)
    for range ticker.C {
        // Simulate changing stats
        api.stats.Users += rand.Intn(3)
        api.stats.Orders += rand.Intn(5)
        api.stats.Revenue += float64(rand.Intn(1000))

        api.statsEvent.Emit(api.stats)
    }
}

func main() {
    statsEvent := webglue.NewEvent("stats")

    dashApi := &DashboardApi{
        stats: Stats{
            Users:   100,
            Revenue: 5000,
            Orders:  50,
        },
        statsEvent: statsEvent,
    }

    go dashApi.updateStats()

    handler, _ := webglue.NewHandler(webglue.Options{
        Modules: []*webglue.Module{{
            Name:      "dashboard",
            Resources: &clientResources,
            Events:    []*webglue.Event{statsEvent},
            Api:       dashApi,
        }},
    })

    http.ListenAndServe(":8080", handler)
}
```

### Client

```javascript
import { api, tags } from "webglue";

let { DIV, H1, H2 } = tags;

export default {
    title: "Dashboard",

    async render() {
        let usersEl, revenueEl, ordersEl;

        const updateStats = (el, stats) => {
            usersEl.text(stats.users);
            revenueEl.text("$" + stats.revenue.toFixed(2));
            ordersEl.text(stats.orders);
        };

        let initialStats = await api.dashboard.getStats();

        return DIV("dashboard")
            .onDashboardStats(updateStats)
            .append([
                H1().text("Live Dashboard"),

                DIV("stats", [
                    DIV("stat-card", [
                        H2().text("Users"),
                        DIV("stat-value", el => {
                            usersEl = el;
                            el.text(initialStats.users);
                        })
                    ]),
                    DIV("stat-card", [
                        H2().text("Revenue"),
                        DIV("stat-value", el => {
                            revenueEl = el;
                            el.text("$" + initialStats.revenue.toFixed(2));
                        })
                    ]),
                    DIV("stat-card", [
                        H2().text("Orders"),
                        DIV("stat-value", el => {
                            ordersEl = el;
                            el.text(initialStats.orders);
                        })
                    ])
                ])
            ]);
    }
}
```

## File Upload Manager

File upload with progress tracking.

### Server

```go
package main

import (
    "embed"
    "io"
    "net/http"
    "os"
    "path/filepath"
    webglue "github.com/burgrp/go-webglue/pkg"
)

//go:embed client/*
var clientResources embed.FS

type FileInfo struct {
    Name string `json:"name"`
    Size int64  `json:"size"`
}

type FilesApi struct {
    uploadDir     string
    progressEvent *webglue.Event
}

func (api *FilesApi) List() ([]FileInfo, error) {
    files, err := os.ReadDir(api.uploadDir)
    if err != nil {
        return nil, err
    }

    result := []FileInfo{}
    for _, file := range files {
        info, _ := file.Info()
        result = append(result, FileInfo{
            Name: file.Name(),
            Size: info.Size(),
        })
    }

    return result, nil
}

func (api *FilesApi) Upload(req *http.Request, filename string) error {
    // This is simplified - in real app, use multipart form
    file, err := os.Create(filepath.Join(api.uploadDir, filename))
    if err != nil {
        return err
    }
    defer file.Close()

    // Copy with progress
    buffer := make([]byte, 32*1024)
    total := 0

    for {
        n, err := req.Body.Read(buffer)
        if n > 0 {
            file.Write(buffer[:n])
            total += n
            api.progressEvent.Emit(filename, total)
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
    }

    return nil
}

func main() {
    os.MkdirAll("./uploads", 0755)

    progressEvent := webglue.NewEvent("progress")

    handler, _ := webglue.NewHandler(webglue.Options{
        Modules: []*webglue.Module{{
            Name:      "files",
            Resources: &clientResources,
            Events:    []*webglue.Event{progressEvent},
            Api: &FilesApi{
                uploadDir:     "./uploads",
                progressEvent: progressEvent,
            },
        }},
    })

    http.ListenAndServe(":8080", handler)
}
```

## User Authentication System

Complete authentication with login, registration, and protected routes.

### Server

```go
package main

import (
    "context"
    "embed"
    "errors"
    "net/http"
    "time"
    "github.com/golang-jwt/jwt/v5"
    "golang.org/x/crypto/bcrypt"
    webglue "github.com/burgrp/go-webglue/pkg"
)

//go:embed client/*
var clientResources embed.FS

var secretKey = []byte("your-secret-key")

type User struct {
    ID       int    `json:"id"`
    Email    string `json:"email"`
    Username string `json:"username"`
    Password string `json:"-"`
}

type AuthApi struct {
    users  map[string]User // email -> user
    nextID int
}

func (api *AuthApi) CheckCall(req *http.Request, funcName string) ([]any, error) {
    // Public endpoints
    if funcName == "Login" || funcName == "Register" {
        return nil, nil
    }

    // Validate token
    tokenString := req.Header.Get("Authorization")
    if tokenString == "" {
        return nil, errors.New("unauthorized")
    }

    token, err := jwt.Parse(tokenString[7:], func(token *jwt.Token) (any, error) {
        return secretKey, nil
    })

    if err != nil || !token.Valid {
        return nil, errors.New("invalid token")
    }

    claims := token.Claims.(jwt.MapClaims)
    email := claims["email"].(string)

    user, exists := api.users[email]
    if !exists {
        return nil, errors.New("user not found")
    }

    return []any{&user}, nil
}

func (api *AuthApi) Register(email, username, password string) error {
    if _, exists := api.users[email]; exists {
        return errors.New("user already exists")
    }

    hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), 14)

    api.users[email] = User{
        ID:       api.nextID,
        Email:    email,
        Username: username,
        Password: string(hashedPassword),
    }
    api.nextID++

    return nil
}

func (api *AuthApi) Login(email, password string) (string, error) {
    user, exists := api.users[email]
    if !exists {
        return "", errors.New("invalid credentials")
    }

    err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
    if err != nil {
        return "", errors.New("invalid credentials")
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "email": email,
        "exp":   time.Now().Add(24 * time.Hour).Unix(),
    })

    tokenString, _ := token.SignedString(secretKey)
    return tokenString, nil
}

func (api *AuthApi) GetCurrentUser(ctx context.Context, user *User) User {
    return *user
}

func main() {
    handler, _ := webglue.NewHandler(webglue.Options{
        Modules: []*webglue.Module{{
            Name:      "auth",
            Resources: &clientResources,
            Api: &AuthApi{
                users: make(map[string]User),
            },
        }},
    })

    http.ListenAndServe(":8080", handler)
}
```

### Client Pages

**`client/login.page.js`**:
```javascript
import { api, asy, goto, tags } from "webglue";

let { DIV, H1, TEXT, PASSWORD, BUTTON, AHREF } = tags;

export default {
    title: "Login",

    async render() {
        let emailEl, passwordEl, errorEl;

        const handleLogin = () => {
            asy(async () => {
                errorEl.empty();
                try {
                    let token = await api.auth.login(
                        emailEl.val(),
                        passwordEl.val()
                    );

                    localStorage.setItem("webglue.headers.Authorization", "Bearer " + token);
                    goto("/");
                } catch (err) {
                    errorEl.text(err.message);
                }
            });
        };

        return DIV("container", [
            H1().text("Login"),
            DIV("error", el => errorEl = el),
            TEXT(el => {
                emailEl = el;
                el.attr("placeholder", "Email");
            }),
            PASSWORD(el => {
                passwordEl = el;
                el.attr("placeholder", "Password");
                el.keypress(e => e.key === "Enter" && handleLogin());
            }),
            BUTTON().text("Login").click(handleLogin),
            AHREF({ href: "/register" }).text("Register")
        ]);
    }
}
```

**`client/home.page.js`**:
```javascript
import { api, asy, goto, tags } from "webglue";

let { DIV, H1, BUTTON } = tags;

export default {
    title: "Home",

    async check() {
        try {
            await api.auth.getCurrentUser();
        } catch (err) {
            return "/login";
        }
    },

    async render() {
        let user = await api.auth.getCurrentUser();

        return DIV("container", [
            H1().text(`Welcome, ${user.username}!`),
            DIV().text(`Email: ${user.email}`),
            BUTTON().text("Logout").click(() => {
                localStorage.removeItem("webglue.headers.Authorization");
                goto("/login");
            })
        ]);
    }
}
```

## More Examples

The repository's `test/` directory contains additional examples demonstrating:
- Basic API calls
- Event handling
- Complex parameters
- Error handling
- State management

## Next Steps

- Modify these examples for your needs
- Combine patterns from multiple examples
- Check [Getting Started](getting-started.md) for basics
- See [API Reference](api-reference.md) for all options
