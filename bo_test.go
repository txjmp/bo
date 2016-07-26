// These tests store and access data related to rectangles.
// A root level db bucket "shapes" and sub bucket "rects" are created.

package bo

import (
	"fmt"
	"testing"
	"time"
)

var rectFlds = FldMap{
	"color": "str",
	"w":     "int",
	"h":     "int",
}

type input struct {
	color string
	w     int64 // width
	h     int64 // height
}

var testData = []input{
	{"red", 7, 70},
	{"red", 7, 10},
	{"red", 7, 15},
	{"black", 100, 50},
	{"black", 1001, 5},
	{"red", 75, 11},
}
var sortedResult = []input{
	{"black", 100, 50},
	{"black", 1001, 5},
	{"red", 7, 70},
	{"red", 7, 15},
	{"red", 7, 10},
	{"red", 75, 11},
}

var perfFlds = FldMap{
	"id": "str",
	"f1": "str",
	"f2": "str",
	"f3": "str",
	"f4": "str",
	"f5": "str",
}
var tblPerf *Table

func TestMisc(t *testing.T) {
	var seq Sequence
	x := seq.Next()
	if x != "0001" {
		t.Log("Sequence Not Working, ", x)
		t.FailNow()
	}
}

func TestLoad1(t *testing.T) {
	CreateBucket("shapes")
	CreateBucket("shapes", "rects")

	tblRects := NewTable(rectFlds, NotShared, "shapes", "rects")
	tblRects.CreateRecMap()
	for _, in := range testData {
		valMap := ValMap{
			"color": in.color,
			"w":     IntToStr(in.w),
			"h":     IntToStr(in.h),
		}
		id := tblRects.GetNextKey()
		tblRects.AddRec(id, valMap)
	}

	tx := StartDBWrite()
	tblRects.Save(tx)
	CommitDBWrite(tx)

	t.Log("shapes/rects load complete")
}

func TestRecGet1(t *testing.T) {
	tblRects := NewTable(rectFlds, NotShared, "shapes", "rects")
	tblRects.Load()

	fmt.Println("---- no order ------------------")
	tblRects.Loop(showRectVals)

	fmt.Println("---- key order ------------------")
	tblRects.Loop(showRectVals, "byKey")

	fmt.Println("---- color order ------------------")
	tblRects.CreateOrderBy("byColor", "color")
	tblRects.Loop(showRectVals, "byColor")

	fmt.Println("--- color-width -------------------")
	tblRects.CreateOrderBy("byColorWidth", "color", "w")
	tblRects.Loop(showRectVals, "byColorWidth")

	fmt.Println("--- by color, width, height(descending) -------------------")
	tblRects.CreateOrderBy("byColorWidthHeight", "color", "w", "h:d")
	i := 0
	tblRects.Loop(func(key string, rec *Rec) {
		color := rec.Get("color")
		w := rec.GetInt("w")
		h := rec.GetInt("h")
		if color != sortedResult[i].color {
			t.Fail()
		}
		if w != sortedResult[i].w {
			t.Fail()
		}
		if h != sortedResult[i].h {
			t.Fail()
		}
		i++
		fmt.Println(key, color, w, h)
	}, "byColorWidthHeight")
}

func TestRecUpdt1(t *testing.T) {
	tblRects := NewTable(rectFlds, NotShared, "shapes", "rects")
	tblRects.Load()

	r1 := tblRects.GetRec("00000001")
	r1.Set("color", "red1")
	r1.SetInt("w", 111)

	tblRects.DeleteRec("00000002")

	fmt.Println("---- add 100 to w, subtract 1 from h ------------------")
	tblRects.Loop(func(key string, rec *Rec) {
		w := rec.GetInt("w")
		h := rec.GetInt("h")
		rec.SetInt("w", w+100)
		rec.SetInt("h", h-1)
	})
	ShowTable(tblRects, "table update before save")

	tx := StartDBWrite()
	tblRects.Save(tx)
	CommitDBWrite(tx)

	ShowTable(tblRects, "table contents after save")

	tblRects.Load()
	tblRects.Loop(showRectVals, "byKey")
}

func showRectVals(key string, rec *Rec) {
	color := rec.Get("color")
	w := rec.GetInt("w")
	h := rec.GetInt("h")
	fmt.Println(key, color, w, h)
}

// ============================================

func TestDates(t *testing.T) {
	CreateBucket("dates")
	var flds = FldMap{
		"id":        "str",
		"lastDate":  "date",
		"nextDate":  "date",
		"startTime": "dateTime",
		"endTime":   "dateTime",
	}
	type input struct {
		id        string
		lastDate  time.Time
		nextDate  time.Time
		startTime time.Time
		endTime   time.Time
	}
	d1, _ := time.Parse(DateFormat, "2015-10-20")
	d2, _ := time.Parse(DateFormat, "2016-01-01")
	dt1, _ := time.Parse(DateTimeFormat, "2015-10-20 08:30:45")
	dt2, _ := time.Parse(DateTimeFormat, "2016-01-01 20:30:15")
	var data = []input{
		{id: "01", lastDate: d1, startTime: dt1},
		{id: "02", lastDate: d2, startTime: dt2},
	}
	tbl := NewTable(flds, NotShared, "dates")
	tbl.CreateRecMap()
	for _, v := range data {
		tbl.AddRec(v.id, ValMap{
			"id":        v.id,
			"lastDate":  DateToStr(v.lastDate),
			"startTime": DateTimeToStr(v.startTime),
		})
	}
	tx := StartDBWrite()
	tbl.Save(tx)
	CommitDBWrite(tx)
	tbl.Load()
	tbl.Loop(func(key string, rec *Rec) {
		rec.SetDate("nextDate", rec.GetDate("lastDate").AddDate(1, 0, 0))
		rec.SetDateTime("endTime", rec.GetDateTime("startTime").Add(7*time.Hour*24))
	})
	tx = StartDBWrite()
	tbl.Save(tx)
	CommitDBWrite(tx)
	// ===========add code to verify values ==============
	ShowTable(tbl, "dates table")
}

func Benchmark_Marshal(b *testing.B) {
	tblPerf = NewTable(perfFlds, NotShared, "perf")
	tblPerf.CreateRecMap()
	keys := tblPerf.GetNextKeys(1000)
	for i := 0; i < len(keys); i++ {
		valMap := ValMap{
			"id": keys[i],
			"f1": "field number 1",
			"f2": "field number 2",
			"f3": "field number 3",
			"f4": "field number 4",
			"f5": "field number 5",
		}
		tblPerf.AddRec(keys[i], valMap)
	}
	tx := StartDBWrite()
	tblPerf.Save(tx)
	CommitDBWrite(tx)

	for n := 0; n < b.N; n++ {
		for _, rec := range tblPerf.RecMap {
			rec.Vals.toJson()
		}
	}
}

func Benchmark_Unmarshal(b *testing.B) {
	valMap := ValMap{
		"id": "1",
		"f1": "field number 1",
		"f2": "field number 2",
		"f3": "field number 3",
		"f4": "field number 4",
		"f5": "field number 5",
	}
	rec1 := Rec{nil, valMap}
	jsonBytes := rec1.Vals.toJson()
	rec2 := Rec{Vals: make(ValMap)}
	for n := 0; n < b.N; n++ {
		rec2.Vals.fromJson(jsonBytes)
	}
}
