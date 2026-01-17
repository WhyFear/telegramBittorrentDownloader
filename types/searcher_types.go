package types

type Torrent struct {
	Category  string `json:"category"`
	Title     string `json:"title"`
	Link      string `json:"link"`
	Torrent   string `json:"torrent"`
	Magnet    string `json:"magnet"`
	Size      string `json:"size"`
	Time      string `json:"time"`
	Seeders   int    `json:"seeders"`
	Leechers  int    `json:"leechers"`
	Downloads int    `json:"downloads"`
}

type SearchResult struct {
	Count int        `json:"count"`
	Data  []*Torrent `json:"data"`
}
