# Flight Search & Aggregation System

A flight search and aggregation API that combines flight data from multiple Indonesian airline providers, normalizes varying data formats, and returns optimized search results with filtering, sorting, and "best value" ranking.

## Setup & Run

**Prerequisites:** Go 1.25+

```bash
# Run the server
go run cmd/api/main.go

# Run tests
go test ./... -v

# Build binary
go build -o flight-aggregator cmd/api/main.go
./flight-aggregator
```

The server starts on port `8080` by default. Set the `PORT` environment variable to change it.

## API Endpoints

### `POST /api/v1/flights/search`

Search for flights across all providers.

**Request:**

```json
{
  "origin": "CGK",
  "destination": "DPS",
  "departureDate": "2025-12-15",
  "returnDate": null,
  "passengers": 1,
  "cabinClass": "economy",
  "filters": {
    "minPrice": 500000,
    "maxPrice": 1500000,
    "maxStops": 1,
    "airlines": ["Garuda Indonesia", "GA"],
    "departureAfter": "06:00",
    "departureBefore": "20:00",
    "arrivalAfter": "08:00",
    "arrivalBefore": "22:00",
    "maxDuration": 300
  },
  "sortBy": "best_value"
}
```

**Required fields:** `origin`, `destination`, `departureDate`, `passengers` (>= 1).

**Validation:**
- `origin` and `destination` must be valid Indonesian airport codes (CGK, DPS, SUB, UPG, etc.) and must differ.
- `departureDate` must be in `YYYY-MM-DD` format.
- `cabinClass`, if provided, must be one of: `economy`, `business`, `first`.
- All filter fields are optional.

**Available `sortBy` values:**
- `price_asc`, `price_desc`
- `duration_asc`, `duration_desc`
- `departure_asc`, `departure_desc`
- `arrival_asc`, `arrival_desc`
- `best_value` (default)

**Airline filter** matches by airline name (case-insensitive) or IATA code (e.g., `"GA"`, `"Garuda Indonesia"`).

**Example:**

```bash
curl -X POST http://localhost:8080/api/v1/flights/search \
  -H "Content-Type: application/json" \
  -d '{
    "origin": "CGK",
    "destination": "DPS",
    "departureDate": "2025-12-15",
    "passengers": 1,
    "cabinClass": "economy"
  }'
```

**Response:**

```json
{
  "search_criteria": {
    "origin": "CGK",
    "destination": "DPS",
    "departure_date": "2025-12-15",
    "passengers": 1,
    "cabin_class": "economy"
  },
  "metadata": {
    "total_results": 13,
    "providers_queried": 4,
    "providers_succeeded": 4,
    "providers_failed": 0,
    "search_time_ms": 275,
    "cache_hit": false
  },
  "flights": [
    {
      "id": "QZ532_AirAsia",
      "provider": "AirAsia",
      "airline": { "name": "AirAsia", "code": "QZ" },
      "flight_number": "QZ532",
      "departure": {
        "airport": "CGK",
        "city": "Jakarta",
        "datetime": "2025-12-15T19:30:00+07:00",
        "timestamp": 1765801800
      },
      "arrival": {
        "airport": "DPS",
        "city": "Denpasar",
        "datetime": "2025-12-15T22:10:00+08:00",
        "timestamp": 1765807800
      },
      "duration": { "total_minutes": 100, "formatted": "1h 40m" },
      "stops": 0,
      "price": { "amount": 595000, "currency": "IDR", "display": "IDR 595,000" },
      "available_seats": 72,
      "cabin_class": "economy",
      "aircraft": null,
      "amenities": [],
      "baggage": { "carry_on": "Cabin baggage only", "checked": "checked bags additional fee" },
      "score": 0.04
    },
    {
      "id": "GA315_Garuda Indonesia",
      "provider": "Garuda Indonesia",
      "airline": { "name": "Garuda Indonesia", "code": "GA" },
      "flight_number": "GA315",
      "departure": {
        "airport": "CGK",
        "city": "Jakarta",
        "datetime": "2025-12-15T14:00:00+07:00",
        "timestamp": 1765782000
      },
      "arrival": {
        "airport": "DPS",
        "city": "Denpasar",
        "datetime": "2025-12-15T18:45:00+08:00",
        "timestamp": 1765795500
      },
      "duration": { "total_minutes": 225, "formatted": "3h 45m" },
      "stops": 1,
      "price": { "amount": 1850000, "currency": "IDR", "display": "IDR 1,850,000" },
      "available_seats": 22,
      "cabin_class": "economy",
      "aircraft": "Boeing 737",
      "amenities": [],
      "baggage": { "carry_on": "1 piece", "checked": "2 pieces" },
      "score": 0.83
    }
  ]
}
```

*(Showing first and last of 13 results, sorted by best value)*

### `GET /api/v1/health`

Health check endpoint.

## Architecture

```
cmd/api/              Entry point, HTTP server, provider/rate-limiter wiring, graceful shutdown
internal/
├── domain/           Core types (Flight, SearchRequest/Response), airport lookup, helpers
├── provider/         Provider interface + 4 implementations
│   └── mockdata/     Embedded JSON mock responses
├── aggregator/       Parallel provider orchestration, retry, request filtering, validation
├── search/           User-facing filtering, sorting, best-value ranking
├── cache/            In-memory cache with per-key TTL
├── ratelimiter/      Token bucket rate limiter (shared by aggregator and HTTP middleware)
├── api/              HTTP handlers, request validation, IP-based rate-limiting middleware
└── mocks/            Generated mocks for Provider and RateLimiter interfaces
```

