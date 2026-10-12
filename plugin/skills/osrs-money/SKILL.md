---
name: osrs-money
description: Make Old School RuneScape money decisions from live Grand Exchange data - item prices, flip margins after the 2% GE tax, buy limits, trade volume, high alch profits, profitable things to make, NPC shop items that resell on the GE, unusual price moves, and what a game update did to prices. Use when the user asks what an item is worth, whether a flip or craft is profitable, how to make money, what to alch, what is crashing or spiking, or how an update moved the market.
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
| NPC shop items that resell on the GE | `shops` / `osrs shops [--item Spade] --json` |
| Items breaking from their recent price | `movers` / `osrs movers [--direction down] --json` |
| Items with reliable hour-of-day or day-of-week cycles | `cycles` / `osrs cycles ["<item>"] [--period weekly] --json` |
| Whether trading a daily cycle actually pays | `backtest` / `osrs backtest ["<item>"] --json` |
| Month-of-year pattern across years | `seasonal` / `osrs seasonal "<item>" --json` |
| What a game update did to prices | `update_impact` / `osrs update-impact "<update title>" --json` |

Item names must match exactly, ignoring case. When `price` fails, its error lists close names; pick the one the user meant or ask.

## Reading the data

- `buy` is the latest instant-buy price and `sell` the latest instant-sell price. `margin` is buying at `sell` and selling at `buy`, after tax. A margin needs patient offers on both sides; say so.
- The GE tax is 2% of the sale price, rounded down, capped at 5m per item. Items under 50 gp and a short list of exempt items (bonds, some tools, teleport tablets, low-level food) pay none.
- Check `volume_24h` and `buy_limit` before recommending anything. A big margin on an item that trades a few times a day is a stale price, not an opportunity. `alchs` and `recipes` already drop items under 500 daily trades; lower `min_volume` only if the user asks.
- `recipes` buys materials at instant-buy and sells at instant-sell. `profit_per_hour` ignores banking and buy limits, so treat it as an upper bound.
- With a player name, `recipes` keeps only recipes the player has the skill levels for. It does not know quest unlocks or tools the player owns.
- `shops` assumes each unit bought raises the shop price by the shop's delta, and counts one inventory for unlimited-stock shops. The cheapest shop may need a quest or area unlock; say the location and let the user judge.
- `movers` scores the latest hour against the item's own 24-hour baseline. A high score with a high `volume_ratio` is a real event; a high score on low volume is often one large trade.
- `cycles` finds each item's usual cheap and dear times on the earlier part of the history and tests them on the later part; trust `wins` of `test_groups`, not the amplitude. `edge` assumes fills at the average price.
- Always run `backtest` before recommending a daily cycle. It fills a buy only when the hour's average instant-sell price reached the limit, caps fills at 10% of volume, and charges tax. Most daily cycles that look good in `cycles` lose money here; say so plainly.
- `seasonal` tests each year against months picked from the other years. Only a few years of data exist (since March 2021), so "3 of 3 years" is suggestive, not proof; adjacent low and high months usually mean one spike, not a season.
- Market scans and `seasonal` download missing history into a local archive the first time, which can take minutes; later runs fetch only new windows.
- `update_impact` compares the same evening hours the day before and the day after an update. It shows correlation, not cause; mention other updates near the same date if relevant.
- Prices move. Give the time of the data when it matters, and never present a margin as guaranteed.
