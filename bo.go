package bo

import (
	"encoding/base64"
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"strconv"
	"time"
)

var db *bolt.DB

var DateFormat = "2006-01-02"
var DateTimeFormat = "2006-01-02 15:04:05"

var ZeroDate time.Time // automatically loaded with a zero date (0001-01-01 00:00:00)

type bs []byte

func Setdb(database *bolt.DB) {
	db = database
}

func StartDBWrite() *bolt.Tx {
	tx, err := db.Begin(true)
	if err != nil {
		log.Panic("StartDBWrite Failed", err)
	}
	return tx
}

func CommitDBWrite(tx *bolt.Tx) {
	err := tx.Commit()
	if err != nil {
		tx.Rollback()
		log.Panic("CommitDBWrite Failed", err)
	}
}

// --- types & methods for sorting --------------------------------------------------------

type sortVal struct {
	direction string // asc or desc
	val       interface{}
	valType   string
}
type sortRec struct {
	recKey string
	vals   []sortVal
}
type sortRecs []*sortRec

func (this sortRecs) Len() int {
	return len(this)
}
func (this sortRecs) Swap(a, b int) {
	this[a], this[b] = this[b], this[a]
}
func (this sortRecs) Less(a, b int) bool {
	var trueResult, falseResult bool
	result := 0
	for i, vala := range this[a].vals {
		if vala.direction == "desc" { // if descending, reverse response
			trueResult = false
			falseResult = true
		} else {
			trueResult = true
			falseResult = false
		}
		switch vala.valType {
		case "int":
			inta := vala.val.(int64)
			intb := this[b].vals[i].val.(int64)
			if inta < intb {
				result = -1
			} else if inta > intb {
				result = 1
			}
		case "float":
			floata := vala.val.(float64)
			floatb := this[b].vals[i].val.(float64)
			if floata < floatb {
				result = -1
			} else if floata > floatb {
				result = 1
			}
		// case "date" works using default string
		default:
			stra := vala.val.(string)
			strb := this[b].vals[i].val.(string)
			if stra < strb {
				result = -1
			} else if stra > strb {
				result = 1
			}
		}
		if result == 0 {
			continue
		}
		break
	}
	if result < 0 { // -1 indicates a < b
		return trueResult
	} else {
		return falseResult
	}
}

// ------------------------------
func OpenBucket(tx *bolt.Tx, bktPath []string) *bolt.Bucket {
	bkt := tx.Bucket(bs(bktPath[0]))
	if bkt == nil {
		log.Panic("OpenBucket failed", bktPath)
	}
	for i := 1; i < len(bktPath); i++ {
		if bkt = bkt.Bucket(bs(bktPath[i])); bkt == nil {
			log.Panic("OpenBucket failed", bktPath)
		}
	}
	return bkt
}

func BucketExists(bktPath []string) bool {
	var bktExists bool
	db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(bs(bktPath[0]))
		if bkt == nil {
			return nil
		}
		for i := 1; i < len(bktPath); i++ {
			if bkt = bkt.Bucket(bs(bktPath[i])); bkt == nil {
				return nil
			}
		}
		bktExists = true
		return nil
	})
	return bktExists
}

// CreateBucket creates a new bucket.
// Last parameter value is new bucket name.
// Preceding values are path above new bucket.
func CreateBucket(bktPath ...string) *bolt.Bucket {
	var bktPointer *bolt.Bucket
	var err error
	db.Update(func(tx *bolt.Tx) error {
		bktName := bktPath[len(bktPath)-1]
		if len(bktPath) > 1 {
			parentBkt := tx.Bucket(bs(bktPath[0]))
			for i := 1; i < (len(bktPath) - 1); i++ {
				parentBkt = parentBkt.Bucket(bs(bktPath[i]))
			}
			bktPointer, err = parentBkt.CreateBucket(bs(bktName))
			if err != nil {
				log.Panic("CreateBucket failed", err, bktPath)
			}
		} else {
			bktPointer, err = tx.CreateBucket(bs(bktName))
			if err != nil {
				log.Panic("CreateBucket failed", err, bktPath)
			}
		}
		return nil
	})
	return bktPointer
}

type Sequence int

// Sequence.Next() returns string values from "0001" to "9999"
func (this Sequence) Next() string {
	this++
	return (fmt.Sprintf("%04d", this))
}

func IntToStr(x int64) string {
	return strconv.FormatInt(x, 10)
}
func FloatToStr(x float64) string {
	return strconv.FormatFloat(x, 'f', -1, 64)
}
func StrToInt(xStr string) int64 {
	x, err := strconv.ParseInt(xStr, 10, 64)
	if err != nil {
		log.Panic("cannot convert str to int, ", xStr)
	}
	return x
}
func StrToFloat(xStr string) float64 {
	x, err := strconv.ParseFloat(xStr, 64)
	if err != nil {
		log.Panic("cannot convert str to float, ", xStr)
	}
	return x
}
func StrToDate(strDate string) time.Time {
	date, err := time.Parse(DateFormat, strDate)
	if err != nil {
		log.Panic("cannot convert string to date", strDate)
	}
	return date
}
func StrToDateTime(strDate string) time.Time {
	date, err := time.Parse(DateTimeFormat, strDate)
	if err != nil {
		log.Panic("cannot convert string to dateTime", strDate)
	}
	return date
}
func DateToStr(date time.Time) string {
	return date.Format(DateFormat)
}
func DateTimeToStr(date time.Time) string {
	return date.Format(DateTimeFormat)
}
func BytesToStr(val []byte) string {
	return base64.StdEncoding.EncodeToString(val)
}
func StrToBytes(val string) []byte {
	valBytes, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		log.Panic("StrToBytes cannot base64 decode string val: ", val)
	}
	return valBytes
}

func ShowTable(tbl *Table, heading string) {
	fmt.Println("\n---- show table: " + heading + " ----")
	fmt.Println("BktPath:", tbl.BktPath, " KeySize:", tbl.KeySize)
	fmt.Println("Flds:", tbl.Flds)
	for recKey, rec := range tbl.RecMap {
		fmt.Print("\nkey:", recKey)
		for fld, val := range rec.Vals {
			fmt.Printf(" %s=%s,", fld, val)
		}
	}
}
