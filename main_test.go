package bo

import (
	"flag"
	"fmt"
	"github.com/boltdb/bolt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	fmt.Println("TestMain Called")
	flag.Parse()
	os.Remove("test.db")
	database, _ := bolt.Open("test.db", 0600, nil)
	defer database.Close()
	Setdb(database)
	CreateBucket("perf") // can't run inside benchmark func (called more than once)
	os.Exit(m.Run())
}
