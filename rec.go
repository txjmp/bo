package bo

import (
	"bytes"
	"encoding/base64"
	"log"
	"time"
)

// --- Rec type ---------------------------------------------------
// All data values are stored in the database using this type.
// For each Vals entry, key=field name, val=string.
// Methods are used to get and set values.
// Get methods convert the string value to the requested type.
// Set methods convert the new value to a string.
type ValMap map[string]string
type Rec struct {
	Tbl  *Table
	Vals ValMap
}

// --- Rec Get methods --------------------------------------------
// Get methods have an optional defaultVal parameter.
// If specified, it will be returned if the requested fld is not found.
// If not specified, an appropriate default will be returned.

var SpecialFlds = map[string]string{
	"#c":       "string", // field added to changed or added recs
	"#deleted": "string", // field added to deleted recs
}

func validFld(flds FldMap, fld string) bool {
	if _, found := flds[fld]; found {
		return true
	}
	if _, found := SpecialFlds[fld]; found {
		return true
	}
	return false
}

// Get returns string value for fld.
func (rec Rec) Get(fld string, defaultVal ...string) string {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld:", fld)
	}
	val, found := rec.Vals[fld]
	if !found {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
	}
	return val
}

// GetBytes []byte value for fld.
func (rec Rec) GetBytes(fld string, defaultVal ...[]byte) []byte {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	val, found := rec.Vals[fld]
	if !found {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		noVal := make([]byte, 0)
		return noVal
	}
	return StrToBytes(val)
}

func (rec Rec) GetInt(fld string, defaultVal ...int64) int64 {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	val, found := rec.Vals[fld]
	if !found {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return 0
	}
	return StrToInt(val)
}

func (rec Rec) GetFloat(fld string, defaultVal ...float64) float64 {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	val, found := rec.Vals[fld]
	if !found {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return 0
	}
	return StrToFloat(val)
}

func (rec Rec) GetDate(fld string, defaultVal ...time.Time) time.Time {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	val, found := rec.Vals[fld]
	if !found {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return ZeroDate
	}
	return StrToDate(val)
}

func (rec Rec) GetDateTime(fld string, defaultVal ...time.Time) time.Time {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	val, found := rec.Vals[fld]
	if !found {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return ZeroDate
	}
	return StrToDateTime(val)
}

func (rec Rec) GetBool(fld string, defaultVal ...bool) bool {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	val, found := rec.Vals[fld]
	if !found {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return false
	}
	if val == "true" {
		return true
	} else if val == "false" {
		return false
	}
	log.Fatal("not a valid bool", val)
	return false
}

// --- Rec set methods -----------------------------------------
func (rec Rec) Set(fld, val string) {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	rec.Vals[fld] = val
	rec.Vals["#c"] = "1"
}

func (rec Rec) SetBytes(fld string, val []byte) {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	rec.Vals[fld] = base64.StdEncoding.EncodeToString(val)
	rec.Vals["#c"] = "1"
}

func (rec Rec) SetInt(fld string, val int64) {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	rec.Vals[fld] = IntToStr(val)
	rec.Vals["#c"] = "1"
}

func (rec Rec) SetFloat(fld string, val float64) {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	rec.Vals[fld] = FloatToStr(val)
	rec.Vals["#c"] = "1"
}

func (rec Rec) SetDate(fld string, date time.Time) {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	rec.Vals[fld] = date.Format(DateFormat)
	rec.Vals["#c"] = "1"
}

func (rec Rec) SetDateTime(fld string, dateTime time.Time) {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	rec.Vals[fld] = dateTime.Format(DateTimeFormat)
	rec.Vals["#c"] = "1"
}

func (rec Rec) SetBool(fld string, val bool) {
	if ok := validFld(rec.Tbl.Flds, fld); !ok {
		log.Fatal("invalid fld: ", fld)
	}
	if val {
		rec.Vals[fld] = "true"
	} else {
		rec.Vals[fld] = "false"
	}
	rec.Vals["#c"] = "1"
}

var quote byte = 34 // ascii codes
var comma byte = 44
var colon byte = 58

// toJson encodes ValMap to json -> {"key":"value","key:"value"}
func (this ValMap) toJson() []byte {
	var buf bytes.Buffer
	firstEntry := true
	buf.WriteByte('{')
	for key, val := range this {
		if !firstEntry {
			buf.WriteByte(comma)
		} else {
			firstEntry = false
		}
		buf.WriteByte(quote)
		buf.WriteString(key)
		buf.WriteByte(quote)
		buf.WriteByte(colon)
		buf.WriteByte(quote)
		buf.WriteString(val)
		buf.WriteByte(quote)
	}
	buf.WriteByte('}')
	return buf.Bytes()
}

// fromJson adds entries to ValMap from jsonBytes
func (this ValMap) fromJson(jsonBytes []byte) {
	var offset int     // current position in buffer
	var qx int         // index of next quote
	var begx, endx int // beginning,end indexes of key or val to be extracted
	var key, val string
	for {
		if offset > len(jsonBytes) {
			log.Fatal("ValMap.fromJson, bad json\n", string(jsonBytes))
		}
		// --- get key ----------------------------------------
		qx = bytes.IndexByte(jsonBytes[offset:], quote) // beg quote for key
		if qx == -1 {
			break
		}
		begx = offset + qx + 1
		offset += qx + 1

		qx = bytes.IndexByte(jsonBytes[offset:], quote) // end quote for key
		endx = offset + qx
		key = string(jsonBytes[begx:endx])
		offset += qx + 1

		// --- get value ------------------------------------
		qx = bytes.IndexByte(jsonBytes[offset:], quote) // beg quote for val
		begx = offset + qx + 1
		offset += qx + 1

		qx = bytes.IndexByte(jsonBytes[offset:], quote) // end quote for val
		endx = offset + qx
		val = string(jsonBytes[begx:endx])

		this[key] = val
		offset += qx + 1
	}
}
