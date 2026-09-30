# hotel-booking

# Hotel booking system — design decisions

Rupeek SDE-3 machine coding round, Question A. Implementation language: **Go** (brief specifies Java/Spring Boot; adapted idiomatically — see note below).

This document tracks decisions as they're made, unit by unit. No code yet — this is the design record.

---

## 0. Scope

**In scope (core):**

- Onboard an owner with one or many properties, each with room types, amenities, pricing
- Search by city, dates, guests; filter by price, amenities, star rating; return only available properties
- Book, pay, cancel, with a defined booking state machine
- Availability check + atomic reservation; inventory release on cancel
- Basic concurrency safety on shared inventory (bonus in the brief, but treated as core — cheap if the inventory model is right, and a likely live follow-up question)
- Unit tests on core business rules

**Seams only (interface + one implementation):** pricing strategy, cancellation policy, payment methods, repositories.

**Bonus (only after core is solid):** payment idempotency, hold expiry, Swagger/OpenAPI docs.

**Out of scope:** auth/authz, UI/frontend, real payment gateway integration, production-grade persistence.

**Language note:** Go instead of Java/Spring Boot. Interfaces are implicit and small, and conventionally defined at the consumer side, not the implementer side — a stronger demonstration of "seams" than Java's style. No DI framework — layering (handler → service → repository) is done by package structure, done deliberately by hand.

---

## 1. Assumptions (locked)

