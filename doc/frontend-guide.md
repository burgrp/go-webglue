# Frontend Development Guide

Master client-side development with go-webglue.

## Page Structure

### Basic Page Template

```javascript
import { api, asy, tags, goto } from "webglue";

let { DIV, BUTTON, H1 } = tags;

export default {
    title: "My Page",

    async render(url, params) {
        // Your UI code here
        return [
            H1().text("Hello World"),
            DIV().text("Content")
        ];
    }
}
```

### Page Lifecycle

1. User navigates to `/mypage?foo=bar`
2. webglue imports `mypage.page.js`
3. Calls `page.check(url, params)` if defined
4. If no redirect, calls `page.render(url, params)`
5. Returned elements replace `<body>` content

### URL Parameters

```javascript
export default {
    async render(url, params) {
        // URL: /users?id=42&tab=profile
        console.log(params.id);    // "42"
        console.log(params.tab);   // "profile"
    }
}
```

### Pre-render Checks

Redirect users before rendering:

```javascript
export default {
    async check(url, params) {
        let user = await api.auth.getCurrentUser();
        if (!user) {
            return "/login"; // Redirect to login
        }
        // Return nothing to continue rendering
    },

    async render(url, params) {
        // Only reached if user is logged in
    }
}
```

### Custom Error Handling

```javascript
export default {
    async render() {
        let errorDiv;

        return [
            DIV("errors", el => errorDiv = el),
            // ... rest of page
        ];
    },

    error(e) {
        // Custom error display
        errorDiv.append(
            DIV("error-message")
                .text(e.message)
                .fadeOut(3000, function() { $(this).remove(); })
        );
    }
}
```

## Tag Factories

### Basic Usage

```javascript
let { DIV, SPAN, BUTTON } = tags;

DIV()                    // <div></div>
DIV("my-class")         // <div class="my-class"></div>
DIV("class1", "class2") // <div class="class1 class2"></div>
```

### Setting Properties

```javascript
DIV({
    id: "myDiv",
    className: "container",
    style: "color: red"
})

IMG({ src: "/logo.png", alt: "Logo" })

INPUT({ type: "text", placeholder: "Enter name" })
```

### Adding Content

```javascript
// Text content
DIV().text("Hello")

// HTML content (use with caution)
DIV().html("<strong>Bold</strong>")

// Append children
DIV([
    SPAN().text("Child 1"),
    SPAN().text("Child 2")
])
```

### Combining Arguments

Arguments are processed left-to-right:

```javascript
DIV(
    "container",              // Add class
    { id: "main" },          // Set properties
    [                        // Add children
        SPAN().text("Hello")
    ],
    el => el.fadeIn()        // Callback
)
```

### Callbacks

Callbacks receive the jQuery element and can return values:

```javascript
let myDiv;
DIV(el => {
    myDiv = el;              // Save reference
    el.css("color", "blue"); // Modify element
    return "Text content";   // Return value is added
})
```

**Async Callbacks**:
```javascript
DIV(async el => {
    let data = await api.mymodule.getData();
    return SPAN().text(data);
})
```

### Form Elements

```javascript
let { TEXT, PASSWORD, NUMBER, CHECKBOX, SELECT, OPTION } = tags;

// Text input
TEXT(el => {
    el.val("Default value");
    el.attr("placeholder", "Enter text");
})

// Number input
NUMBER({ min: 0, max: 100, value: 50 })

// Checkbox
CHECKBOX(el => {
    el.prop("checked", true);
    el.change(() => console.log(el.is(":checked")));
})

// Select dropdown
SELECT([
    OPTION({ value: "1" }).text("Option 1"),
    OPTION({ value: "2" }).text("Option 2")
], el => {
    el.change(() => console.log(el.val()));
})
```

### Buttons

```javascript
BUTTON().text("Click Me").click(() => {
    alert("Clicked!");
})

// With async action
BUTTON().text("Load Data").click(() => {
    asy(async () => {
        let data = await api.mymodule.getData();
        console.log(data);
    });
})
```

### Links

```javascript
// Internal link (SPA navigation)
AHREF({ href: "/users" }).text("Users")

// External link
AHREF({ href: "https://example.com", target: "_blank" })
    .text("External")

// Programmatic navigation
BUTTON().text("Go to Profile").click(() => {
    goto("/profile?id=42");
})
```

## Calling APIs

### Simple Calls

```javascript
// No parameters
let count = await api.mymodule.getCount();

// Single parameter
let user = await api.mymodule.getUser(42);

// Multiple parameters
let result = await api.mymodule.calculate(10, 20, "add");
```

### Complex Parameters

