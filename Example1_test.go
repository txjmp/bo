// See TestMain func in main_test.go for code which creates database and calls Setdb()
/*
Example1 stores and retrieves weather & accident data by location.
Details:
	creates root level database buckets: locations, weather, accidents
	locations bkt contains 1 record for each location, key is a sequential #
	weather bkt contains 1 bucket for each location
		weather sub bkts contain 1 rec per date for the specific location
		sub bkt name is "loc_" + key of location, ex: "loc_00001"
		key for recs inside sub bkt is date in format of yyyy-mm-dd
	accidents bkt contains 1 bucket for each location
		accidents sub bkts contain 1 rec for each accident for the specific location
		sub bkt name is "loc_" + key of location, ex: "loc_00001"
		key for recs inside sub bkt is date-time in format of yyyy-mm-dd hh:mm:ss
	new data is created and loaded into tables and saved to the database
	an in-memory secondary index is created (see func loadLocNameIndex) to access loc by name
	data is loaded for specfic year using LoadPrefix() & date range using LoadRange()
	data is sorted and displayed in various ways
	data from 1 loc is compared to data from other loc
	data for all locations is merged into a single table, sorted, displayed
*/
package bo

import "fmt"
import "encoding/json"
import "time"
import "strings"

var locationFlds = FldMap{
	"id":        "string",
	"name":      "string",
	"region":    "string",
	"reporters": "bytes", // marshaled from []string
}
var weatherFlds = FldMap{
	"locId":    "string",
	"date":     "date",
	"highTemp": "float",
	"lowTemp":  "float",
	"precip":   "float",
}
var accidentFlds = FldMap{
	"locId":          "string",
	"dateTime":       "dateTime",
	"primaryRoadway": "string",
	"severity":       "int",
}

var locationTbl *Table

var locNameIndex map[string]string // key=loc name, val=loc key

