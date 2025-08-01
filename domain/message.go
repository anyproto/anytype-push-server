package domain

type Message struct {
	Tokens   []string
	Data     map[string]string
	Platform Platform
	Silent   bool
}
