package config

type Config struct {
	Traffic string `json:"traffic"`
	Count   int    `json:"count"`
	Size    int    `json:"size"`
	Wait    int    `json:"wait"`
}
