package persistence

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

const dbName = "fiber_test"
const mongoURI = "mongodb://woweb:wowebfs22@33339.hostserv.eu:27017/" + dbName

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

func Connect() (MongoInstance, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return MongoInstance{}, err
	}

	mg := MongoInstance{
		Client: client,
		Db:     db,
	}

	return mg, nil

}
