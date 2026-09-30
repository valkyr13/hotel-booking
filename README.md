# Hotel Booking System

Rupeek SDE-3 Machine Coding Round — Question A

Built in **Go 1.22** (the brief specifies Java 17+/Spring Boot; the language was swapped, all functional and technical requirements unchanged).

## How to run

**Prerequisites:** Go 1.22+

```bash
# from the project root
go build ./...              # confirm everything compiles
go test ./... -race         # run the full test suite (unit + integration), with the race detector
go run ./cmd                # start the REST server on :8080
```

## API

| Method | Path | Purpose |
|---|---|---|
| POST | `/owners` | Create an owner |
| POST | `/owners/{ownerID}/properties` | Add a property to an owner |
| POST | `/properties/{propertyID}/room-types` | Add a room type to a property (also registers its inventory) |
| GET | `/search` | Discover properties — query params: `city`, `minStars`, `amenity`, `minPrice`+`maxPrice`, `checkIn`+`checkOut`+`guests` |
| POST | `/bookings` | Book a room type for a date range |
| POST | `/bookings/{id}/pay` | Pay for a pending booking |
| POST | `/bookings/{id}/cancel` | Cancel a confirmed booking (refund calculated automatically) |

A complete working example of every endpoint together — onboard → search → book → pay → cancel — is in `api/integration_test.go`, run against a real HTTP server via `httptest`.

## Project structure

hotel-booking/
├── cmd/main.go — composition root: the only place that knows concrete implementation types
├── owner/ — Owner, Property, RoomType, Amenity, Money, DateRange + their repositories
├── availability/ — inventory: sparse per-night map, RWMutex, Reserve/Release/IsAvailable
├── booking/ — Booking struct, state machine, BookingService (orchestrates owner + availability)
├── payment/ — PaymentMethod, PaymentGateway (mocked), PaymentService
├── refund/ — RefundPolicy, TieredRefundPolicy, cancellation orchestration
├── search/ — Filter interface + concrete filters, search orchestration
└── api/ — HTTP handlers + DTOs; imports every package above, none of them import it



Packages are organized by domain responsibility (the Go idiom), not by technical layer (`controller/service/repo`), and each domain package owns its own repository interface rather than sharing one generic package — see "Key design decisions" below.

## Key design decisions

**Domain model.** `RoomType` is a plain struct with a data-driven `Name`, not a hardcoded enum — adding a new category ("Suite") is a data change at onboarding, not a code change. A single-property owner is simply an `Owner` with one property; there's no separate type for it. `Property` stores `RoomTypeIDs []string`, not nested `RoomType` data, so other parts of the system can reference a room type directly by ID.

**Availability.** Inventory is tracked as a sparse map — `map[date]int` per room type, storing only nights that actually have bookings; absence means fully available. Checking and reserving are one atomic operation guarded by a `sync.RWMutex` per room type: `Reserve()` takes the exclusive write lock, `IsAvailable()` takes the shared read lock, so browsing traffic never blocks on itself while still serializing genuine contention for the same room. A dedicated concurrency test spins up 100 goroutines against the last available room and asserts exactly one wins.

**Booking lifecycle.** A strict state machine: `PENDING_PAYMENT → CONFIRMED` or `FAILED` (payment outcome), `CONFIRMED → CANCELLED` (guest cancels). Every transition is guarded — attempting one from the wrong state returns an error rather than silently succeeding, which matters because `Release()` (and refunds) must never fire twice on a double-cancel.

**Payment.** `PaymentMethod` is an interface; only `CardPayment` is implemented fully. UPI and wallet would be the same shape wrapping the same mocked `PaymentGateway` — left as a stated extension point rather than built shallow, given the brief's explicit preference for a smaller, well-designed core over broad, thin feature coverage. A decline and a gateway-unreachable error are deliberately treated identically at the booking layer (both mean `FAILED` + `Release()`) even though the `PaymentGateway` interface preserves the distinction for a future UI layer.

**Refund policy.** `RefundPolicy.CalculateRefund(booking, cancelDate)` returns a final rupee `Money` amount, not a percentage — a percentage-only interface would silently assume every policy is shaped like a percentage, which a flat cancellation-fee policy wouldn't fit. The default `TieredRefundPolicy`: ≥10 days = 90%, ≥3 days = 80%, ≥1 day = 50%, same-day = 30%.

**Search.** Every filter — shallow (reads one `Property` field) or deep (traverses into that property's room types or live availability) — implements the same `Matches(property) bool` contract. Search runs the full property list through each active filter in sequence. Adding a new filter later means writing one new type with that same shape; the search loop itself never changes.

**Package layering.** Go interfaces are defined at the consumer side, not the implementer side — e.g. `BookingRepository` is declared inside `booking`, next to the `BookingService` that uses it, not in a shared `repository` package. Domain packages never import `api`; `api` imports them. `main.go` is the only file that knows persistence is currently in-memory — swapping to a real database means new files inside the relevant package and a one-line change in `main.go`, nothing else.

## Assumptions

1. One booking = one room, of one room type (no multi-room bookings).
2. Dates are half-open: `[checkIn, checkOut)` — checkout day itself doesn't block a new check-in.
3. Inventory is tracked at the room-type level, not individual room numbers.
4. Owner → properties is one level deep (no nested chains/brands).
5. Money is integer rupees only; no sub-rupee amounts; discounts round to the nearest rupee.
6. Payment failure fails the booking and releases inventory immediately — no pending/retry hold with a TTL.
7. Inventory is held at booking creation (state `PENDING_PAYMENT`), confirmed on payment success.
8. Kids are counted as adults — no separate child-capacity math; `MaxOccupancy` is a single number.
9. A `PENDING_PAYMENT` booking cannot be independently cancelled — it only resolves via the payment outcome.
10. Most domain errors (invalid input, invalid state transitions, no availability) map to HTTP 400; only "record doesn't exist" errors map to 404, since the domain layer doesn't yet distinguish validation failures from other error types with dedicated error types of their own.

## What I'd do with more time

- Additional payment methods (UPI, wallet) — `PaymentMethod` is already built for this; only `CardPayment` was completed
- Payment idempotency and a hold-expiry TTL on `PENDING_PAYMENT` bookings (both explicitly bonus in the brief)
- Swap `availability`'s in-memory mutex for a real database transaction (`SELECT ... FOR UPDATE`) if persistence moved to Postgres — the atomicity concept survives; the mechanism would change
- OpenAPI/Swagger documentation for the REST API
- Typed/sentinel errors throughout the domain layer, so the API can return more precise HTTP status codes than the current not-found-vs-everything-else split

## Test coverage

All packages pass under `go test ./... -race`. Testing priority followed the rubric's weighting toward core business logic over blanket coverage:

1. **Booking state machine** — every transition, positive and negative, with side-effect verification (checking `Release()` actually fired, not just the status flag)
2. **Concurrency** — a dedicated test proving `Reserve()` allows exactly one winner among concurrent requests for the same last room
3. **Refund tier boundaries** — one test per boundary value (10, 3, 1, and 0 days out)
4. **Filter matching** — each filter individually, including the "any room type qualifies" behavior for deep filters, plus a chained multi-filter search
5. **One end-to-end HTTP integration test** exercising the full stack against a real server