package domain

type Space struct {
	Id      string `bson:"_id"`
	Author  string `bson:"author"`
	Created int64  `bson:"created"`
}
