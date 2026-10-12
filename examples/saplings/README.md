# Sapling profits

A small web page that compares tree seed prices with the saplings they grow into, using live Grand Exchange prices from the OSRS Wiki. Each row shows the profit per sapling after the 2% GE tax, the profit for a full 4-hour seed buy limit, and how many saplings traded in the last 24 hours, so you can tell whether the market can absorb what you grow.

Prices assume patient offers on both sides: seeds bought at the instant-sell price and saplings sold at the instant-buy price.

```sh
cd examples
OSRS_USER_AGENT="my-saplings - @you on Discord" go run ./saplings -addr :8080
```

Responses are cached in your user cache directory, following the price API's own freshness, so reloading the page does not hit the API again within a minute.

For seasonal timing, the `osrs` CLI can show when seeds are usually cheapest, for example `osrs seasonal "Mahogany seed"`.
