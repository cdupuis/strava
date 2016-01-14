package persistence

import (
	"fmt"
	"os"
	"strconv"

	"github.com/cdupuis/strava/activity-namer/persistence/Godeps/_workspace/src/github.com/boltdb/bolt"
)

type DB struct {
	Store *bolt.DB
}

func (db DB) Reset(count string) {

	err := db.Store.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("actvities"))
		err := b.Put([]byte("commute_counter"), []byte(count))
		return err
	})

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (db DB) Read() string {

	var counter string

	db.Store.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("actvities"))
		counter = string(b.Get([]byte("commute_counter")))
		return nil
	})

	return counter
}

func (db DB) Increment() string {

	counter := db.Read()
	intCounter, err := strconv.ParseInt(counter, 10, 8)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	counter = strconv.FormatInt(intCounter+1, 10)

	db.Store.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("actvities"))
		err := b.Put([]byte("commute_counter"), []byte(counter))
		return err
	})

	return counter
}

func Open() *bolt.DB {

	db, err := bolt.Open(os.Getenv("HOME")+"/.strava/strava.db", 0600, nil)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("actvities"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})

	return db

}

func (db DB) Close() {
	db.Store.Close()
}
