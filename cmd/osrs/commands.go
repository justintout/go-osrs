package main

func commands() []command {
	return []command{
		hiscoresCmd, accountCmd, compareCmd, leaderboardCmd,
		priceCmd, guidePriceCmd, alchsCmd, recipesCmd,
		wikiItemCmd, wikiMonsterCmd, dropsCmd, wikiSearchCmd, wikiSummaryCmd,
		womPlayerCmd, womGainsCmd, womUpdateCmd, womGroupCmd, womCompetitionCmd,
		questCmd, questsCmd,
		oddsCmd, xpCmd, combatCmd,
	}
}
