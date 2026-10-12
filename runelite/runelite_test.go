package runelite

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOffers(t *testing.T) {
	dir := t.TempDir()
	props := `#RuneLite configuration
rsprofile.rsprofile.AbCdEf12.displayName=Main Acct
geoffer.rsprofile.AbCdEf12.2={"itemId"\:22209,"quantitySold"\:50,"totalQuantity"\:500,"price"\:20100,"spent"\:1000000,"state"\:"BUYING"}
geoffer.rsprofile.AbCdEf12.0={"itemId"\:19529,"quantitySold"\:0,"totalQuantity"\:1,"price"\:15000000,"spent"\:0,"state"\:"BUYING"}
other.setting=true
`
	if err := os.WriteFile(filepath.Join(dir, "$rsprofile--1.properties"), []byte(props), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Offers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Slot != 0 || got[1].ItemID != 22209 || got[1].Account != "Main Acct" || got[1].AveragePrice() != 20000 || !got[1].Buy() {
		t.Errorf("offers = %+v", got)
	}
}
