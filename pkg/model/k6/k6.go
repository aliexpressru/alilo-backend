package k6

type Response struct {
	Data Data `json:"data"`
}

type Data []struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes struct {
		Type     string      `json:"type"`
		Contains string      `json:"contains"`
		Tainted  interface{} `json:"tainted"`
		Sample   struct {
			Count int     `json:"count"`
			Rate  float64 `json:"rate"`
			Value int     `json:"value"`
			Avg   float64 `json:"avg"`
			Max   float64 `json:"max"`
			Med   float64 `json:"med"`
			Min   float64 `json:"min"`
			P90   float64 `json:"p(90)"`
			P95   float64 `json:"p(95)"`
		} `json:"sample"`
	} `json:"attributes"`
}
