package items

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type priceResponse struct {
	Data Prices `json:"data"`
}

type Prices map[int]Spread

// unfortunately the map key is the item id as a string
// so we'll implement a custom unmarshaller to always receive it
// as an int
func (p *Prices) UnmarshalJSON(data []byte) error {
	fmt.Println(string(data))
	var items map[string]Spread
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	*p = make(Prices)
	for k, v := range items {
		id, err := strconv.Atoi(k)
		if err != nil {
			return err
		}
		(*p)[id] = v
	}
	return nil
}

type Spread struct {
	High     int       `json:"high,omitempty"`
	HighTime time.Time `json:"highTime,omitempty"`
	Low      int       `json:"low,omitempty"`
	LowTime  time.Time `json:"lowTime,omitempty"`
}

type rawSpread struct {
	High     int
	HighTime int
	Low      int
	LowTime  int
}
