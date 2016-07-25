package bo

import (
	"bytes"
	"fmt"
	"github.com/boltdb/bolt"
	"sort"
	"strings"
	"sync"
)

const Shared = true
const NotShared = false

var DefaultKeySize = "%08d"

type FldMap map[string]string // fieldName=type (str, int, float, date, dateTime, bool, bytes)

// --- Table type --------------------------------------------------------------
// Contains a collection of Recs and OrderBy slices for retrieving records in a specific order.
// Each OrderBy contains RecMap keys in the desired order.
type Table struct {
	Lock    sync.RWMutex
	Shared  bool // set to true if multiple goroutines can access simultaneously (unless all readonly)
	KeySize string
	BktPath []string
	Flds    FldMap              // used to validate fldName and type for sorting
	RecMap  map[string]*Rec     // key is record's database key
	OrderBy map[string][]string // key indicates field order (ex. partno)
}

func (this *Table) StartRead() {
	if this.Shared {
		this.Lock.RLock()
	}
}
func (this *Table) EndRead() {
	if this.Shared {
		this.Lock.RUnlock()
	}
}
func (this *Table) StartWrite() {
	if this.Shared {
		this.Lock.Lock()
	}
}
func (this *Table) EndWrite() {
	if this.Shared {
		this.Lock.Unlock()
	}
}

func (this *Table) GetRec(key string) *Rec {
	rec := this.RecMap[key]
	return rec
}

// AddRec adds entry to table's RecMap, optional valMap sets Rec values.
func (this *Table) AddRec(key string, valMap ...ValMap) *Rec {
	if len(valMap) > 0 {
		this.RecMap[key] = &Rec{Tbl: this, Vals: valMap[0]}
	} else {
		this.RecMap[key] = &Rec{Tbl: this, Vals: make(ValMap)}
	}
	this.RecMap[key].Vals["#c"] = "1" // turn on change flag for Save method
	return this.RecMap[key]
}

// DeleteRec marks rec for deletion, when Save method is executed.
func (this *Table) DeleteRec(key string) {
	this.RecMap[key].Vals["#delete"] = "1"
}

// GetNetKey returns bucket's NextSequence value as a zero prefixed string "00012".
func (this *Table) GetNextKey() string {
	bktPath := this.BktPath
	var nextKey uint64
	db.Update(func(tx *bolt.Tx) error {
		bkt := OpenBucket(tx, bktPath)
		nextKey, _ = bkt.NextSequence()
		return nil
	})
	return fmt.Sprintf(this.KeySize, nextKey)
}

func (this *Table) GetNextKeys(count int) []string {
	bktPath := this.BktPath
	keys := make([]string, count)
	var nextKey uint64
	db.Update(func(tx *bolt.Tx) error {
		bkt := OpenBucket(tx, bktPath)
		for i := 0; i < count; i++ {
			nextKey, _ = bkt.NextSequence()
			keys[i] = fmt.Sprintf(this.KeySize, nextKey)
		}
		return nil
	})
	return keys
}

// CreateOrderBy creates slice of rec key values in sorted order.
// The orderByName is used to reference the result. Ex: tbl.OrderBy[orderByName]
// The sortBy values are names of fields to be sorted.
// If a sortBy field name ends with ":d" or ":desc", this fld will be sorted in descending order
func (this *Table) CreateOrderBy(orderByName string, sortBy ...string) {
	this.StartWrite()
	sorted := make(sortRecs, 0, len(this.RecMap))
	for key, rec := range this.RecMap {
		srtRec := &sortRec{
			recKey: key,
			vals:   make([]sortVal, len(sortBy)),
		}
		for i, fldName := range sortBy {
			sepNdx := strings.Index(fldName, ":d") // look for descending flag
			if sepNdx > -1 {
				fldName = fldName[:sepNdx]
				srtRec.vals[i].direction = "desc"
			} else {
				srtRec.vals[i].direction = "asc"
			}
			valType := this.Flds[fldName]
			srtRec.vals[i].valType = valType
			switch valType {
			case "int":
				srtRec.vals[i].val = rec.GetInt(fldName)
			case "float":
				srtRec.vals[i].val = rec.GetFloat(fldName)
			default:
				srtRec.vals[i].val = rec.Get(fldName)
			}
		}
		sorted = append(sorted, srtRec)
	}
	sort.Sort(sorted)
	this.OrderBy[orderByName] = make([]string, len(sorted))
	for i, v := range sorted {
		this.OrderBy[orderByName][i] = v.recKey
	}
	this.EndWrite()
}

// Load loads Table.RecMap with all db records from specified bucket
// RecMap is recreated, so existing entries are lost
// If loading from a nested bucket, specify path to it
func (this *Table) Load() int {
	this.StartWrite()
	this.RecMap = make(map[string]*Rec)
	this.OrderBy = make(map[string][]string)
	keys := make([]string, 0, 100)
	db.View(func(tx *bolt.Tx) error {
		bkt := OpenBucket(tx, this.BktPath)
		cursor := bkt.Cursor()
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			valMap := make(ValMap)
			valMap.fromJson(v)
			key := string(k)
			this.RecMap[key] = &Rec{Tbl: this, Vals: valMap}
			keys = append(keys, key) // Bolt returns keys in sorted order
		}
		this.OrderBy["key"] = keys
		return nil
	})
	this.EndWrite()
	return len(this.RecMap)
}