## Providers

| Provider | Time Format | Simulated Latency | Rate Limit | Notes |
|---|---|---|---|---|
| Garuda Indonesia | RFC 3339 | 50-100ms | 10 req/s | Segments override arrival/stops when present |
| Lion Air | Naive datetime + IANA timezone | 100-200ms | 8 req/s | Fare types: ECONOMY, BUSINESS, FIRST |
| Batik Air | Compact offset (`+0700`) | 200-400ms | 5 req/s | Fare class letters: Y/C/F |
| AirAsia | RFC 3339 | 50-150ms | 15 req/s | 10% simulated failure rate, airline code from flight code prefix |

Each provider normalizes its response into the unified `domain.Flight` type. Durations are always computed from parsed departure/arrival timestamps rather than trusting provider-supplied values.

## Design Decisions

**Provider normalization at the boundary.** Each provider has different JSON structures, time formats, and naming conventions. Each provider implementation parses its specific format and emits the unified `domain.Flight` type. All field values (ID, airline name, airline code) come from the provider's payload.

**Data inconsistency handling.** Garuda's GA315 flight reports `stops: 0` and arrival at SUB, but contains segments showing a connecting flight through SUB to DPS. The normalizer detects when segments exist and overrides the arrival, stops count, and duration with values derived from the actual segments.

**Missing optional fields.** Providers may omit optional fields. The normalize functions handle this: nil amenities and services become empty slices (not null in JSON), nil aircraft stays null, and boolean service flags (Lion Air's wifi/meals) are converted to a string list only when true.

**Flight data validation.** After aggregation, `validateFlights` filters out invalid data before caching: arrival timestamp must be after departure, duration must be positive, and price must be positive.

**Two-stage filtering.** `filterByRequest` in the aggregator filters provider results to match the search criteria (route, date, passengers, cabin class). `search.Apply` then applies user-facing filters (price range, stops, airlines, time windows, duration) and sorting. This separation means the aggregator caches only relevant flights, while filter/sort combinations share a single cache entry.

**Cache-then-filter strategy.** Raw results are cached by search key (origin + destination + date + passengers + cabin class). Filters and sorting are applied after cache retrieval, so different filter/sort combinations on the same search criteria hit the same cache entry. Partial failures (some providers down) are not cached to force a fresh query on the next request.

**Per-key TTL cache.** TTL is set per cache entry at the call site (default 5 min). The cache stores up to 1000 entries. 5 min is short enough that flight prices stay reasonably fresh and long enough to absorb repeated searches with different filter/sort combinations.

**Rate limiting.** Each provider gets its own token-bucket rate limiter, configured by the provider itself via `RateLimit()`. The HTTP layer has a separate IP-based rate limiter (30 req/min) using the same `RateLimiter` interface, with stale entry cleanup based on `LastRefill()`.

**Retry with backoff.** Failed provider calls are retried up to 3 times with exponential backoff (500ms, 1s). Backoff respects context cancellation for early exit.

**Best value scoring.** The ranking algorithm normalizes price and duration to [0,1] relative to the current result set, then combines with a stops penalty:

```
score = 0.5 × normalized_price + 0.3 × normalized_duration + 0.2 × stops_penalty
```

Lower scores = better value. Stops penalty: 0 (direct), 0.5 (1 stop), 1.0 (2+ stops).

## Performance

- **Parallel provider queries.** All providers are queried concurrently. Total latency is bounded by the slowest provider (max ~400ms for Batik Air), not the sum.
- **Cache lookup:** O(1) via hash map. Repeated searches with different filters/sorts skip provider queries entirely.
- **Filtering:** O(n) single pass over flights for both `filterByRequest` and `applyFilters`.
- **Sorting:** O(n log n) via Go's `slices.SortStableFunc`. Stable sort preserves relative order for flights with equal sort keys.
- **Best value scoring:** O(n). One pass to find min/max, one pass to compute scores.
- **Per-request timeout.** Each provider call has a 2s timeout. A single slow or hung provider cannot block the entire search.

## Test Coverage

The project has 114 test cases across 6 packages:

- **aggregator** — caching, partial failure, timeout, retry (success/exhausted/context cancelled), rate limiter fail-open, cache-with-filters, request filtering (date, route, seats, cabin class)
- **cache** — set/get, TTL expiry, eviction, max size
- **domain** — duration formatting, IDR formatting, cache key, airport lookup
- **provider** — table-driven normalize tests per provider (Garuda, Lion Air, Batik Air, AirAsia) covering field mapping, timezone handling, segment override, fare class mapping, baggage parsing, amenities, edge cases, error paths; time parsing format tests
- **ratelimiter** — immediate/burst/blocking wait, context cancellation, token refill, concurrency, allow when available/exhausted
- **search** — all filter types (price range, stops, duration, airlines by name/code, departure/arrival time windows, combined), all sort modes (price, duration, departure, arrival, best value, default, unknown), scoring (single/multi flight, stops penalty, boundary values)

## Not Implemented

- Round-trip searches
- Multi-city searches