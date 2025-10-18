package woodcutting

type Tree struct {
	Name          string
	Level         int
	XP            float64
	MembersOnly   bool
}

var Trees = []Tree{
	{Name: "Normal", Level: 1, XP: 25, MembersOnly: false},
	{Name: "Achey", Level: 1, XP: 25, MembersOnly: true},
	{Name: "Oak", Level: 15, XP: 37.5, MembersOnly: false},
	{Name: "Willow", Level: 30, XP: 67.5, MembersOnly: false},
	{Name: "Teak", Level: 35, XP: 85, MembersOnly: true},
	{Name: "Maple", Level: 45, XP: 100, MembersOnly: true},
	{Name: "Hollow", Level: 45, XP: 100, MembersOnly: true},
	{Name: "Mahogany", Level: 50, XP: 125, MembersOnly: true},
	{Name: "Yew", Level: 60, XP: 175, MembersOnly: false},
	{Name: "Magic", Level: 75, XP: 250, MembersOnly: true},
}