func (this *Table) Load1(key string) int {
	this.StartWrite()
	this.RecMap = make(map[string]*Rec)
	this.OrderBy = make(map[string][]string)
	db.View(func(tx *bolt.Tx) error {
		bkt := OpenBucket(tx, this.BktPath)
		v := bkt.Get(bs(key))
		if v != nil {
			valMap := make(ValMap)
			valMap.fromJson(v)
			this.RecMap[key] = &Rec{Tbl: this, Vals: valMap}
		}
		return nil
	})
	this.EndWrite()
	return len(this.RecMap)
}

// LoadSome loads Table.RecMap with records where db key matches a key in keys
func (this *Table) LoadSome(keys []string) int {
	this.StartWrite()
	this.RecMap = make(map[string]*Rec)
	this.OrderBy = make(map[string][]string)
	db.View(func(tx *bolt.Tx) error {
		bkt := OpenBucket(tx, this.BktPath)
		for _, key := range keys {
			v := bkt.Get(bs(key))
			if v == nil {
				continue
			}
			valMap := make(ValMap)
			valMap.fromJson(v)
			this.RecMap[key] = &Rec{Tbl: this, Vals: valMap}
		}
		return nil
	})
	this.EndWrite()
	return len(this.RecMap)
}

// LoadRange loads Table.RecMap with all db records where key is in a range, from start to end
// If loading from a nested bucket, specify path to it
func (this *Table) LoadRange(start, end string) int {
	this.StartWrite()
	this.RecMap = make(map[string]*Rec)
	this.OrderBy = make(map[string][]string)
	keys := make([]string, 0, 100)
	db.View(func(tx *bolt.Tx) error {
		bkt := OpenBucket(tx, this.BktPath)
		cursor := bkt.Cursor()
		stop := bs(end)
		for k, v := cursor.Seek(bs(start)); k != nil && bytes.Compare(k, stop) <= 0; k, v = cursor.Next() {
			valMap := make(ValMap)
			valMap.fromJson(v)
			key := string(k)
			this.RecMap[key] = &Rec{Tbl: this, Vals: valMap}
			keys = append(keys, key)
		}
		this.OrderBy["key"] = keys
		return nil
	})
	this.EndWrite()
	return len(this.RecMap)
}

// LoadPrefix loads Table.RecMap with all db records where key begins with prefix
func (this *Table) LoadPrefix(prefix string) int {
	this.StartWrite()
	this.RecMap = make(map[string]*Rec)
	this.OrderBy = make(map[string][]string)
	keys := make([]string, 0, 100)
	db.View(func(tx *bolt.Tx) error {
		bkt := OpenBucket(tx, this.BktPath)
		cursor := bkt.Cursor()
		keyPrefix := bs(prefix)
		for k, v := cursor.Seek(keyPrefix); bytes.HasPrefix(k, keyPrefix); k, v = cursor.Next() {
			valMap := make(ValMap)
			valMap.fromJson(v)
			key := string(k)
			this.RecMap[key] = &Rec{Tbl: this, Vals: valMap}
			keys = append(keys, key)
		}
		this.OrderBy["key"] = keys
		return nil
	})
	this.EndWrite()
	return len(this.RecMap)
}

// Loop reads thru RecMap calling fn for each record.
// Optional orderBy can be used to control order records are accessed.
func (this *Table) Loop(fn func(key string, rec *Rec), orderBy ...[]string) {
	if len(orderBy) > 0 {
		for _, key := range orderBy[0] {
			if this.RecMap[key].Vals["delete"] == "1" {
				continue
			}
			fn(key, this.RecMap[key])
		}
	} else {
		for key, rec := range this.RecMap {
			if rec.Vals["delete"] == "1" {
				continue
			}
			fn(key, rec)
		}
	}
}

// Save writes added/changed/deleted recs in table.RecMap to database
// calling func passes db transaction used for every table that needs to be saved for transaction
// provides all or none (transaction) functionality
// returns number of records added, changed, and deleted in database
func (this *Table) Save(tx *bolt.Tx) int {
	this.StartWrite()
	var count int
	bkt := OpenBucket(tx, this.BktPath)
	for key, rec := range this.RecMap {
		deleteFlag, _ := rec.Vals["#delete"] // #delete is fldname for delete flag
		if deleteFlag == "1" {
			bkt.Delete(bs(key))
			delete(this.RecMap, key) // remove this record from table RecMap
			count++
			continue
		}
		changed, _ := rec.Vals["#c"] // #c is key for change flag field
		if changed == "1" {
			delete(rec.Vals, "#c") // remove change field
			val := rec.Vals.toJson()
			bkt.Put(bs(key), val)
			count++
		}
	}
	this.EndWrite()
	return count
}

func (this *Table) CreateRecMap() {
	this.RecMap = make(map[string]*Rec)
	this.OrderBy = make(map[string][]string)
}

func (this *Table) SetBktPath(bktPath ...string) {
	this.BktPath = bktPath
}

func (this *Table) SetKeySize(size int) {
	this.KeySize = fmt.Sprintf("%s%dd", "%0", size) // size = 5, returns "%05d"
}

// NewTable creates and inits a new Table
// For shared parm use constants Shared, NotShared defined at top.
func NewTable(flds FldMap, shared bool, bktPath ...string) *Table {
	t := &Table{
		Shared:  shared,
		KeySize: DefaultKeySize,
		BktPath: bktPath,
		Flds:    flds,
	}
	return t
}