func Example1() {
	fmt.Println("....... Example1 ..........")

	// create database root level buckets
	CreateBucket("locations")
	CreateBucket("weather")
	CreateBucket("accidents")

	createLocations() // add some location data to the database

	// load global locationTbl so other funcs can use
	locationTbl = NewTable(locationFlds, NotShared, "locations")
	locationTbl.Load()

	// add some weather, accident data to the database
	createWeatherData()
	createAccidentData()

	loadLocNameIndex() // load secondary index for locations, so name can be used to get the primary key

	fmt.Println("\n------- weather data for Dallas, sorted by precip -------")
	locKey := locNameIndex["dallas"]
	weatherTbl := NewTable(weatherFlds, NotShared, "weather", "loc_"+locKey)
	weatherTbl.Load()
	weatherTbl.CreateOrderBy("precip", "precip")
	//weatherTbl.Loop(showWeatherData, weatherTbl.OrderBy["precip"])

	fmt.Println("\n\n---- accident data for Waco in 2015, sorted by primaryRoadway & severity (descending) ----")
	locKey = locNameIndex["waco"]
	accidentsTbl := NewTable(accidentFlds, NotShared, "accidents", "loc_"+locKey)
	accidentsTbl.LoadPrefix("2015")
	accidentsTbl.CreateOrderBy("road_severity", "primaryRoadway", "severity:d")
	//accidentsTbl.Loop(showAccidentData, accidentsTbl.OrderBy["road_severity"])

	fmt.Println("\n\n---- compare highTemp for Waco & Dallas between 2015-01-01:2015-06-30 sorted by date ----")
	locKey = locNameIndex["dallas"]
	weather1Tbl := NewTable(weatherFlds, NotShared, "weather", "loc_"+locKey)
	weather1Tbl.LoadRange("2015-01-01", "2015-06-30")
	locKey = locNameIndex["waco"]
	weather2Tbl := NewTable(weatherFlds, NotShared, "weather", "loc_"+locKey)
	weather2Tbl.LoadRange("2015-01-01", "2015-06-30")
	// the key for weather recs is date
	//weather1Tbl.Loop(func(key string, rec *Rec) {
	//	highTemp1 := rec.GetFloat("highTemp")
	//	rec2 := weather2Tbl.GetRec(key)
	//	highTemp2 := rec2.GetFloat("highTemp")
	//	fmt.Printf("High Temps on: %s Dallas: %.1f  Waco: %.1f \n", key, highTemp1, highTemp2)
	//}, weather1Tbl.OrderBy["key"])

	fmt.Println("\n\n----- weather data for all locations sorted by highTemp -----")
	weatherTbl = mergeWeather() // create table with weather data for all locations
	weatherTbl.CreateOrderBy("highTemp", "highTemp:d")
	weatherTbl.Loop(showWeatherData, weatherTbl.OrderBy["highTemp"])

	// Output:
	// ....... xExample1 ..........
	// ----- weather data for all locations sorted by highTemp -----

	// 2016-06-24     Waco, highTemp:110.0, lowTemp:80.0, precip:0.15
	// 2016-06-24   Dallas, highTemp:110.0, lowTemp:80.0, precip:0.15
	// 2016-03-26     Waco, highTemp:100.0, lowTemp:70.0, precip:0.00
	// 2016-03-26   Dallas, highTemp:100.0, lowTemp:70.0, precip:0.00
	// 2015-12-27     Waco, highTemp:90.0, lowTemp:60.0, precip:0.70
	// 2015-12-27   Dallas, highTemp:90.0, lowTemp:60.0, precip:0.70
	// 2015-09-28     Waco, highTemp:80.0, lowTemp:50.0, precip:0.30
	// 2015-09-28   Dallas, highTemp:80.0, lowTemp:50.0, precip:0.30
	// 2015-06-30   Dallas, highTemp:70.0, lowTemp:40.0, precip:0.20
	// 2015-06-30     Waco, highTemp:70.0, lowTemp:40.0, precip:0.20
	// 2015-04-01   Dallas, highTemp:60.0, lowTemp:30.0, precip:1.10
	// 2015-04-01     Waco, highTemp:60.0, lowTemp:30.0, precip:1.10
	// 2015-01-01   Dallas, highTemp:50.0, lowTemp:20.0, precip:0.00
	// 2015-01-01     Waco, highTemp:50.0, lowTemp:20.0, precip:0.00}
}

func showWeatherData(key string, rec *Rec) {
	date := rec.Get("date") // display string val of date
	locId := rec.Get("locId")
	locRec := locationTbl.GetRec(locId)
	locName := locRec.Get("name")
	highTemp := rec.GetFloat("highTemp")
	lowTemp := rec.GetFloat("lowTemp")
	precip := rec.GetFloat("precip")
	fmt.Printf("\n%s  %7s, highTemp:%.1f, lowTemp:%.1f, precip:%.2f", date, locName, highTemp, lowTemp, precip)
}
func showAccidentData(key string, rec *Rec) {
	road := rec.Get("primaryRoadway")
	severity := rec.GetInt("severity")
	fmt.Printf("\nkey:%s, primary roadway:%s, severity:%d", key, road, severity)
}

func createLocations() {
	type input struct {
		id        string
		name      string
		region    string
		reporters []string
	}
	tblLoc := NewTable(locationFlds, NotShared, "locations")
	tblLoc.CreateRecMap()

	data := []input{
		{tblLoc.GetNextKey(), "Dallas", "North Texas", []string{"Bob", "Alis"}},
		{tblLoc.GetNextKey(), "Waco", "Central Texas", []string{"Zeke", "Soosie"}},
	}

	for _, v := range data {
		reporterBytes, _ := json.Marshal(v.reporters)
		tblLoc.AddRec(v.id, ValMap{
			"id":        v.id,
			"name":      v.name,
			"region":    v.region,
			"reporters": BytesToStr(reporterBytes),
		})
	}
	tx := StartDBWrite()
	tblLoc.Save(tx)
	CommitDBWrite(tx)

	tblLoc.Load()
	var reporters []string
	tblLoc.Loop(func(key string, rec *Rec) {
		reporterBytes := rec.GetBytes("reporters")
		json.Unmarshal(reporterBytes, &reporters)
		fmt.Println(key, rec.Get("name"), rec.Get("region"), reporters)
	})

	// create weather & accident / location sub buckets
	tblLoc.Loop(func(key string, rec *Rec) {
		CreateBucket("weather", "loc_"+key)
		CreateBucket("accidents", "loc_"+key)
	})
}

