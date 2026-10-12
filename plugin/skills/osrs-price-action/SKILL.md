---
name: osrs-price-action
description: Investigate why an Old School RuneScape Grand Exchange price moved - spikes, crashes, volume surges, or slow climbs - by combining order-flow data, in-game causes (updates, new item uses, wiki changes), and web search of Reddit, X, Jagex news, and guides. Use when the user asks why an item's price changed, what is driving the market, whether a move is real or one big order, or wants to scan for moves worth explaining.
---

# Investigating OSRS price action

Explain a price move with evidence. Separate what the data shows from what you infer, and say what would confirm or rule out each explanation.

## Tools

Use the `osrs` MCP tools when they are loaded, otherwise `osrs <command> --json`. If `osrs` is missing, tell the user to install it with `go install github.com/justintout/go-osrs/cmd/osrs@latest`. Use your web search tool for the external steps.

| Step | Tool / command |
| --- | --- |
| Find moves worth explaining | `movers` / `osrs movers --json` |
| Hour-of-day or day-of-week cycles and dump times | `cycles` / `osrs cycles ["<item>"] [--period weekly] --json` |
| Month-of-year pattern | `seasonal` / `osrs seasonal "<item>" --json` |
| Order flow by side, single-order bursts, updates in range | `price_history` / `osrs price-history "<item>" --step 5m --json` (then `1h`, `24h` for context) |
| In-game context: uses, sources, alch floor, update mentions, recent wiki edits, search links | `item_context` / `osrs item-context "<item>" --json` |
| What an update did across the market | `update_impact` / `osrs update-impact "<title>" --json` |
| Related items | `price`, `price_history` on items from `used_in` and `made_from` |

## Playbook

1. **Confirm the move.** Run `price_history` at 5m. Read both sides: `buy_*` is trades at the instant-buy price, `sell_*` at the instant-sell price. Note when the move started and whether both sides moved.
2. **Check for a single order.** A `burst` that accounts for most of the move, with `buy_limit_share` at or under 1, while the other side keeps trading at its normal price, is one large order, not a repricing. Check whether it repeats about 4 hours later, when the buy limit resets; repeats mean deliberate accumulation.
3. **Check the timescale.** Run `price_history` at 1h and 24h. A move that holds across both sides for hours or days with rising volume is a real change in supply or demand. Run `cycles "<item>"` (daily and weekly) and `seasonal "<item>"`: a "spike" at the item's usual high hour, weekday, or month may just be its cycle.
4. **Look for in-game causes.** Run `item_context`:
   - `used_in` / `made_from`: a new or changed recipe creates demand. Check those related items for the same move.
   - `update_mentions` with recent dates, and `recent_edits` clustered near the move: content changed. Read the update with `wiki_summary`.
   - `high_alch`: a price near the alch value has a floor; a crash below it will not last.
   - Run `update_impact` for any game update within a few days of the move.
5. **Search outside the game.** Search the web with the item name in quotes and the move's date:
   - Reddit r/2007scape (`item_context.search_links.reddit`), sorted by new
   - X/Twitter for the item and "osrs" (`search_links.x`), including Jagex Mod accounts
   - Jagex news posts and future-update blogs and polls, which move prices before release
   - devtrackers for Jagex Mod comments
   - YouTube or guide sites for a new money-making method featuring the item
   - "ban wave" or bot news when a botted resource dumps
6. **Classify and report.**

## Common drivers and their signatures

| Driver | Signature in the data | Confirm with |
| --- | --- | --- |
| Single large buy or sell, or a misclick | One burst near one buy limit at an off-market price; the other side unchanged; snaps back | Repeats after the 4-hour reset or not; no news |
| Accumulation or merchant pump | Buy-limit-sized bursts every 4 hours; price stepping up on thin volume; then a sharp drop | Discord or Reddit chatter; the pattern in 1h data |
| Update or new use | Both sides move and hold; volume rises on or after an update date; related items move too | `update_mentions`, `update_impact`, the update post |
| Announced future change | Gradual move after a blog, poll, or Q&A, before any update | News post or poll date matches the start |
| Supply shock | Sell-side price falls with heavy sell volume across many windows | Drop-rate change, new source, or a guide popularizing the method |
| Bot ban wave | Sudden scarcity in botted resources (logs, ores, herbs, fish) and the price climbs; or dumps before bans | Ban-wave posts; several botted items moving at once |
| Price floor or ceiling | Price pinned at high alch or a shop price | `high_alch`, `shops` |
| Daily cycle | The same low and high UTC hours most days; selling bunched in a few hours (`dump_share` well above 12.5%) | `cycles` held-out wins; the dump hours line up with the low hour |
| Thin market noise | Big percentage moves on a few trades | Low `volume_24h`; few trades per window |

## Report shape

- **What happened:** the move in numbers, with times in UTC, and whether it was one order or a repricing.
- **Most likely driver:** with a confidence of low, medium, or high, and the evidence for it.
- **Ruled out:** the drivers checked and why they don't fit.
- **What would confirm it:** a concrete next signal and when to check, such as the next buy-limit reset.
- **Sources:** links for every external claim.

Never state a player's motive as fact. Without external evidence, describe the order and list the plausible reasons.
