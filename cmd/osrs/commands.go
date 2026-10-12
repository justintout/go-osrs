package main

func commands() []command {
	return []command{
		hiscoresCmd, accountCmd, leaderboardCmd,
		priceCmd, guidePriceCmd,
		wikiItemCmd, wikiMonsterCmd, dropsCmd, wikiSearchCmd, wikiSummaryCmd,
		womPlayerCmd, womGainsCmd, womUpdateCmd, womGroupCmd, womCompetitionCmd,
		oddsCmd, xpCmd, combatCmd,
	}
}
