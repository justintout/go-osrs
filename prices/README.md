# prices

A client for the [OSRS Wiki Real-time Prices API](https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices).

The Wiki team requires a [descriptive user agent](https://oldschool.runescape.wiki/w/RuneScape:Real-time_Prices#Please_set_a_descriptive_User-Agent!).

```go
c := prices.NewClient("my-price-checker - @username on Discord")
spread, err := c.LatestFor(ctx, 20056)
if err != nil {
	return err
}
fmt.Printf("Ale of the gods: buy %d, sell %d\n", spread.High, spread.Low)
```

Options select Deadman Reborn or Fresh Start World prices, set a custom HTTP client, or point at another base URL.
