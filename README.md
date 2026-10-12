# go-osrs

Go clients and tools for Old School RuneScape.

```sh
go get github.com/justintout/go-osrs
```

## Packages

- [`prices`](./prices/): OSRS Wiki real-time prices, volumes, timeseries, item lookup by name, and GE tax.
- [`ge`](./ge/): official Jagex Grand Exchange guide prices, trends, and 180-day history.
- [`wiki`](./wiki/): OSRS Wiki items, equipment bonuses, monsters, drop tables, page search and summaries, and raw Bucket queries.
- [`wom`](./wom/): Wise Old Man players, gains, snapshots, achievements, groups, and competitions.
- [`odds`](./odds/): drop chances, dry-streak rarity, and expected kills to collect a set.
- [`profit`](./profit/): live high alch and recipe profits after tax, filtered by trade volume.
- [`quests`](./quests/): full quest requirements across prerequisite trees, checked against a player's levels.
- [`skills`](./skills/): skill names, XP table with virtual levels, and combat level.
- [`hiscores`](./hiscores/): Jagex hiscores for all 10 game modes, account-type inference, and paged leaderboards.
- [`planner`](./planner/): skill training plans that switch methods at unlocks, by fewest actions or best GP per XP, with market caps.
- [`woodcutting`](./woodcutting/): woodcutting tree data and plans.

Every client takes a `context.Context` and requires a descriptive User-Agent, because the OSRS Wiki and Jagex block generic ones. HTTP errors are `*osrs.StatusError` values that match `osrs.ErrNotFound` and `osrs.ErrRateLimited` with `errors.Is`.

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

The [`plugin`](./plugin/) directory is a Claude Code plugin with skills that teach an agent to use these tools for player lookups, money-making decisions, goal planning, and drop odds. It also starts the MCP server. Install the CLI first, then:

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
