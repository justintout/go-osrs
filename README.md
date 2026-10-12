# go-osrs

Go clients and tools for Old School RuneScape.

```sh
go get github.com/justintout/go-osrs
```

## Packages

API clients:

- [`hiscores`](./hiscores/): Jagex hiscores for all 10 game modes, account-type inference, paged leaderboards, and bulk lookups.
- [`prices`](./prices/): OSRS Wiki real-time prices, volumes, timeseries, item lookup by name, and GE tax.
- [`ge`](./ge/): official Jagex Grand Exchange guide prices, trends, and 180-day history.
- [`wiki`](./wiki/): OSRS Wiki items, equipment bonuses, monsters, drop tables, recipes, quests, search, summaries, and raw Bucket queries.
- [`wom`](./wom/): Wise Old Man players, gains, snapshots, achievements, groups, and competitions.

Tools built on them:

- [`skills`](./skills/): skill names, XP table with virtual levels, and combat level.
- [`odds`](./odds/): drop chances, dry-streak rarity, and expected kills to collect a set.
- [`profit`](./profit/): live high alch, recipe, and NPC shop resale profits after tax, filtered by trade volume.
- [`quests`](./quests/): full quest requirements across prerequisite trees, checked against a player's levels.
- [`bossing`](./bossing/): every boss's loot GP per hour, hours to its pet, and hours to finish its collection log.
- [`market`](./market/): items breaking from their own price baseline, single-order bursts, hour-of-day price cycles and bot-dump hours tested on held-out days, and price changes across game updates.
- [`archive`](./archive/): Grand Exchange price history in SQLite, filled only with the windows a caller asks for.
- [`craft`](./craft/): an item's full crafting tree down to raw materials, as text or a Mermaid diagram.
- [`planner`](./planner/): skill training plans that switch methods at unlocks, by fewest actions or best GP per XP, with market caps.
- [`woodcutting`](./woodcutting/): woodcutting tree data and plans.

Every client takes a `context.Context` and requires a descriptive User-Agent, because the OSRS Wiki and Jagex block generic ones. HTTP errors are `*osrs.StatusError` values that match `osrs.ErrNotFound` and `osrs.ErrRateLimited` with `errors.Is`.

## Caching and API etiquette

Clients space requests to each host and wait out short `429 Retry-After` responses. The `prices`, `wiki`, and `wom` clients accept `WithCache`, which stores responses on disk:

- Responses follow the server's `max-age` and `Expires`, then revalidate with `ETag` or `Last-Modified`, so an unchanged response costs a `304`.
- Price averages for windows that have closed never change and stay cached for good.
- Wiki API requests send `maxage` and `smaxage`, so Cloudflare serves repeat queries instead of the wiki's servers. Whole tables are cached for a day.
- Jagex hiscores and catalogue responses are marked `no-cache` and are not stored.

Price history for market scans goes into a SQLite archive (`prices.db`) in the same directory. It downloads only the windows a scan needs and does not already have, so history builds up from use rather than a bulk download.

The `osrs` CLI and MCP server use the cache in your user cache directory. `osrs cache warm` fetches the large wiki tables once; `osrs cache status` and `osrs cache clear` manage it; `OSRS_NO_CACHE=1` turns it off.

## CLI and MCP server

```sh
go install github.com/justintout/go-osrs/cmd/osrs@latest
osrs hiscores Lynx Titan
osrs price abyssal whip
osrs drops Vorkath --json
```

Every command prints a table by default and JSON with `--json`. Run `osrs` for the command list.

`osrs mcp` serves the same commands as MCP tools over stdio. To add it to Claude Code:

```sh
claude mcp add osrs -- osrs mcp
```

Set `OSRS_USER_AGENT` to identify yourself to the APIs, and `WOM_API_KEY` for a higher Wise Old Man rate limit.

## Agent skills

The [`plugin`](./plugin/) directory is a Claude Code plugin with skills that teach an agent to use these tools for player lookups, money-making decisions, investigating price moves, goal planning, and drop odds. It also starts the MCP server. Install the CLI first, then:

```
/plugin marketplace add justintout/go-osrs
/plugin install osrs@go-osrs
```

## Examples

The [`examples`](./examples/) module holds small web apps built on the library. It is a separate module, so its dependencies never reach your `go.mod`.

## Development

```sh
make check   # vet, race tests, build examples
make lint    # golangci-lint
```
