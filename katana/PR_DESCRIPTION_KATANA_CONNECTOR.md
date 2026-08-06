# Katana Perps connector

Adds `katana/` — a connector for [Katana Perps](https://api-docs-v1-perps.katana.network/), a
perpetual-futures DEX. Modelled on `hyperliquid/`, the closest existing analogue: also a perps DEX,
also EIP-712 signature-based, same crypto dependencies.

## What Katana is

- **Perpetuals only.** No spot markets. `SpotClient` exists solely to serve account info and the
  WebSocket auth token; it exposes no trading builders, so spot trading is a compile-time
  impossibility rather than a runtime rejection.
- **Cross-margin only, one-way only.** No hedge mode, no margin-mode switch, so there is no
  `NewSetMarginMode` / `NewSetPositionMode` factory.
- Collateral is vbUSDC, surfaced as USDC. Symbols are `ETH-USD` — quoted in **USD, not USDT**.
- **Leverage is derived, not set.** There is no set-leverage endpoint; leverage is
  `1 / initialMarginFraction`, tiered by position size, and "setting" it means submitting an
  initial-margin-fraction override, which can only *tighten*.

## How it works

`NewFuturesClient(apiKey, secretKey, delegatedPrivateKey)` — three credentials, matching the
`okx`/`kucoin`/`bitget` shape. The third is the private key of a **delegated trading key**: a keypair
the custody wallet authorises to trade on its behalf, so the signing key is never the wallet's own.

`SetDemo` moves the base URL **and** the EIP-712 domain together (different `version`, `chainId` and
`verifyingContract`), and `demo` is unexported, so the two cannot drift apart. Call it before
constructing builders — they snapshot the signer at construction.

Every action is a builder from a `New*` factory with a `Do(ctx)`, injected with two client-owned
resolvers that cache for five minutes: `markets()` (`GET /v1/markets`, filtered to active
perpetuals) and `resolveWallet()` (`GET /v1/wallets`, which errors rather than guessing when several
funded wallets are ambiguous).

Requests carry a UUID v1 nonce — the version matters, because Katana reads its time component to
reject stale requests — and an HMAC-SHA256 signature over the exact wire bytes (query string for
GET, raw body otherwise) in `KP-HMAC-SIGNATURE` alongside `KP-API-KEY`.

Write actions carry a **second** signature: EIP-712 over a typed struct, shipped as
`{"parameters": {…}, "signature": "0x…"}`.

Responses unmarshal into `katana*` structs mirroring the wire, and the `account_converts` /
`futures_converts` mappers turn them into the shared `entity.*` types.

## Two deliberate divergences from `hyperliquid/`

**Signing lives on the builder, not in the transport.** The REST body encodes type, side, TIF, STP
and trigger as lowercase strings while the signed struct encodes them as `uint8`. Signing inside
`callAPI` would mean re-deriving the numeric form from the wire string — a second derivation path
that can describe a *different order* than the one being sent. Both forms come from one
`resolveOrderType` / `resolveOrderSide` call instead.

**Debug logging is narrower than every sibling** — no header dump, no `%#v` of the request — because
both carry `KP-API-KEY`.

## Notes for the reviewer

- **Katana has no native amend.** `amendOrder` fully builds and validates the replacement *first*,
  then cancels and re-places, so a validation failure cannot leave the book with a cancelled order
  and no replacement.
- **The `*Hash` wrappers and `signCancelBy*` have no production caller and are not dead.** They back
  golden-vector tests whose hashes and signatures are frozen from the pre-port implementation. For a
  port that is the point: a determinism test cannot catch field-order drift, a frozen hash can.
- Money that cannot be computed is reported **empty rather than as a confident zero**; per-fill fee
  and realised PnL aggregate at 256-bit precision.
- `effectiveIMFForSize`'s ceil-rounding and the override-vs-tier interaction are marked
  `NEEDS LIVE-SANDBOX CONFIRMATION` — the docs specify neither. Ceil is the conservative direction
  (it understates leverage).
- `triggerType` is hardcoded to `last`; no input selects `index`. Assumption, not verified.

## Cross-cutting items, reported not taken

These need a decision from the library owner and are deliberately **not** in this PR:

1. `entity.Futures_OrdersList` has no `reduceOnly`. It is still read internally to invert
   `positionSide`, but cannot be surfaced.
2. `entity.Futures_MarginMode` has no `symbol`.
3. `entity.SignAuthStream` has no `Timestamp`.
4. `utils` retries GETs on 408/429/5xx, which replays the UUID v1 nonce. Bounded — verified GET-only,
   so no write is ever repeated and there is no duplicate-order risk; a read may fail to recover.
5. `onetrades.go` has no `NewKatanaSpot` / `NewKatanaFutures`, and `Options` has no `Demo` field, so
   Katana is unreachable through the root façade. This matches `hyperliquid`, `weex`, `whitebit` and
   `blofin`, none of which are registered there either — confirm that is intentional.

Also house-wide, not fixed here: `defer res.Body.Close()` sits after the `ReadAllBody` error return,
leaking the body on a read error. It is byte-identical in `hyperliquid`, `binance`, `bybit` and
`gateio` — fixing it only in `katana` would create the divergence rather than remove it.

## Verification

`go build ./...`, `go vet ./...`, `gofmt -l katana/` all clean. `go test ./katana/...` — 245 tests
pass.

The root package's `onetrades_test.go` fails on a missing `.env` (`log.Fatalln`). Pre-existing, and
it does not import `katana`.

Not verified: no call has been made against a live Katana sandbox. EIP-712 acceptance, `fromId`
inclusivity, fill-truncation direction and the margin-tier questions above all need a real session.
