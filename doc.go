// Package osrs holds errors shared by the go-osrs API clients.
//
// API clients:
//   - hiscores: Jagex hiscores, account-type inference, leaderboards
//   - prices: OSRS Wiki real-time prices, volumes, item lookup, GE tax
//   - ge: official Jagex Grand Exchange guide prices and history
//   - wiki: OSRS Wiki items, monsters, drops, recipes, quests, search
//   - wom: Wise Old Man tracking, gains, groups, competitions
//
// Tools built on them:
//   - skills: XP table, virtual levels, combat level
//   - odds: drop chances and expected kills for a set of drops
//   - profit: high alch, recipe, and shop resale profits at live prices
//   - quests: quest requirements across prerequisite trees
//   - bossing: boss GP per hour, hours to pet, hours to collection log
//   - market: price moves, single-order bursts, cycles, update impact
//   - portfolio: investments checked against their plans
//   - runelite: Grand Exchange offers from RuneLite profile files
//   - backtest: strategy replay with conservative fills, tax, buy limits
//   - archive: price history in SQLite, filled on demand
//   - craft: crafting trees down to raw materials
//   - planner: breakpoint-aware training plans by hours, coins, or both
//   - woodcutting: woodcutting tree data and plans
package osrs
