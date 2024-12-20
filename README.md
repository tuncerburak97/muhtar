# Muhtar - Enterprise API Gateway

Muhtar is a high-performance, feature-rich API Gateway written in Go, designed to handle enterprise-level traffic with robust security, monitoring, and scalability features.

## Table of Contents
- [Overview](#overview)
- [Key Features](#key-features)
- [Architecture](#architecture)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Features in Detail](#features-in-detail)
- [Advanced Usage](#advanced-usage)
- [Performance Tuning](#performance-tuning)
- [Monitoring & Observability](#monitoring--observability)
- [Security](#security)
- [Contributing](#contributing)
- [License](#license)

## Overview

### Why Muhtar?

In today's microservices architecture, managing API traffic effectively is crucial. Muhtar addresses common challenges:

- **Traffic Management**: Handle millions of requests with sophisticated rate limiting
- **Security**: Protect services from abuse and attacks
- **Monitoring**: Get real-time insights into API traffic
- **Flexibility**: Support multiple databases and configurations
- **Performance**: Built for high throughput and low latency
- **Scalability**: Distributed architecture support

### Use Cases

1. **API Management**
   - Rate limiting and quota management
   - Request/response transformation
   - Traffic monitoring and analytics

2. **Microservices Gateway**
   - Service discovery
   - Load balancing
   - Circuit breaking
   - Request routing

3. **Security Gateway**
   - Authentication/Authorization
   - IP filtering
   - Request validation
   - Security headers management

## Key Features

### Core Features

- **High Performance**
  - Non-blocking I/O
  - Connection pooling
  - Efficient memory usage
  - Fast HTTP routing

- **Security**
  - Rate limiting
  - IP whitelisting
  - Security headers
  - Request validation

- **Monitoring**
  - Prometheus metrics
  - Structured logging
  - Request tracing
  - Performance analytics

- **Flexibility**
  - Multiple database support
  - Configurable middleware
  - Plugin system
  - Custom handlers

## Installation

### Prerequisites

- Go 1.19 or higher
- Redis (for distributed rate limiting)
- One of the supported databases:
  - PostgreSQL
  - MongoDB
  - Oracle
  - Couchbase

### From Source

```bash
# Clone the repository
git clone https://github.com/tuncerburak97/muhtar.git
cd muhtar

# Install dependencies
go mod download

# Build
go build -o muhtar cmd/main.go

# Run
./muhtar
```

### Using Docker

```bash
# Build image
docker build -t muhtar .

# Run container
docker run -p 8080:8080 muhtar
```

## Quick Start

1. **Basic Configuration**

```yaml
server:
  port: 8080
  host: "0.0.0.0"

proxy:
  target: "http://your-backend:8080"
  timeout: 30s

rate_limit:
  enabled: true
  global:
    requests: 1000
    window: 1m
```

2. **Run the Gateway**

```bash
./muhtar --config config.yaml
```

3. **Test the Setup**

```bash
curl http://localhost:8080/api/test
```

## Configuration

### Complete Configuration Reference

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  read_timeout: 10s
  write_timeout: 10s
  idle_timeout: 120s

proxy:
  target: "http://backend:8080"
  timeout: 30s
  max_idle_conns: 100
  retry_count: 3

rate_limit:
  enabled: true
  global:
    requests: 1000
    window: 1m
  per_ip:
    enabled: true
    requests: 100
    whitelist:
      - "127.0.0.1"
      - "10.0.0.0/8"

log:
  level: "info"
  format: "json"

db:
  type: "postgres"
  host: "localhost"
  port: 5432
```

## Features in Detail

### Rate Limiting

Muhtar provides multiple rate limiting strategies:

1. **Global Rate Limiting**
   ```yaml
   rate_limit:
     global:
       requests: 1000    # requests per window
       window: 1m       # time window
       burst: 50        # burst capacity
   ```

2. **Per-IP Rate Limiting**
   ```yaml
   rate_limit:
     per_ip:
       enabled: true
       requests: 100
       window: 1m
       whitelist:
         - "10.0.0.0/8"
   ```

3. **Route-Specific Limits**
   ```yaml
   rate_limit:
     routes:
       - path: "/api/v1/users"
         method: "POST"
         requests: 10
         window: 1m
   ```

### Header Transformation

Muhtar can modify request and response headers:

1. **Security Headers**
   - HSTS
   - CSP
   - XSS Protection
   - Frame Options

2. **Custom Headers**
   - Add/Remove headers
   - Rename headers
   - Set conditional headers

3. **Tracing Headers**
   - Request ID
   - Correlation ID
   - B3 Propagation

### Database Support

Multiple database backends with automatic connection pooling:

1. **PostgreSQL**
   - ACID compliance
   - Complex queries
   - JSON support

2. **MongoDB**
   - Document storage
   - High performance
   - Flexible schema

3. **Oracle**
   - Enterprise support
   - Advanced security

4. **Couchbase**
   - Distributed architecture
   - Memory-first design

## Advanced Usage

### Custom Middleware

```go
func CustomMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // Your custom logic
        return c.Next()
    }
}
```

### Error Handling

```go
app.Use(func(c *fiber.Ctx) error {
    return c.Status(500).JSON(fiber.Map{
        "error": "Internal Server Error"
    })
})
```

### Circuit Breaking

```yaml
circuit_breaker:
  threshold: 5
  timeout: 10s
  half_open_requests: 3
```

## Performance Tuning

### Memory Optimization

```yaml
server:
  read_buffer_size: 4096
  write_buffer_size: 4096
```

### Connection Pooling

```yaml
proxy:
  max_idle_conns: 100
  max_conns_per_host: 100
```

### Rate Limit Storage

```yaml
rate_limit:
  storage:
    type: "redis"
    redis:
      pool_size: 10
```

## Monitoring & Observability

### Prometheus Metrics

```yaml
metrics:
  enabled: true
  path: "/metrics"
```

Available metrics:
- Request count
- Response times
- Error rates
- Rate limit hits

### Logging

```yaml
log:
  level: "debug"
  format: "json"
  output: "stdout"
```

### Tracing

```yaml
tracing:
  enabled: true
  type: "jaeger"
```

## Security

### Best Practices

1. **Rate Limiting**
   - Set appropriate limits
   - Use IP whitelisting
   - Implement burst control

2. **Headers**
   - Remove internal headers
   - Add security headers
   - Validate content types

3. **Authentication**
   - Use strong auth methods
   - Implement token validation
   - Set proper timeouts

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

### Development Setup

```bash
# Setup development environment
make setup

# Run tests
make test

# Run linter
make lint
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 