package domain

type Account struct {
	Id      string  `bson:"_id"`
	Topics  []Topic `bson:"topics"`
	Updated int64   `bson:"updated"`
	Created int64   `bson:"created"`
}
