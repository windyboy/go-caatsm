# ADR-004: REST API Design

## Status
Accepted

## Context
With the migration away from Hasura GraphQL, we need to provide an API for:
- Querying telegram messages
- Filtering and pagination
- Time-range queries
- Integration with frontend applications
- Third-party integrations

Options considered:
1. **GraphQL**: Flexible but adds complexity, already moving away from Hasura
2. **gRPC**: High performance but requires code generation, less web-friendly
3. **REST API**: Standard, widely supported, easy to integrate

## Decision
We will implement a REST API using the Echo framework, following RESTful principles and OpenAPI specification.

## Consequences

### Positive
- **Standard**: REST is a widely understood standard
- **Tooling**: Rich ecosystem of tools and libraries
- **Documentation**: OpenAPI/Swagger provides automatic documentation
- **Integration**: Easy to integrate with various clients
- **Caching**: HTTP caching can be leveraged
- **Simplicity**: Simpler than GraphQL for most use cases

### Negative
- **Over-fetching/Under-fetching**: Less flexible than GraphQL
- **Versioning**: Need to manage API versions
- **Multiple Requests**: Some operations may require multiple requests

### Mitigation
- Design endpoints to return appropriate data structures
- Use API versioning (e.g., `/api/v1/`)
- Provide comprehensive filtering and pagination options
- Document all endpoints with OpenAPI

## Implementation Notes
- API will follow RESTful conventions
- All endpoints will be versioned (`/api/v1/`)
- JSON will be used for request/response bodies
- Pagination will use limit/offset pattern
- Filtering will use query parameters
- OpenAPI 3.0 specification will be maintained
- Swagger UI will be available at `/swagger/`

## API Endpoints
- `GET /api/v1/telegrams` - List telegrams with filters
- `GET /api/v1/telegrams/:id` - Get telegram by ID
- `GET /api/v1/telegrams/time-range` - Get telegrams by time range
- `GET /health` - Health check
- `GET /health/ready` - Readiness check
- `GET /health/live` - Liveness check
- `GET /metrics` - Prometheus metrics

