# Deployment Guide

Deploy go-webglue applications to production.

## Building for Production

### Single Binary

go-webglue applications compile to a single executable with all resources embedded:

```bash
# Build
go build -o myapp

# The binary includes:
# - Go code
# - All embedded client resources (HTML, CSS, JS)
# - No external dependencies needed
```

### Optimized Build

```bash
# Reduce binary size
go build -ldflags="-s -w" -o myapp

# Further compression with UPX (optional)
upx --best --lzma myapp
```

### Cross-Platform Builds

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o myapp-linux

# Windows
GOOS=windows GOARCH=amd64 go build -o myapp.exe

# macOS
GOOS=darwin GOARCH=amd64 go build -o myapp-mac
```

## Resource Optimization

### Automatic Minification

go-webglue automatically minifies resources in production:

```go
// No special configuration needed
handler, _ := webglue.NewHandler(options)

// Resources are automatically:
// - Minified (HTML, CSS, JS, JSON, SVG, XML)
// - Cached in memory
// - Served with proper MIME types
```

### Development vs Production

```bash
# Development: Files served from disk, no minification
MYMODULE_DEV=/path/to/client go run main.go

# Production: No env var, everything embedded and minified
go build && ./myapp
```

## Server Configuration

### Basic Server

```go
func main() {
    handler, err := webglue.NewHandler(options)
    if err != nil {
        log.Fatal(err)
    }

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Starting server on port %s\n", port)
    if err := http.ListenAndServe(":"+port, handler); err != nil {
        log.Fatal(err)
    }
}
```

### HTTPS Server

```go
func main() {
    handler, _ := webglue.NewHandler(options)

    // Load certificates
    certFile := os.Getenv("CERT_FILE")
    keyFile := os.Getenv("KEY_FILE")

    port := ":443"
    log.Printf("Starting HTTPS server on %s\n", port)

    err := http.ListenAndServeTLS(port, certFile, keyFile, handler)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Graceful Shutdown

```go
func main() {
    handler, _ := webglue.NewHandler(options)

    server := &http.Server{
        Addr:    ":8080",
        Handler: handler,
    }

    // Start server in goroutine
    go func() {
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v\n", err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")

    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        log.Fatal("Server forced to shutdown:", err)
    }

    log.Println("Server exited")
}
```

## Deployment Platforms

### Docker

**Dockerfile**:
```dockerfile
# Build stage
FROM golang:1.20-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o myapp

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/myapp .

EXPOSE 8080

CMD ["./myapp"]
```

**docker-compose.yml**:
```yaml
version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
    restart: unless-stopped
```

**Build and run**:
```bash
docker build -t myapp .
docker run -p 8080:8080 myapp
```

### Kubernetes

**deployment.yaml**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
        image: myapp:latest
        ports:
        - containerPort: 8080
        env:
        - name: PORT
          value: "8080"
        resources:
          limits:
            memory: "128Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: myapp-service
spec:
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

### Linux Server (systemd)

**Create service file** `/etc/systemd/system/myapp.service`:
```ini
[Unit]
Description=My go-webglue Application
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/myapp
ExecStart=/opt/myapp/myapp
Restart=on-failure
RestartSec=5s

# Environment variables
Environment="PORT=8080"

[Install]
WantedBy=multi-user.target
```

**Commands**:
```bash
# Deploy binary
sudo cp myapp /opt/myapp/
sudo chown www-data:www-data /opt/myapp/myapp
sudo chmod +x /opt/myapp/myapp

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable myapp
sudo systemctl start myapp

# Check status
sudo systemctl status myapp

# View logs
sudo journalctl -u myapp -f
```

### Cloud Platforms

#### Heroku

**Procfile**:
```
web: ./myapp
```

Deploy:
```bash
heroku create myapp
git push heroku main
```

#### Google Cloud Run

```bash
# Build and deploy
gcloud run deploy myapp \
  --source . \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

#### AWS Elastic Beanstalk

```bash
# Create application
eb init -p go myapp

# Deploy
eb create myapp-env
eb deploy
```

## Reverse Proxy Setup

### nginx

**Configuration**:
```nginx
server {
    listen 80;
    server_name example.com;

    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name example.com;

    ssl_certificate /etc/ssl/certs/example.com.crt;
    ssl_certificate_key /etc/ssl/private/example.com.key;

    # Proxy to go-webglue app
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # SSE requires special configuration
    location /events {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Connection '';
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 24h;
    }
}
```

### Caddy

**Caddyfile**:
```
example.com {
    reverse_proxy localhost:8080
}
```

Caddy automatically handles HTTPS with Let's Encrypt.

### Traefik

**docker-compose.yml**:
```yaml
version: '3.8'

services:
  traefik:
    image: traefik:v2.9
    command:
      - "--providers.docker=true"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.myresolver.acme.tlschallenge=true"
      - "--certificatesresolvers.myresolver.acme.email=admin@example.com"
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock

  app:
    build: .
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.myapp.rule=Host(`example.com`)"
      - "traefik.http.routers.myapp.entrypoints=websecure"
      - "traefik.http.routers.myapp.tls.certresolver=myresolver"
```

## Scaling

### Horizontal Scaling

go-webglue apps are stateless (except API struct state), making them easy to scale:

```bash
# Docker Swarm
docker service create --replicas 5 --name myapp -p 8080:8080 myapp:latest

# Kubernetes
kubectl scale deployment myapp --replicas=5
```

### SSE with Multiple Instances

SSE connections are per-instance. To broadcast events across instances:

#### Option 1: Redis Pub/Sub

```go
import "github.com/go-redis/redis/v8"

type DistributedEvents struct {
    redis  *redis.Client
    events map[string]*webglue.Event
}

func (d *DistributedEvents) Emit(eventName string, data ...any) {
    // Emit locally
    d.events[eventName].Emit(data...)

    // Publish to Redis
    payload, _ := json.Marshal(data)
    d.redis.Publish(ctx, eventName, payload)
}

func (d *DistributedEvents) Subscribe() {
    pubsub := d.redis.Subscribe(ctx, "events")

    for msg := range pubsub.Channel() {
        var data []any
        json.Unmarshal([]byte(msg.Payload), &data)

        // Emit to local connections only
        event := d.events[msg.Channel]
        event.Emit(data...)
    }
}
```

#### Option 2: Sticky Sessions

Configure load balancer for session affinity:

**nginx**:
```nginx
upstream myapp {
    ip_hash;  # Sticky sessions based on client IP
    server 10.0.0.1:8080;
    server 10.0.0.2:8080;
    server 10.0.0.3:8080;
}
```

## Database Configuration

### Connection Pooling

```go
import "database/sql"

type MyApi struct {
    db *sql.DB
}

func NewMyApi() *MyApi {
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }

    // Configure pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)

    return &MyApi{db: db}
}
```

### Environment Variables

```bash
export DATABASE_URL="postgres://user:pass@localhost/dbname?sslmode=disable"
export REDIS_URL="redis://localhost:6379"
export SECRET_KEY="your-secret-key"
```

## Monitoring

### Health Check Endpoint

```go
type HealthApi struct{}

