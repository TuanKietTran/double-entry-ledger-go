# Ledger consumer examples

Examples use simulated IDs and localhost only. Provision both accounts explicitly before posting; the ledger never creates accounts as a side effect of a post.

```sh
curl -i -X POST http://localhost:8101/v1/accounts \
  -H 'Content-Type: application/json' \
  -d '{"account_id":"550e8400-e29b-41d4-a716-446655440000","owner_id":"demo-customer","currency":"USD","type":"ASSET"}'

curl -i -X POST http://localhost:8101/v1/accounts \
  -H 'Content-Type: application/json' \
  -d '{"account_id":"550e8400-e29b-41d4-a716-446655440001","owner_id":"demo-merchant","currency":"USD","type":"REVENUE"}'

curl -i -X POST http://localhost:8101/v1/transactions \
  -H 'Content-Type: application/json' \
  -d '{"idempotency_key":"demo-transfer-001","description":"synthetic transfer","expected_versions":{"550e8400-e29b-41d4-a716-446655440000":0,"550e8400-e29b-41d4-a716-446655440001":0},"entries":[{"account_id":"550e8400-e29b-41d4-a716-446655440000","amount":"12.34","direction":"DEBIT","currency":"USD"},{"account_id":"550e8400-e29b-41d4-a716-446655440001","amount":"12.34","direction":"CREDIT","currency":"USD"}]}'

curl http://localhost:8101/v1/accounts/550e8400-e29b-41d4-a716-446655440000
curl 'http://localhost:8101/v1/accounts/550e8400-e29b-41d4-a716-446655440000/balance?as_of=2026-10-18T00:00:00Z'
curl 'http://localhost:8101/v1/accounts/550e8400-e29b-41d4-a716-446655440000/entries?limit=50&offset=0'
curl http://localhost:8101/v1/trial-balance
curl http://localhost:8101/v1/transactions/by-idempotency-key/demo-transfer-001
curl -i -X POST http://localhost:8101/v1/transactions/TRANSACTION_UUID/reversals \
  -H 'Content-Type: application/json' \
  -d '{"idempotency_key":"demo-transfer-001-reversal"}'
```

A same-key, same-payload post returns the original `transaction_id`; same-key, changed payload and stale account versions return HTTP 409. Current legacy validation errors use HTTP 400. Balances and amounts are exact decimal strings; currency is explicit and fixed to the provisioned account currency.
