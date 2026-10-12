package prices

// taxExempt holds the IDs of items the Grand Exchange does not tax, from
// https://oldschool.runescape.wiki/w/Category:Items_exempt_from_Grand_Exchange_tax.
// The wiki lists "Energy potion" without a dose, so every dose is included.
var taxExempt = map[int]bool{
	8011:  true, // Ardougne teleport (tablet)
	365:   true, // Bass
	2309:  true, // Bread
	882:   true, // Bronze arrow
	806:   true, // Bronze dart
	1891:  true, // Cake
	8010:  true, // Camelot teleport (tablet)
	1755:  true, // Chisel
	28824: true, // Civitas illa fortis teleport
	2140:  true, // Cooked chicken
	2142:  true, // Cooked meat
	3014:  true, // Energy potion(1)
	3012:  true, // Energy potion(2)
	3010:  true, // Energy potion(3)
	3008:  true, // Energy potion(4)
	8009:  true, // Falador teleport (tablet)
	3853:  true, // Games necklace(8)
	5325:  true, // Gardening trowel
	1785:  true, // Glassblowing pipe
	2347:  true, // Hammer
	347:   true, // Herring
	884:   true, // Iron arrow
	807:   true, // Iron dart
	28790: true, // Kourend castle teleport (tablet)
	379:   true, // Lobster
	8008:  true, // Lumbridge teleport (tablet)
	355:   true, // Mackerel
	2327:  true, // Meat pie
	558:   true, // Mind rune
	1733:  true, // Needle
	13190: true, // Old school bond
	233:   true, // Pestle and mortar
	351:   true, // Pike
	5341:  true, // Rake
	2552:  true, // Ring of dueling(8)
	329:   true, // Salmon
	8794:  true, // Saw
	5329:  true, // Secateurs
	5343:  true, // Seed dibber
	1735:  true, // Shears
	315:   true, // Shrimps
	952:   true, // Spade
	886:   true, // Steel arrow
	808:   true, // Steel dart
	8013:  true, // Teleport to house (tablet)
	361:   true, // Tuna
	8007:  true, // Varrock teleport (tablet)
	5331:  true, // Watering can
}