```javascript
// Object parameter
let user = await api.mymodule.createUser({
    firstName: "John",
    lastName: "Doe",
    age: 30,
    email: "john@example.com"
});

// Array parameter
let stats = await api.mymodule.calculateStats([1, 2, 3, 4, 5]);

// Mixed parameters
let result = await api.mymodule.processOrder(
    42,                    // Order ID
    { rush: true },       // Options
    ["item1", "item2"]    // Items
);
```

### Error Handling

```javascript
// Try-catch
try {
    let result = await api.mymodule.dangerousOperation();
} catch (err) {
    console.error("Operation failed:", err.message);
    alert(err.message);
}

// Using asy() helper
BUTTON().click(() => {
    asy(async () => {
        let result = await api.mymodule.dangerousOperation();
        // Errors automatically handled by page.error() or alert()
    });
});
```

### Multiple Return Values

```javascript
// Go returns (int, int, error)
let [quotient, remainder] = await api.mymodule.divMod(10, 3);
console.log(`${quotient} remainder ${remainder}`);
```

## State Management

### Component-Level State

```javascript
function createCounter() {
    let count = 0;
    let display;

    return DIV([
        DIV(el => display = el).text(count),
        BUTTON().text("+").click(() => {
            count++;
            display.text(count);
        })
    ]);
}

// Usage
export default {
    async render() {
        return [
            createCounter(),
            createCounter()  // Each has independent state
        ];
    }
}
```

### Page-Level State

```javascript
let pageState = {
    users: [],
    selectedId: null
};

export default {
    async render(url, params) {
        pageState.users = await api.mymodule.getUsers();

        return [
            renderUserList(),
            renderUserDetail()
        ];
    }
}
```

### Local Storage

```javascript
// Save state
localStorage.setItem("settings", JSON.stringify({ theme: "dark" }));

// Load state
let settings = JSON.parse(localStorage.getItem("settings") || "{}");

// Remove
localStorage.removeItem("settings");
```

### Session Storage

Same API as localStorage, but cleared when browser closes:

```javascript
sessionStorage.setItem("tempData", "value");
```

## Event Handling

### DOM Events

```javascript
BUTTON()
    .click(e => console.log("Clicked"))
    .dblclick(e => console.log("Double clicked"))
    .mouseenter(e => $(e.target).addClass("hover"))

INPUT()
    .keyup(e => console.log("Key:", e.key))
    .change(e => console.log("Changed:", $(e.target).val()))
    .focus(e => console.log("Focused"))
```

### Server Events

```javascript
// Assuming Go has: event := webglue.NewEvent("userCreated")
DIV().onUsersUserCreated((el, userId, userName) => {
    el.append(
        DIV().text(`User ${userName} created`)
    );
})
```

### Built-in Tick Event

```javascript
DIV(el => {
    let start = Date.now();
    el.onWebglueTick(() => {
        let elapsed = Math.floor((Date.now() - start) / 1000);
        el.text(`${elapsed} seconds`);
    });
})
```

## Common Patterns

### Loading Indicators

```javascript
async function loadData() {
    let container, loading;

    return DIV([
        DIV("loading", el => {
            loading = el;
            el.text("Loading...");
        }),
        DIV(el => {
            container = el;
            asy(async () => {
                let data = await api.mymodule.getData();
                loading.hide();
                container.append(renderData(data));
            });
        })
    ]);
}
```

### Forms with Validation

```javascript
function createLoginForm() {
    let emailInput, passwordInput, errorDiv;

    const handleSubmit = () => {
        asy(async () => {
            errorDiv.empty();

            let email = emailInput.val();
            let password = passwordInput.val();

            if (!email || !password) {
                errorDiv.text("All fields required");
                return;
            }

            try {
                let token = await api.auth.login(email, password);
                localStorage.setItem("webglue.headers.Authorization", `Bearer ${token}`);
                goto("/dashboard");
            } catch (err) {
                errorDiv.text(err.message);
            }
        });
    };

    return FORM([
        DIV("errors", el => errorDiv = el),
        TEXT(el => {
            emailInput = el;
            el.attr("placeholder", "Email");
            el.keypress(e => e.key === "Enter" && handleSubmit());
        }),
        PASSWORD(el => {
            passwordInput = el;
            el.attr("placeholder", "Password");
            el.keypress(e => e.key === "Enter" && handleSubmit());
        }),
        BUTTON({ type: "button" })
            .text("Login")
            .click(handleSubmit)
    ]);
}
```

### Dynamic Lists