1. One booking = one room, of one room type. Multi-room bookings are a future extension.
2. Dates are half-open: `[checkIn, checkOut)`. Checkout day itself is not blocked for a new guest.
3. Inventory lives at the room-type level (a count of identical rooms), not at the level of individually numbered rooms.
4. Owner → properties is one level deep. A single-property owner is just an `Owner` with one `Property` — not a separate type.
5. Single currency. Money is a value object, stored as **integer rupees** — no sub-rupee precision. Any discount math rounds to the nearest rupee. *(Trade-off: ₹999.50 is not representable — an accepted simplification for this exercise.)*
6. Kids are treated as adults for occupancy purposes — no separate child-capacity math. A room type has a single `MaxOccupancy` number.
7. Payment failure fails the booking and releases inventory immediately (no pending-retry hold with a TTL — that's a possible future extension).
8. Inventory is reserved at booking creation (state `PENDING_PAYMENT`) and confirmed on payment success.

---

## 2. Domain model (Unit 1)

### Owner

- Identity, name.
- Owns one or more properties (`Properties []Property` or by ID — see Property below).
- A single-property owner is simply an `Owner` with `len(Properties) == 1`. No hard-coded special case.

### Property

- Belongs to exactly one `Owner`.
- Fields: identity, name, location (city/locality), star rating, property-level amenities (pool, parking), and `RoomTypeIDs []string` — **a list of IDs, not embedded RoomType data.**
- **Why IDs, not embedded structs:** other parts of the system (inventory, bookings) need to reference a `RoomType` directly by ID without first locating its parent `Property`. Nesting `RoomType` data inside `Property` would force every such lookup to search through properties. Full `RoomType` data lives in its own store, keyed by ID.

### RoomType

- Belongs to exactly one `Property` — carries `PropertyID` as an explicit back-reference (bidirectional relationship by ID: `Property → RoomTypeIDs` down, `RoomType → PropertyID` up).
- Fields: identity, name (free text — "Single", "Deluxe", "Suite" — **not a hardcoded enum**), max occupancy (single number; see Assumption 6), base price per night, room-level amenities (AC, TV), and a total room count (this count is the room type's total inventory capacity).
- **Why a struct, not an enum:** the brief requires new room types to be addable "with minimal, localised change." An enum baked into the type system requires a code change (new constant + every switch statement branching on it) to add a type. A struct means adding a room type is just an owner creating a new `RoomType` instance at onboarding — zero code change. Single/Double/Duplex/Whole House are seed examples, not an exhaustive set.

### Amenity

- A **closed, typed set of constants** (WiFi, Pool, Parking, AC, TV, ...) — not an enum baked into room type identity, and not open-ended free-form tags.
- **Why constants over free-form tags:** type safety and clean filtering/search outweigh the flexibility loss here. The brief's extensibility requirement is carried by room types, payment methods, and filters — amenities don't need to shoulder that burden too. (Considered and rejected: a hybrid of curated constants + free-form "other tags" field. Deferred as a cheap future add, not needed for the core exercise.)

### Money

- Integer rupees (see Assumption 5).

### DateRange

- `CheckIn` / `CheckOut` pair, half-open `[CheckIn, CheckOut)`. Used identically for search queries and booking stay periods.

---

## 3. Availability / inventory (Unit 2)

This is the highest-risk unit — it's where correctness (no double-booking) and concurrency safety live, and it's the seam the brief most directly tests.

### 3.1 What "availability" actually is

Availability is **not** a property of a `RoomType` alone — a single number like `Deluxe.RoomsRemaining` cannot simultaneously represent "2 free on March 6th" and "2 free on March 20th" if those are two independent, unrelated bookings. Availability is a property of the pair **(RoomType, specific night)**.

### 3.2 Storage model

Store **only booked counts**, not remaining counts, and store **only nights that have actual bookings** (sparse):

```
booked: map[RoomTypeID]map[Date]int   // value = rooms taken on that night
available(roomTypeID, date) = TotalRooms(roomTypeID) - booked[roomTypeID].get(date, default: 0)
```

**Why "booked," not "remaining":** the default for an untouched date is always `0` for every room type — no need to know `TotalRooms` just to write a default entry. "Remaining" would need a per-room-type default (`TotalRooms`), which is circular to populate.

**Why sparse, not a pre-populated calendar:** pre-populating a `date → count` row for every night, for every room type, for months or years ahead, for every property, is enormous and almost entirely wasted storage (nearly every row would just say "fully available," which is already the default). Storing only actual bookings turns that into near-zero storage until real demand exists. *(This is the same idea real systems use: store deltas/exceptions from a default, not the default itself.)*

**Lifecycle of a map entry:**

- Booking a night: increment `booked[roomTypeID][date]` (create the entry at 1 if absent).
- Cancelling a night: decrement; **delete the entry entirely once it reaches 0** — keeps the map genuinely sparse rather than accumulating zero-entries.

### 3.3 A booking's date range → which nights are touched

A booking for `[checkIn, checkOut)` touches every night from `checkIn` up to, but not including, `checkOut`. E.g. March 5–8 touches the 5th, 6th, and 7th only — the 8th is checkout day, and a new guest may check in then without conflict.

### 3.4 Atomicity — the core correctness requirement

**The problem:** if "check availability" and "reserve" are two separate steps (read remaining count, then later decrement), two concurrent requests can each read the same "1 room left" before either has written its decrement — both then decrement, and the count goes negative. This is a genuine double-booking, not just a cosmetic bug.

**The fix:** `Reserve()` must be **one atomic operation**: for every night in the requested range, check `available > 0`; only if *all* nights pass does it decrement *all* of them — as a single critical section, under one lock acquisition.

**Why the whole range must be locked together, not night-by-night:** locking/checking/decrementing one night, releasing, then repeating for the next night allows a different booking to slip in between and take the last room on a later night — leaving the original booking confirmed on some nights and stranded on others. The entire multi-night check-and-reserve must be indivisible.

### 3.5 Lock granularity

**Scope: one lock per `(RoomType, Property)`** — not global, not per-night.

- **Not global:** a single system-wide lock would serialize completely unrelated bookings (Mumbai and Bengaluru, different properties, different room types) that share no data at all — pure wasted contention that would bottleneck the whole platform's booking throughput as it scales, for zero correctness benefit.
- **Not per-night:** would reintroduce the interleaving problem from 3.4 for multi-night bookings.
- **Per (RoomType, Property):** exactly matches the actual contention boundary — two guests only need to wait on each other when they're genuinely competing for the same inventory.

### 3.6 Read/write lock split

Two distinct operations touch this data, with very different risk profiles:

- **`IsAvailable()`** — read-only (used by search/browsing). A stale read here costs, at worst, a minor UX hiccup (guest sees a room listed as available, then finds it gone a moment later) — not a correctness bug.
- **`Reserve()`** — read-then-write (used by booking creation). A race here corrupts data (the double-booking scenario in 3.4).

**Decision: use `sync.RWMutex`, not a plain `sync.Mutex`.** `IsAvailable()` takes `RLock()` (many concurrent readers allowed); `Reserve()` takes the exclusive `Lock()`. Search traffic (high volume, low risk) is never serialized behind booking traffic (low volume, high risk) — a plain mutex would force every search to queue behind every booking attempt, collapsing search throughput to match booking throughput for no correctness reason.

### 3.7 Interface (seam)

The rest of the system never touches the map or the lock directly — only through:

- `IsAvailable(roomTypeID, dateRange) bool` — read-only, used by search
- `Reserve(roomTypeID, dateRange) (bool, error)` — atomic check-and-decrement, used by booking creation
- `Release(roomTypeID, dateRange) error` — inverse of Reserve, used by cancellation

Implemented as a standalone `AvailabilityService` (or similar name) — **not** a method on the `RoomType` struct itself. Keeps `RoomType` a plain data struct (matches "domain logic kept separate from persistence/system concerns"); this is also the seam that would back onto a real datastore with row-level locking later, per the brief's "keep it behind repository interfaces so it could be swapped later."

---

## Still to design

- Unit 3 — booking lifecycle / state machine (what calls `Reserve()`, and what state a booking is created in)
- Unit 4 — payment (method abstraction, mock gateway, outcome driving booking state)
- Unit 5 — cancellation and refund (policy strategy, inventory release via `Release()`)
- Unit 6 — search and discovery (filter abstraction composed with `IsAvailable()`)
- Unit 7 — package structure and layering
- Unit 8 — test strategy and README