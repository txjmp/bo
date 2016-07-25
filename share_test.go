package bo

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

var flds = FldMap{
	"id":        "str",
	"name":      "str",
	"birthdate": "date",
	"hits":      "int",
}
var wg sync.WaitGroup

var sharedTbl *Table
var howMany int = 5

func Test_share(t *testing.T) {
	CreateBucket("sharedBkt")
	sharedTbl = NewTable(flds, Shared, "sharedBkt")
	sharedTbl.CreateRecMap()

	wg.Add(3)
	go process1()
	go process2()
	go process3()
	wg.Wait()

	ShowTable(sharedTbl, "Shared Table")
}

func process1() {
	keys := sharedTbl.GetNextKeys(howMany)
	for i := 0; i < len(keys); i++ {
		sharedTbl.StartWrite()
		fmt.Println("process1 write start")
		sharedTbl.AddRec(keys[i], ValMap{
			"id":        keys[i],
			"name":      "Bill",
			"birthdate": DateToStr(time.Now()),
			"hits":      IntToStr(5),
		})
		sharedTbl.EndWrite()
		fmt.Println("process1 write end")
	}
	wg.Done()
}

func process2() {
	count := 0
	for len(sharedTbl.RecMap) < howMany {
		count++
		sharedTbl.StartRead()
		fmt.Println("process2 start, recs=", len(sharedTbl.RecMap))
		sharedTbl.Loop(display)
		// add wait here
		sharedTbl.EndRead()
		fmt.Println("process2 end")
	}
	wg.Done()
	fmt.Println("process2 execute count=", count)
}

func process3() {
	count := 0
	for len(sharedTbl.RecMap) < howMany {
		count++
		sharedTbl.StartRead()
		fmt.Println("process3 start, recs=", len(sharedTbl.RecMap))
		sharedTbl.Loop(display)

		sharedTbl.EndRead()
		fmt.Println("process3 end")
	}
	wg.Done()
	fmt.Println("process3 execute count=", count)
}

func display(key string, rec *Rec) {
	fmt.Println(key, rec.Get("name"))
}
