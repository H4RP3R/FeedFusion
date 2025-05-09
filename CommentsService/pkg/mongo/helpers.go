package mongo

import "context"

var MongoTestConf = Config{
	Host:   "localhost",
	Port:   "27018",
	DBName: "comments",
}

// StorageConnect is a helper function that establishes a connection to the predefined test Mongo instance.
// It returns a connected Storage object or an error if connection fails.
func StorageConnect() (*Storage, error) {
	conf := MongoTestConf
	db, err := New(conf)
	if err != nil {
		return nil, ErrConnectDB
	}

	err = db.Ping()
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
