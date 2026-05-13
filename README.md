# web3-backend-go

Go backend scaffold for Web3 APIs with Gin, PostgreSQL, Redis, and go-ethereum.

## Structure

- `cmd/api`: API entrypoint.
- `internal/config`: Environment-based configuration.
- `internal/database`: PostgreSQL and Redis clients.
- `internal/evm`: EVM RPC client wrapper.
- `internal/server`: HTTP router, middleware, and handlers.
- `migrations`: SQL migrations.

## Run

```sh
cp .env.example .env
docker compose up -d --build
```

Health check:

```sh
curl http://localhost:8502/healthz
```

EVM status and native balance:

```sh
curl "http://localhost:8502/api/v1/chains?page=1&size=30"
curl "http://localhost:8502/api/v1/chains?key=eth&page=1&size=100"
curl "http://localhost:8502/api/v1/chains?key=eth,arb,bas&page=1&size=100"
curl "http://localhost:8502/api/v1/tokens?chainIds=1,56"
curl "http://localhost:8502/api/v1/tokens?chainIds=1,56&address=0x000000000000000000000000000000000000dEaD"
curl http://localhost:8502/api/v1/evm/network
curl "http://localhost:8502/api/v1/evm/balances?addresses=0x0000000000000000000000000000000000000000&asset=USDT"
curl http://localhost:8502/api/v1/evm/balances/0x0000000000000000000000000000000000000000/native
curl http://localhost:8502/api/v1/evm/balances/0x0000000000000000000000000000000000000000/tokens/0x55d398326f99059fF775485246999027B3197955
```

App-compatible local APIs:

```sh
curl "http://localhost:8502/app/available_chains" \
  -H "accept: application/json"

curl "http://localhost:8502/app/available_coins?chain=BSC" \
  -H "accept: application/json"

curl "http://localhost:8502/app/quicknode/broadcast" \
  -H "Content-Type: application/json" \
  -d '{"chain_code":"eth","raw_tx":"0x..."}'
```

API responses use the Android client wrapper shape:

```json
{
  "code": 0,
  "msg": "success",
  "data": {}
}
```

Paginated responses use the same outer wrapper, with pagination inside `data`:

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "page": 1,
    "size": 30,
    "total": 100,
    "list": []
  }
}
```

## Environment

Set `EVM_RPC_URL` in `.env` when you are ready to connect to an Ethereum/EVM RPC provider.
Set `ALCHEMY_API_KEY` before using `/app/quicknode/broadcast`; the endpoint broadcasts through Alchemy and preserves the app response shape:

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "chain_code": "eth",
    "tx_hash": "0x..."
  }
}
```

## Migrations

After containers are running:

```sh
make migrate-up
```
