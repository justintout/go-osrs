# go-osrs

Go clients and tools for Old School RuneScape.

```sh
go get github.com/justintout/go-osrs
```

## Packages

- [`prices`](./prices/): OSRS Wiki real-time prices, volumes, timeseries, item lookup by name, and GE tax.
- [`skills`](./skills/): skill names, XP table with virtual levels, and combat level.
- [`hiscores`](./hiscores/): Jagex hiscores for all 10 game modes, account-type inference, and paged leaderboards.
- [`woodcutting`](./woodcutting/): XP table and woodcutting training plans.

Every client takes a `context.Context` and requires a descriptive User-Agent, because the OSRS Wiki and Jagex block generic ones. HTTP errors are `*osrs.StatusError` values that match `osrs.ErrNotFound` and `osrs.ErrRateLimited` with `errors.Is`.

## Examples

The [`examples`](./examples/) module holds small web apps built on the library. It is a separate module, so its dependencies never reach your `go.mod`.

## Development

```sh
make check   # vet, race tests, build examples
make lint    # golangci-lint
```