func (api *HealthApi) Health() string {
    return "OK"
}

// Add to modules
modules := []*webglue.Module{
    {
        Name: "health",
        Api:  &HealthApi{},
    },
    // ... other modules
}
```

Access at: `http://your-app/api/health/health`

### Logging

```go
import "log/slog"

func (api *MyApi) GetUser(id int) (*User, error) {
    slog.Info("GetUser called", "userId", id)

    user, err := api.db.GetUser(id)
    if err != nil {
        slog.Error("Failed to get user", "error", err, "userId", id)
        return nil, err
    }

    return user, nil
}
```

### Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    apiCalls = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "api_calls_total",
            Help: "Total number of API calls",
        },
        []string{"module", "function"},
    )
)

func (api *MyApi) CheckCall(req *http.Request, funcName string) ([]any, error) {
    apiCalls.WithLabelValues("mymodule", funcName).Inc()
    // ... auth logic
}
```

## Security

### Environment Variables

```bash
# Never commit these!
export JWT_SECRET="your-secret-key-here"
export DATABASE_PASSWORD="secure-password"
export API_KEY="api-key"
```

Use `.env` files (with `godotenv` package):

```go
import "github.com/joho/godotenv"

func main() {
    godotenv.Load()
    secretKey := os.Getenv("JWT_SECRET")
    // ...
}
```

### CORS (if needed)

```go
func withCORS(handler http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }

        handler.ServeHTTP(w, r)
    })
}

func main() {
    handler, _ := webglue.NewHandler(options)
    http.ListenAndServe(":8080", withCORS(handler))
}
```

### Rate Limiting

See [authentication.md](authentication.md#rate-limiting) for implementation.

## Performance Tuning

### Go Runtime

```bash
# Set GOMAXPROCS (usually auto-detected)
export GOMAXPROCS=4

# Tune garbage collector
export GOGC=100  # Default, lower = more frequent GC
```

### Connection Limits

```go
server := &http.Server{
    Addr:           ":8080",
    Handler:        handler,
    ReadTimeout:    10 * time.Second,
    WriteTimeout:   10 * time.Second,
    MaxHeaderBytes: 1 << 20, // 1 MB
}
```

### Resource Caching

Resources are automatically cached in memory by go-webglue. No additional configuration needed.

## Backup and Recovery

### Database Backups

```bash
# PostgreSQL
pg_dump mydb > backup.sql

# Restore
psql mydb < backup.sql
```

### Application State

If your API structs maintain state, implement persistence:

```go
type MyApi struct {
    counter int
    mu      sync.RWMutex
}

func (api *MyApi) saveState() error {
    api.mu.RLock()
    defer api.mu.RUnlock()

    data, _ := json.Marshal(api.counter)
    return os.WriteFile("state.json", data, 0644)
}

func (api *MyApi) loadState() error {
    data, err := os.ReadFile("state.json")
    if err != nil {
        return err
    }

    api.mu.Lock()
    defer api.mu.Unlock()

    return json.Unmarshal(data, &api.counter)
}
```

## Troubleshooting

### Common Issues

**Port already in use**:
```bash
# Find process using port
lsof -i :8080

# Kill process
kill -9 <PID>
```

**SSE not working behind proxy**:
- Check proxy buffering is disabled
- Ensure `Connection: keep-alive` is set
- Set appropriate timeouts

**Large binary size**:
```bash
# Strip debug info
go build -ldflags="-s -w"

# Check embedded resources
ls -lh client/
# Remove unused files
```

## Next Steps

- [Authentication](authentication.md) - Secure your deployment
- [Examples](examples.md) - Complete applications
- [Architecture](architecture.md) - Understanding internals
