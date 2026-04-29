# internal

Private application packages.

- `calculator/` - dynamic-programming algorithm that finds the optimal pack breakdown for an order.
- `handler/` - REST API handlers with request validation.
- `middleware/` - per-IP token-bucket rate limiter.
- `store/` - pluggable pack-size persistence: in-memory, SQLite, DynamoDB.
- `web/` - web UI handlers, embedded static assets, and templ templates.
