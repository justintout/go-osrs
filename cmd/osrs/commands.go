package main

func commands() []command {
	return []command{
		hiscoresCmd, accountCmd, compareCmd, leaderboardCmd,
		priceCmd, guidePriceCmd, moversCmd, updateImpactCmd, alchsCmd, recipesCmd, shopsCmd,
		wikiItemCmd, wikiMonsterCmd, dropsCmd, wikiSearchCmd, wikiSummaryCmd,
		womPlayerCmd, womGainsCmd, womUpdateCmd, womGroupCmd, womCompetitionCmd,
		questCmd, questsCmd, trainCmd, craftCmd,
		bossesCmd, oddsCmd, xpCmd, combatCmd,
	}
}
