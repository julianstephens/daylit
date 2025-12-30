package storage

type Settings struct {
	DayStart        string `json:"day_start"`
	DayEnd          string `json:"day_end"`
	DefaultBlockMin int    `json:"default_block_min"`
}
