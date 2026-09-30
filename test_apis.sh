#!/usr/bin/env bash
# smoke_test.sh - exercises every endpoint against a running server
# (go run ./cmd, in another terminal), printing each request before its
# response so the output can be read/verified line by line.
#
# Requires: curl, jq

set -e
BASE="http://localhost:8080"

# req <label> <curl args...>
# Prints the request and pretty-printed response to stderr (visible on
# screen), and returns ONLY the raw response body on stdout - so
# `X=$(req ...)` can be piped straight into jq without the display text
# getting mixed in.
req() {
  local label="$1"; shift
  {
    echo
    echo "=== $label ==="
    echo "\$ curl $*"
  } >&2
  local body
  body=$(curl -s "$@")
  echo "$body" | jq . >&2 2>/dev/null || echo "$body" >&2
  echo "$body"
}

echo "############################################"
echo "# HAPPY PATH: onboard -> search -> book -> pay -> cancel"
echo "############################################"

RESP=$(req "1. Create owner" -X POST "$BASE/owners" \
  -d '{"name":"Vee'"'"'s Hotels"}')
OWNER_ID=$(echo "$RESP" | jq -r '.id')

RESP=$(req "2. Add property" -X POST "$BASE/owners/$OWNER_ID/properties" \
  -d '{"name":"Taj Residency","city":"Bengaluru","locality":"MG Road","starRating":4,"amenities":["POOL"]}')
PROPERTY_ID=$(echo "$RESP" | jq -r '.id')

RESP=$(req "3. Add room type" -X POST "$BASE/properties/$PROPERTY_ID/room-types" \
  -d '{"name":"Deluxe","maxOccupancy":2,"basePrice":3000,"amenities":["AC"],"roomCount":2}')
ROOM_TYPE_ID=$(echo "$RESP" | jq -r '.id')

req "4. Search (expect the property just onboarded)" -G "$BASE/search" \
  --data-urlencode "city=Bengaluru" \
  --data-urlencode "checkIn=2027-03-05" \
  --data-urlencode "checkOut=2027-03-08" \
  --data-urlencode "guests=2" > /dev/null

RESP=$(req "5. Create booking (expect status PENDING_PAYMENT, amount 9000)" \
  -X POST "$BASE/bookings" \
  -d "{\"guestId\":\"guest1\",\"roomTypeId\":\"$ROOM_TYPE_ID\",\"checkIn\":\"2027-03-05\",\"checkOut\":\"2027-03-08\",\"guests\":2}")
BOOKING_ID=$(echo "$RESP" | jq -r '.id')

req "6. Pay (expect status CONFIRMED, non-empty transactionId)" \
  -X POST "$BASE/bookings/$BOOKING_ID/pay" \
  -d '{"userId":"guest1"}' > /dev/null

req "7. Cancel (expect status CANCELLED, refundAmount 8100 = 90%)" \
  -X POST "$BASE/bookings/$BOOKING_ID/cancel" > /dev/null

echo
echo "############################################"
echo "# ERROR PATH: booking against a nonexistent room type"
echo "############################################"

echo
echo "=== 8. Create booking with unknown roomTypeId (expect HTTP 404) ==="
echo "\$ curl -i -X POST $BASE/bookings -d '{...roomTypeId: does-not-exist...}'"
curl -i -s -X POST "$BASE/bookings" \
  -d '{"guestId":"guest1","roomTypeId":"does-not-exist","checkIn":"2027-03-05","checkOut":"2027-03-08","guests":2}'
echo