func createWeatherData() {
	highTemps := []float64{50, 60, 70, 80, 90, 100, 110}
	lowTemps := []float64{20, 30, 40, 50, 60, 70, 80}
	precips := []float64{0, 1.1, .2, .3, .7, 0, .15}

	for locKey, _ := range locationTbl.RecMap {
		locBktName := "loc_" + locKey
		tblWeather := NewTable(weatherFlds, NotShared, "weather", locBktName)
		tblWeather.CreateRecMap()
		date, _ := time.Parse(DateFormat, "2015-01-01")
		for i := 0; i < len(highTemps); i++ {
			key := DateToStr(date)
			tblWeather.AddRec(key, ValMap{
				"locId":    locKey,
				"date":     DateToStr(date),
				"highTemp": FloatToStr(highTemps[i]),
				"lowTemp":  FloatToStr(lowTemps[i]),
				"precip":   FloatToStr(precips[i]),
			})
			date = date.AddDate(0, 0, 90)
		}
		tx := StartDBWrite()
		tblWeather.Save(tx)
		CommitDBWrite(tx)

		//tblWeather.Load()
		//ShowTable(tblWeather, locBktName+" weather data init load")
	}
}

func createAccidentData() {
	roadways := []string{"hwy 77", "reddy way", "bobcat trail", "lincoln Dr", "hwy 77", "roger Rd"}
	severityCodes := []int64{1, 2, 3, 2, 3, 1}

	for locKey, _ := range locationTbl.RecMap {
		locBktName := "loc_" + locKey
		tblAccidents := NewTable(accidentFlds, NotShared, "accidents", locBktName)
		tblAccidents.CreateRecMap()
		dateTime, _ := time.Parse(DateTimeFormat, "2015-01-01 22:05:15")
		for i := 0; i < len(roadways); i++ {
			key := DateTimeToStr(dateTime)
			tblAccidents.AddRec(key, ValMap{
				"locId":          locKey,
				"dateTime":       DateTimeToStr(dateTime),
				"primaryRoadway": roadways[i],
				"severity":       IntToStr(severityCodes[i]),
			})
			dateTime = dateTime.AddDate(0, 0, 90)
		}
		tx := StartDBWrite()
		tblAccidents.Save(tx)
		CommitDBWrite(tx)

		//tblAccidents.Load()
		//ShowTable(tblAccidents, locBktName+" accident data init load")
	}
}

// creates a secondary index enabling use of loc name to retrieve rec's primary key
func loadLocNameIndex() {
	locNameIndex = make(map[string]string)
	for k, v := range locationTbl.RecMap {
		locName := strings.ToLower(v.Get("name"))
		locNameIndex[locName] = k
	}
}

func mergeWeather() *Table {
	allWeather := NewTable(weatherFlds, NotShared)
	allWeather.CreateRecMap()
	locWeather := NewTable(weatherFlds, NotShared)
	locationTbl.Loop(func(locKey string, locRec *Rec) {
		locWeather.SetBktPath("weather", "loc_"+locKey)
		locWeather.Load()
		for weatherKey, weatherRec := range locWeather.RecMap {
			mergedKey := locKey + "_" + weatherKey // makes all keys in merged table unique
			allWeather.RecMap[mergedKey] = weatherRec
		}
	})
	return allWeather
	//ShowTable(allWeather, "merged weather")
}
