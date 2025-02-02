package domain

type Message struct {
	Tokens   []string `bson:"tokens"`
	Platform Platform
}
