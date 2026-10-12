// Package osrs holds errors shared by the go-osrs API clients.
//
// The clients live in subpackages:
//   - prices: OSRS Wiki real-time prices, volumes, item lookup, GE tax
//   - ge: official Jagex Grand Exchange guide prices and history
//   - hiscores: Jagex hiscores, account-type inference, leaderboards
//   - wiki: OSRS Wiki items, monsters, drops, search, Bucket queries
//   - wom: Wise Old Man tracking, gains, groups, competitions
//   - odds: drop chances and expected kills for a set of drops
//   - profit: high alch and recipe profits at live prices
//   - quests: quest requirements across prerequisite trees
//   - skills: XP table, virtual levels, combat level
//   - woodcutting: woodcutting XP planning
package osrs