```javascript
function renderUserList(users) {
    return DIV("user-list",
        users.map(user =>
            DIV("user-item", [
                SPAN().text(user.name),
                BUTTON()
                    .text("Delete")
                    .click(() => {
                        asy(async () => {
                            await api.mymodule.deleteUser(user.id);
                            // Refresh page or update UI
                            goto(window.location.pathname, true);
                        });
                    })
            ])
        )
    );
}
```

### Modal Dialogs

```javascript
function showModal(title, content) {
    let modal = DIV("modal-overlay", [
        DIV("modal", [
            H1().text(title),
            DIV("modal-content", content),
            BUTTON().text("Close").click(() => {
                modal.fadeOut(() => modal.remove());
            })
        ])
    ]);

    $("body").append(modal);
    modal.fadeIn();

    return modal;
}

// Usage
BUTTON().click(() => {
    showModal("Confirm", [
        DIV().text("Are you sure?"),
        BUTTON().text("Yes").click(() => {
            // Do something
        })
    ]);
})
```

### Tables

```javascript
function renderTable(data) {
    let { TABLE, TR, TH, TD } = tags;

    return TABLE([
        TR([
            TH().text("Name"),
            TH().text("Email"),
            TH().text("Actions")
        ]),
        ...data.map(row => TR([
            TD().text(row.name),
            TD().text(row.email),
            TD([
                BUTTON().text("Edit").click(() => editUser(row.id)),
                BUTTON().text("Delete").click(() => deleteUser(row.id))
            ])
        ]))
    ]);
}
```

### Tabs

```javascript
function createTabs(tabs) {
    let contentDiv;
    let activeTab = tabs[0].id;

    const showTab = (tabId) => {
        activeTab = tabId;
        let tab = tabs.find(t => t.id === tabId);
        contentDiv.empty().append(tab.render());

        // Update active state
        $(".tab-button").removeClass("active");
        $(`[data-tab="${tabId}"]`).addClass("active");
    };

    return DIV([
        DIV("tabs",
            tabs.map(tab =>
                BUTTON({ "data-tab": tab.id })
                    .addClass("tab-button")
                    .addClass(tab.id === activeTab ? "active" : "")
                    .text(tab.label)
                    .click(() => showTab(tab.id))
            )
        ),
        DIV("tab-content", el => {
            contentDiv = el;
            contentDiv.append(tabs[0].render());
        })
    ]);
}

// Usage
createTabs([
    { id: "profile", label: "Profile", render: () => DIV().text("Profile") },
    { id: "settings", label: "Settings", render: () => DIV().text("Settings") }
])
```

## Styling

### Inline Styles

```javascript
DIV().css({
    color: "red",
    fontSize: "20px",
    padding: "10px"
})
```

### CSS Classes

Create `client/styles.css`:

```css
.container {
    max-width: 800px;
    margin: 0 auto;
}

.button-primary {
    background: blue;
    color: white;
    padding: 10px 20px;
}
```

Use in JavaScript:

```javascript
DIV("container", [
    BUTTON("button-primary").text("Click Me")
])
```

### Dynamic Classes

```javascript
let isActive = true;

DIV()
    .addClass(isActive ? "active" : "inactive")
    .toggleClass("highlight")
    .removeClass("old-class")
```

## Debugging

### Console Logging

```javascript
console.log("Data:", data);
console.error("Error:", err);
console.warn("Warning:", message);
```

### Network Tab

Monitor API calls in browser DevTools:
- Look for requests to `/api/{module}/{function}`
- Check request payload (JSON array of parameters)
- Check response: `{"result": ...}` or `{"error": "..."}`

### Breakpoints

```javascript
async render() {
    debugger; // Execution pauses here
    let data = await api.mymodule.getData();
    return DIV().text(JSON.stringify(data));
}
```

## Performance Tips

### Minimize API Calls

```javascript
// Bad: Multiple calls
let user = await api.users.get(id);
let posts = await api.posts.getByUser(id);
let comments = await api.comments.getByUser(id);

// Good: Single call
let data = await api.users.getFullProfile(id); // Returns all data
```

### Lazy Loading

```javascript
DIV("user-details", el => {
    // Only load when element is visible
    asy(async () => {
        let data = await api.mymodule.getDetails();
        el.append(renderDetails(data));
    });
})
```

### Debouncing

```javascript
let timeout;
INPUT().keyup(e => {
    clearTimeout(timeout);
    timeout = setTimeout(() => {
        asy(async () => {
            let results = await api.search.query($(e.target).val());
            displayResults(results);
        });
    }, 300); // Wait 300ms after last keystroke
});
```

## Next Steps

- [Events Guide](events.md) - Real-time updates
- [Authentication](authentication.md) - Secure your app
- [Examples](examples.md) - Complete applications
