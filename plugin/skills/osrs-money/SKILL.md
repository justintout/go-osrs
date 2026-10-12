---
name: osrs-money
description: Make Old School RuneScape money decisions from live Grand Exchange data - item prices, flip margins after the 2% GE tax, buy limits, trade volume, high alch profits, and profitable things to make with a player's levels. Use when the user asks what an item is worth, whether a flip or craft is profitable, how to make money, or what to alch.
---

# OSRS money decisions

## Tools

Use the `osrs` MCP tools when they are loaded. Otherwise run the CLI with `--json`. If `osrs` is missing, tell the user to install it with `go install github.com/justintout/go-osrs/cmd/osrs@latest`.

| Question | Tool / command |
| --- | --- |
| Live buy/sell price, tax, margin, volume, limit | `price` / `osrs price <item> --json` |
| Official guide price and 30/90/180-day trend | `guide_price` / `osrs guide-price <item> --json` |
| Best items to high alch | `alchs` / `osrs alchs --json` |
| Profitable things to make | `recipes` / `osrs recipes [player] [--skill Crafting] --json` |
| Item stats, alch value, weight | `wiki_item` / `osrs wiki-item <item> --json` |

Item names must match exactly, ignoring case. When `price` fails, its error lists close names; pick the one the user meant or ask.

## Reading the data

- `buy` is the latest instant-buy price and `sell` the latest instant-sell price. `margin` is buying at `sell` and selling at `buy`, after tax. A margin needs patient offers on both sides; say so.
- The GE tax is 2% of the sale price, rounded down, capped at 5m per item. Items under 50 gp and a short list of exempt items (bonds, some tools, teleport tablets, low-level food) pay none.
- Check `volume_24h` and `buy_limit` before recommending anything. A big margin on an item that trades a few times a day is a stale price, not an opportunity. `alchs` and `recipes` already drop items under 500 daily trades; lower `min_volume` only if the user asks.
- `recipes` buys materials at instant-buy and sells at instant-sell. `profit_per_hour` ignores banking and buy limits, so treat it as an upper bound.
- With a player name, `recipes` keeps only recipes the player has the skill levels for. It does not know quest unlocks or tools the player owns.
- Prices move. Give the time of the data when it matters, and never present a margin as guaranteed.
