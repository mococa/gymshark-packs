# RE Partners Pack Calculator

**Live Demo:** [gymshark-challenge.moureau.dev](https://gymshark-challenge.moureau.dev)
**API Docs:** [/docs](https://gymshark-challenge.moureau.dev/docs)

Pack calculator implementation for the RE Partners challenge.

Built by [Luiz Moureau](https://moureau.dev) | luiz@moureau.dev

---

## Quick Start

```bash
# Install dependencies
make install-tools

# Run locally
make dev

# Run tests
make test

# Build
make build
```

Server runs on `http://localhost:8080`

---

## Running with Docker

**Docker Compose (recommended for local use):**

```bash
docker-compose up --build
```

Opens on `http://localhost:8080`. Uses the `dev` target (Alpine-based, plain HTTP server) with SQLite persistence mounted at `/data/packs.db`.

**Docker only:**

```bash
docker build -t packs .
docker run -p 8080:8080 packs
```

> **Note:** The `Dockerfile` has two targets: `dev` (Alpine, plain HTTP server — **default**) and `prod` (Lambda Web Adapter, for AWS deployment only). Use `--target prod` only when building for Lambda.

---

## The Challenge

Customers order items shipped in fixed pack sizes. Calculate the optimal pack breakdown:

1. **Only whole packs** (cannot break them open)
2. **Minimize total items** (least overage)
3. **Minimize number of packs** (fewest boxes)

### Examples

| Order | Result                       | Total Items | Packs |
|-------|------------------------------|-------------|-------|
| 1     | 1×250                        | 250         | 1     |
| 251   | 1×500                        | 500         | 1     |
| 501   | 1×500 + 1×250                | 750         | 2     |
| 12001 | 2×5000 + 1×2000 + 1×250      | 12,250      | 4     |

### Edge Case (from challenge email)

| Order | Sizes | Result | Total |
|-------|-------|--------|-------|
| 500,000 | [23, 31, 53] | 9429×53 + 7×31 + 2×23 | 500,000 |

---

## Algorithm: Dynamic Programming

The challenge requires **configurable pack sizes**, so greedy algorithms fail.

**Example where greedy fails:**
- Sizes: `[6, 9, 20]`, Order: `15`
- Greedy (largest first): `1×20 = 20` ❌ (overage: 5)
- **DP optimal**: `1×9 + 1×6 = 15` ✅ (overage: 0)

### Implementation

```go
// 1. Build DP table: dp[i] = minimum packs to make exactly i items
// 2. Find minimum total >= order (minimizes overage first)
// 3. Reconstruct packs from DP table (minimizes pack count second)
```

See [internal/calculator/calculator.go](internal/calculator/calculator.go) for full implementation.

**Time complexity:** O(n × m) where n = order size, m = pack count
**Space complexity:** O(n)

**Benchmark results:**
```
// Cold path — DP computation
BenchmarkCalculatePacksCold/Small_Order_1-4               20616    109758 ns/op
BenchmarkCalculatePacksCold/Large_Order_12001-4            6229    358211 ns/op
BenchmarkCalculatePacksCold/Very_Large_Order_50000-4       2086   1141335 ns/op
BenchmarkCalculatePacksCold/Edge_Case_500000-4              237   9018147 ns/op

// Warm path — cache hit (flat regardless of order size)
BenchmarkCalculatePacksWarm/Small_Order_1-4             926421      2622 ns/op
BenchmarkCalculatePacksWarm/Edge_Case_500000-4          755872      2664 ns/op
```

---

## API

### Calculate Packs
```bash
POST /api/calculate
Content-Type: application/json

{
  "order": 501
}

# Response:
{
  "order": 501,
  "total_items": 750,
  "total_packs": 2,
  "packs": { "250": 1, "500": 1 }
}
```

### Manage Pack Sizes
```bash
GET    /api/pack-sizes           # List all sizes
POST   /api/pack-sizes           # Add new size
DELETE /api/pack-sizes/{size}    # Remove size (atomic, prevents deleting last)
```

### Health Check
```bash
GET /healthz
```

**Full API documentation:** [/docs](https://gymshark-challenge.moureau.dev/docs) (Swagger UI)

---

## Testing

All spec examples validated + edge case:

```bash
make test
```

| Order | Expected            | ✅ Result          |
|-------|---------------------|--------------------|
| 1     | 1×250              | 1×250              |
| 250   | 1×250              | 1×250              |
| 251   | 1×500              | 1×500              |
| 501   | 1×500 + 1×250      | 1×500 + 1×250      |
| 12001 | 2×5000 + 1×2000... | 2×5000 + 1×2000... |
| 500000 (23,31,53) | 9429×53 + 7×31 + 2×23 | ✅ Exact match |

**Test coverage:**
- ✅ All challenge specification examples
- ✅ Edge case from email (500,000 with sizes [23, 31, 53])
- ✅ Edge cases (zero, negative, empty sizes)
- ✅ Arbitrary pack sizes (validates DP correctness vs greedy)
- ✅ Thread safety (concurrent store operations)
- ✅ Request validation (unknown fields, max body size, trailing JSON)
- ✅ Business rules (atomic deletion, last pack protection)

---

## Architecture

**Single Go Application:**
- Server-side rendering with `templ` (type-safe Go templates)
- Embedded static assets (`//go:embed`)
- REST API + Web UI in one binary
- Swagger documentation (`/docs`)

**Stack:**
- Go 1.23 with Chi router
- templ for type-safe HTML templates
- Vanilla JavaScript (no frameworks)
- AWS Lambda with Web Adapter (or Docker for local dev)

**Why this approach:**
- Backend-focused (single Go binary)
- Type-safe templates compile to Go code
- Embedded assets (no build orchestration)
- Serverless deployment via Lambda Web Adapter (preserves standard net/http patterns)

**Persistence Strategy:**
- **Interface-based store** with pluggable drivers
- **DynamoDB** for Lambda/production (fully persistent)
- **SQLite** for local development (file-based)
- **In-memory** for tests (fast, isolated)
- Driver selected via `STORE_DRIVER` environment variable

---

## Project Structure

```
gymshark-packs/
├── cmd/server/
│   └── main.go                  # Server entrypoint with graceful shutdown
├── internal/
│   ├── calculator/              # DP algorithm + comprehensive tests
│   ├── handler/                 # REST API handlers with validation
│   ├── store/                   # Pluggable store: memory, SQLite, DynamoDB
│   └── web/
│       ├── templates/           # templ files (compile to Go)
│       │   ├── base.templ
│       │   ├── home.templ
│       │   └── notfound.templ
│       ├── static/
│       │   ├── css/style.css
│       │   └── js/app.js        # Vanilla JS for API calls
│       └── web.go               # Web handlers with go:embed
├── terraform/                   # Infrastructure as Code (optional)
├── Dockerfile                   # Multi-stage (distroless base)
├── docker-compose.yml           # Local development
└── go.mod
```

---

## Deployment

### AWS Lambda (current deployment)

1. **Set up GitHub OIDC trust in AWS** (one-time, manual):

   GitHub Actions authenticates to AWS via OIDC — no long-lived access keys stored in secrets.

   ```bash
   sh ./scripts/setup-oidc.sh YOUR_GITHUB_USERNAME YOUR_REPO
   ```

   The script derives the thumbprint from the live TLS chain, creates the OIDC provider and IAM role, and prints the role ARN to add as a secret.

   Then add these to your GitHub repository secrets:

   | Secret | Value |
   |--------|-------|
   | `AWS_ROLE_ARN` | Printed by the script above |
   | `TF_STATE_BUCKET` | Name of the S3 bucket (set in next step) |
   | `TF_STATE_LOCK_TABLE` | Name of the DynamoDB lock table (set in next step) |
   | `DOMAIN_NAME` | Your optional custom domain (e.g. `packs.example.com`) |

2. **Bootstrap Terraform state** (one-time):
   ```bash
   make bootstrap
   ```

3. **Deploy Lambda function:**
   ```bash
   cd terraform/environments/prod
   terraform init
   terraform apply -var="domain_name=your-domain.example.com"
   ```

4. **Configure custom domain** (optional):

   After `terraform apply`, run `terraform output dns_setup_instructions` for complete setup.

   Example for DNS provider:
   ```
   Type:   CNAME
   Name:   your-subdomain
   Target: abc123xyz.lambda-url.us-east-1.on.aws
   TTL:    300
   ```

5. **Deploy code:**
   - Push to `main` branch → GitHub Actions builds and deploys automatically
   - Or manually: build Docker image → push to ECR → Lambda updates automatically

### Alternative Deployments

**Fly.io** (simpler):
```bash
fly launch
fly deploy
```

**Render**: Connect repo, select Docker, deploy

**Docker Compose** (local — uses `dev` target):
```bash
docker-compose up --build
```

---

## Development

### Prerequisites
- Go 1.23+
- templ CLI: `go install github.com/a-h/templ/cmd/templ@latest`

### Workflow
```bash
# Generate templ files
make templ

# Run tests (uses in-memory store)
make test

# Run server with SQLite persistence
STORE_DRIVER=sqlite make dev-api

# Run server with in-memory (no persistence)
make dev-api
```

### Storage Drivers

Set `STORE_DRIVER` environment variable:

- **`dynamodb`** - DynamoDB (Lambda production, fully persistent)
  - Requires: `DYNAMODB_TABLE`, `AWS_REGION_CUSTOM`
- **`sqlite`** - SQLite (local dev, file-based persistence)
  - Optional: `SQLITE_PATH` (default: `/tmp/packs.db`)
- **empty/unset** - In-memory (tests, no persistence)

The Makefile handles templ generation automatically for build commands.

---

## Security

- ✅ Request validation (`DisallowUnknownFields`, 1KB max body, trailing JSON rejection)
- ✅ Atomic business rule enforcement (last pack protection)
- ✅ Minimal container image (AWS Lambda base + Web Adapter)
- ✅ Structured JSON logging (compatible with CloudWatch)
- ✅ AWS-managed runtime isolation with IAM-configurable permissions

---

## What I'd Add for Production

- [x] **Persistence** - DynamoDB for Lambda, SQLite for local (implemented)
- [x] **Rate limiting** - per-IP token-bucket (10 req/s, burst 20) on `/api/*`
- [ ] **Observability** - Prometheus metrics, distributed tracing
- [ ] **Distributed caching** - Redis/ElastiCache for multi-instance calc results
- [ ] **API versioning** - `/v1/calculate` for breaking changes
- [ ] **Multi-region** - DynamoDB global tables for high availability

---

Built for RE Partners technical challenge.

Main domain: [moureau.dev](https://moureau.dev)
