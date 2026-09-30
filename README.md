# Hotel Booking System — Design Decisions

Rupeek SDE-3 machine coding round · built in Go (brief specifies Java/Spring Boot; language swapped, requirements unchanged)

This document tracks decisions as they're made, unit by unit, along with the reasoning behind each — so it can double as the "key design decisions and assumptions" section of the final README.

---

## Scope

**In scope (core):**

- Onboard an owner with one or many properties, each with room types, amenities, and pricing
- Search by city, dates, and guests, with filters for price, amenities, and star rating — returning only available properties
- Book, pay, and cancel, with a defined booking state machine
- Availability check plus atomic reservation, and inventory release on cancel
- Basic concurrency safety on shared inventory (treated as core, not bonus, despite being listed under bonus in the brief — it's cheap once the inventory model is right, and likely to come up in the follow-up discussion)
- Unit tests on the core business rules

**Seams only (an interface plus one implementation, nothing more):** pricing strategy, cancellation policy, payment methods, repositories.

**Bonus, after the core is solid:** payment idempotency, hold expiry, Swagger docs.

**Out of scope:** auth, UI, a real payment gateway, a real database.

---

## Assumptions

1. One booking = one room, of one room type (no multi-room bookings).
2. Dates are half-open: `[checkIn, checkOut)` — checkout day itself doesn't block a new check-in.
3. Inventory is tracked at the room-type level, not individual room numbers.
4. Owner → properties is one level deep (no nested chains/brands).
5. Money is integer rupees only; no sub-rupee amounts; discounts round to the nearest rupee.
6. Payment failure fails the booking and releases inventory immediately — no pending/retry hold with a TTL.
7. Inventory is held at booking creation (state `PENDING_PAYMENT`), confirmed on payment success.
8. Kids are counted as adults — no separate child-capacity math; `MaxOccupancy` is a single number.

---

## Unit 1 — Domain model

**Owner**

- Has an identity and a name.
- Owns one or more properties; a single-property owner is just an `Owner` with one `Property` — not a separate type.

**Property**

- Belongs to exactly one owner.
- Holds `RoomTypeIDs []string` — not nested `RoomType` data — so other parts of the system (bookings, inventory) can reference a `RoomType` directly without going through `Property`.
- Has: identity, name, location (city/locality), star rating, property-level amenities.

**RoomType**

- A struct, not an enum — this is the key extensibility decision. A hardcoded enum (Single/Double/Duplex/...) can't support "capacity is customisable per property," and adding a new type would mean editing code. A struct means adding "Suite" is just creating a new `RoomType` instance at onboarding — no code change.
- Belongs to exactly one property via a `PropertyID` back-reference (so a `RoomType` can be looked up independently and still traced back to its property).
- Has: identity, name (free text), `MaxOccupancy` (single number — see assumption 8), base price per night, room-level amenities, and `RoomCount` (how many identical rooms of this type exist — this *is* the inventory total).

**Amenity**

- A closed, typed set of Go constants (not free-form strings) — type-safe, cheap for filtering. Deliberately not the extensibility seam; that job belongs to room types, payment methods, and filters instead.

**Money**

- Integer rupees, no sub-unit (assumption 5).

**DateRange**

- `CheckIn` / `CheckOut` pair, half-open, used identically for search queries and booking stays.

---

## Unit 2 — Availability & concurrency

**Core insight:** availability is a fact about **(RoomType, single night)** — never one number for a whole room type. Two independent bookings on different dates for the same room type must be trackable independently.

**Storage — sparse map:**

```
booked map[RoomTypeID]map[Date]int   // taken counts only
available(date) = TotalRooms - booked.get(date, default=0)
```

Only nights with actual bookings get an entry — absence means "fully available." This avoids pre-populating years of mostly-empty rows per room type, which doesn't scale. Booking increments touched nights; cancellation decrements, and deletes the entry once it reaches zero.

**Atomicity:** a booking touches every night in its range at once. Checking availability and reserving must be a single atomic operation — read-then-write split across two steps allows two concurrent requests to both read "1 remaining" and both decrement, causing a double-booking (verified by walking through a two-guest race).

**All nights locked together, not one at a time** — locking night-by-night can let a competing booking take a later night in the range after the current booking has already claimed an earlier one, producing a partial, broken reservation.

**Lock granularity — read vs write:**

- `IsAvailable()` (search / browsing) is read-only. A stale read here is harmless — worst case, a guest sees a room that gets taken a moment later and receives a normal "just booked" rejection. No correctness bug.
- `Reserve()` (check-and-decrement) is the only operation that must be exclusive — concurrent writes are what actually corrupt data.
- This maps to `sync.RWMutex`: `Reserve()` takes `Lock()` (exclusive), `IsAvailable()` takes `RLock()` (shared, many concurrent readers). Using a plain `Mutex` for both would serialize high-volume, low-risk search traffic behind low-volume, high-risk booking traffic — unnecessary contention.

**Lock scope — per (RoomType, Property), not global.** A global lock would serialize unrelated bookings anywhere in the system (Mumbai vs. Bengaluru) even though they share no data — pure wasted contention. Scoping the lock per room type per property means only genuine contention for the *same* inventory ever waits.

**Interface (seam):**

```
AvailabilityService:
  IsAvailable(roomTypeID, dateRange) bool
  Reserve(roomTypeID, dateRange) (bool, error)
  Release(roomTypeID, dateRange) error
```

Booking logic depends only on this interface, never on the map or the mutex directly — the implementation (in-memory now) can later be swapped for a real store without touching booking code.

---

## Unit 3 — Booking lifecycle

| From | Event | To | Side effect |
| --- | --- | --- | --- |
| — | `Reserve()` succeeds | `PENDING_PAYMENT` | inventory decremented |
| `PENDING_PAYMENT` | payment succeeds | `CONFIRMED` | — |
| `PENDING_PAYMENT` | payment fails | `FAILED` | `Release()` |
| `CONFIRMED` | guest cancels | `CANCELLED` | `Release()` |
| `FAILED` / `CANCELLED` | any transition attempted | — | **rejected as an error** |

**Key decisions:**

- `FAILED` and `CANCELLED` are terminal, and distinct from each other — `FAILED` means payment never succeeded; `CANCELLED` means it was confirmed, then the guest backed out. Different history, same eventual release of inventory.
- A `PENDING_PAYMENT` booking cannot be independently cancelled — it only resolves via the payment outcome (success or fail). This is a deliberate scope narrowing, not an oversight, and worth stating as an assumption.
- Rejecting transitions from a terminal state isn't just tidiness — it prevents `Release()` (and later, refunds) from firing twice on a double cancellation attempt. This is the "invalid state transitions" edge case the rubric calls out by name.

---

## Unit 4 — Payment

**Interfaces (seams):**

```
PaymentMethod:
  Pay(userID, bookingID, amount) → (success bool, transactionID string, error)

PaymentGateway (used internally by each PaymentMethod):
  Charge(...) → mocked result
```

**Flow:** `PaymentService` calls `PaymentMethod.Pay()` → gets the outcome → transitions the booking (`CONFIRMED` on success, `FAILED` + `Release()` on failure). This is the one place payment and the booking lifecycle meet — neither `Booking` nor `PaymentMethod` knows about the other directly.

**Key decisions:**

- Payment is synchronous only — every call resolves immediately to success or failure. No pending/async/webhook flow; idempotency is deferred to bonus, consistent with the brief treating it as bonus-only.
- Declined (gateway processed, said no) and gateway-error (couldn't reach it) collapse to the same booking outcome (`FAILED`) — the booking doesn't need a different code path for the two, even though a real UI layer might message them differently.
- The mock gateway is a separate, swappable `PaymentGateway` component called by each `PaymentMethod` implementation — not baked directly into `CardPayment`/`UpiPayment`/`WalletPayment`. This lets tests inject "always fail" or "fail every Nth call" without touching payment-method code, and mirrors the brief's instruction to mock third-party dependencies behind an abstraction.

---

## Unit 5 — Cancellation & refund

**Interface (seam):**

```
RefundPolicy:
  CalculateRefund(booking, cancelDate) → Money
```

**Default implementation — `TieredRefundPolicy`:**

| Days until check-in | Refund |
| --- | --- |
| ≥ 10 | 90% |
| ≥ 3 | 80% |
| ≥ 1 | 50% |
| 0 (same-day) | 30% |

**Flow:** cancelling a `CONFIRMED` booking → `RefundPolicy.CalculateRefund()` returns the rupee amount → `Release()` frees the inventory → booking moves to `CANCELLED`.

**Key decision:** `CalculateRefund` takes the whole `booking` and returns a final rupee `Money` amount — not a bare percentage. A percentage-only interface would silently assume every policy is percentage-shaped; a flat-fee policy ("₹500 cancellation fee, refund the rest") wouldn't fit that shape at all. Returning the final amount keeps the interface honest about not knowing how a policy computes its answer. This is the same seam pattern applied a third time, after `AvailabilityService` and `PaymentMethod` — worth calling out explicitly as a consistent design choice across the system.

---

## Unit 6 — Search & discovery

**Interface (seam), applied uniformly across every filter:**

```
Filter:
  Matches(property) → bool
```

Every filter — no matter what it checks — reduces to one yes/no decision per property. This is the Specification pattern: search runs the full property list through each active filter in sequence, keeping only the survivors, then feeds those survivors into the next filter. By the end, only properties that passed every filter remain.

**Concrete filters:**

- `CityFilter` — shallow, reads `Property.Location` directly
- `StarRatingFilter` — shallow, reads `Property.StarRating`
- `PropertyAmenityFilter` — shallow, reads `Property.Amenities` (pool, parking — property-level)
- `PriceRangeFilter` — deep, matches if *any* of the property's room types falls in range (price lives on `RoomType`, not `Property` — a property can have both a cheap and an expensive room type)
- `RoomAmenityFilter` — deep, same reasoning as price — matches if any room type has the amenity (AC, TV — room-level)
- `AvailabilityFilter` — deep, the guest's date range is baked in once at construction time (it's fixed for the whole search, so it doesn't need to be re-passed per property); internally calls `AvailabilityService.IsAvailable()` per room type

**Key decision:** despite "shallow" filters (read one field off `Property`) and "deep" filters (traverse into `RoomType`s) doing meaningfully different internal work, all of them expose the identical `Matches(property) → bool` contract. The search loop never needs to know or care which kind a filter is.

**Why this matters for the brief:** adding a brand-new filter later (e.g. "pet-friendly") means writing one new type that implements `Matches(property) → bool` and adding it to the active filter list — **zero changes to the search loop itself.** This is Open/Closed in direct practice, and satisfies the brief's explicit "design so new filters can be added later without reworking search."

---

## Next steps

- Unit 7 — package structure & layering
- Unit 8 — tests + README