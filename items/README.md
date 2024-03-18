# OSRS Real-time Price API Client

> A Go package to query the [OSRS Real-time Price API](https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices)

## Usage

Create a client to query items and prices.
The client must set a descriptive user agent, as [requested by the Wiki team](https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Please_set_a_descriptive_User-Agent!).

```go
c := items.NewClient("my price querier - @USERNAME on Discord")
spread, err := c.LatestFor(20056)
if err != nil {
  panic(err)
}
fmt.Printf("Ale of the gods: %d @ %s\n", spread.High, spread.HighTime.Format(time.RFC3339))
```

`NewClient()` accepts a few functional options to query prices for Deadman Mode or Fresh Start, or pass your own HTTP client.
