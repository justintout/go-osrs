package items

type Item struct {
	Examine  string `json:"examine"`
	ID       int    `json:"id"`
	Members  bool   `json:"members"`
	Lowalch  int    `json:"lowalch"`
	Limit    int    `json:"limit"`
	Value    int    `json:"value"`
	Highalch int    `json:"highalch"`
	Icon     string `json:"icon"`
	Name     string `json:"name"`
}
