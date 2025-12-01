package core

type Cluster struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Brokers []string `json:"brokers"`
}
