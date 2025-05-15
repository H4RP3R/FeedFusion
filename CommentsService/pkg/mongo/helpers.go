package mongo

import "context"

var MongoTestConf = &Config{
	Host:   "localhost",
	Port:   "27018",
	DBName: "comments_test",
}

// StorageConnect is a helper function that establishes a connection to the predefined test Mongo instance.
// It returns a connected Storage object or an error if connection fails.
func StorageConnect(ctx context.Context) (*Storage, error) {
	conf := MongoTestConf
	db, err := New(ctx, conf)
	if err != nil {
		return nil, ErrConnectDB
	}

	err = db.Ping(ctx)
	if err != nil {
		return nil, ErrDBNotResponding
	}

	return db, nil
}

// RestoreDB drops the "comments" collection to reset the database state.
// WARNING: Use only in tests to avoid data loss.
func RestoreDB(db *Storage) error {
	coll := db.client.Database(db.dbName).Collection("comments")
	return coll.Drop(context.Background())
}
