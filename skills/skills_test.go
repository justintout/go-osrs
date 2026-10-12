package skills

import "testing"

func TestXP(t *testing.T) {
	for level, xp := range map[int]int{1: 0, 2: 83, 50: 101_333, 92: 6_517_253, 99: 13_034_431, 126: 188_884_740} {
		if got := XPForLevel(level); got != xp {
			t.Errorf("XPForLevel(%d) = %d, want %d", level, got, xp)
		}
		if got := VirtualLevel(xp); got != level {
			t.Errorf("VirtualLevel(%d) = %d, want %d", xp, got, level)
		}
	}
	if Level(MaxXP) != 99 || VirtualLevel(MaxXP) != 126 || VirtualLevel(82) != 1 {
		t.Error("level caps wrong")
	}
	if XPToLevel(13_034_431, 99) != 0 || XPToLevel(0, 2) != 83 {
		t.Error("XPToLevel wrong")
	}
}

func TestCombatLevel(t *testing.T) {
	tests := []struct {
		name                                     string
		att, str, def, hp, ranged, magic, prayer int
		want                                     int
	}{
		{"new account", 1, 1, 1, 10, 1, 1, 1, 3},
		{"maxed", 99, 99, 99, 99, 99, 99, 99, 126},
		{"pure ranger", 1, 1, 1, 99, 99, 1, 52, 79},
		{"1 def pure", 60, 99, 1, 90, 1, 1, 52, 80},
	}
	for _, tt := range tests {
		if got := CombatLevel(tt.att, tt.str, tt.def, tt.hp, tt.ranged, tt.magic, tt.prayer); got != tt.want {
			t.Errorf("%s: CombatLevel = %d, want %d", tt.name, got, tt.want)
		}
	}
}
