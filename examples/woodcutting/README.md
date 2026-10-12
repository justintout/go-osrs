# Woodcutting planner

A small web page that plans woodcutting training: which tree to cut from a start level to a target level, and how many logs of each. It uses the `woodcutting` package, which picks the tree with the most XP per log at each level and switches when a better tree unlocks. You can skip trees or start one later than its unlock level.

Look up a player to fill in their current level from the hiscores.

```sh
cd examples
OSRS_USER_AGENT="my-planner - @you on Discord" go run ./woodcutting -addr :8080
```

The plan counts logs, not time: it does not account for how fast each tree is cut. For plans across production skills priced at live Grand Exchange prices, use `osrs train`.
