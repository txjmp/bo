package bo

import (
	"fmt"
	"testing"
	"time"
)

func TestStress1(t *testing.T) {
	start := time.Now()
	CreateBucket("stress1")
	var flds = FldMap{
		"id":    "str",
		"date":  "date",
		"amt":   "float",
		"count": "int",
	}
	tbl := NewTable(flds, NotShared, "stress1")
	tbl.CreateRecMap()

	type input struct {
		id    string
		date  time.Time
		amt   float64
		count int64
	}
	data := input{
		date:  time.Now(),
		amt:   1.11,
		count: 123,
	}
	for i := 0; i < 100; i++ {
		keys := tbl.GetNextKeys(100)
		for i := 0; i < len(keys); i++ {
			tbl.AddRec(keys[i], ValMap{
				"id":    keys[i],
				"date":  DateToStr(data.date),
				"amt":   FloatToStr(data.amt),
				"count": IntToStr(data.count),
			})
		}
		tx := StartDBWrite()
		tbl.Save(tx)
		CommitDBWrite(tx)
		//loaded := tbl.Load()
		tbl.Loop(func(key string, rec *Rec) {
			rec.Get("id")
			rec.GetDate("date")
			rec.GetFloat("amt")
			rec.GetInt("count")
		})
		//fmt.Println("stress1 loop count ", i, "recs loaded ", loaded)
	}
	stop := time.Now()
	elapsed := stop.Sub(start)
	fmt.Println("stress1 elapsed: ", elapsed)

	cnt := tbl.Load()
	fmt.Println("stress1 final load count ", cnt)
	var tot int64
	tbl.Loop(func(key string, rec *Rec) {
		tot += rec.GetInt("count")
	})
	fmt.Println("stress1 tot ", tot)
	if tot != 1230000 {
		t.Fatal("stress1 tot wrong, should be 1230000, but is ", tot)
	}
}

func TestStress2(t *testing.T) {
	CreateBucket("stress2")
	var flds = FldMap{
		"id":    "str",
		"date":  "date",
		"amt":   "float",
		"count": "int",
	}
	tbl := NewTable(flds, NotShared, "stress2")
	tbl.CreateRecMap()

	type input struct {
		id    string
		date  time.Time
		amt   float64
		count int64
	}
	data := input{
		date:  time.Now(),
		amt:   1.11,
		count: 123,
	}
	keys := tbl.GetNextKeys(100)
	for i := 0; i < len(keys); i++ {
		tbl.AddRec(keys[i], ValMap{
			"id":    keys[i],
			"date":  DateToStr(data.date),
			"amt":   FloatToStr(data.amt),
			"count": IntToStr(data.count),
		})
	}
	tx := StartDBWrite()
	tbl.Save(tx)
	CommitDBWrite(tx)

	start := time.Now()
	for i := 0; i < 1000; i++ {
		tbl.Load()
		tbl.Loop(func(key string, rec *Rec) {
			rec.Get("id")
			rec.GetDate("date")
			rec.GetFloat("amt")
			rec.GetInt("count")
		})
	}
	stop := time.Now()
	elapsed := stop.Sub(start)
	fmt.Println("stress2 elapsed: ", elapsed)

	var tot int64
	tbl.Loop(func(key string, rec *Rec) {
		tot += rec.GetInt("count")
	})
	fmt.Println("stress2 tot ", tot)
	if tot != 12300 {
		t.Fatal("stress2 tot wrong, should be 12300, but is ", tot)
	}
}
