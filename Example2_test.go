// See TestMain func in main_test.go for code which creates database and calls Setdb()
/*
Example2 stores and retrieves customer sales data
Details:
	creates root level database buckets: customers, sales
	customers bkt contains 1 record for each customer, key is sequential number
	sales bkt contains:
		1 record for each sale, key is sequential number
		cust_ndx bkt which contains 1 record for each sale, key is cust_id + sales_id
*/
package bo

import "fmt"
import "time"

const custKeySize = 4
const salesKeySize = 7

var customerFlds = FldMap{
	"id":   "string",
	"name": "string",
}
var salesFlds = FldMap{
	"id":     "string",
	"custId": "string",
	"date":   "date",
	"amt":    "float",
}
var customers *Table

func Example2() {
	fmt.Println("....... Example2 ..........")

	// create database buckets
	CreateBucket("customers")
	CreateBucket("sales")
	CreateBucket("sales", "cust_ndx")

	// put some data into the database
	createCustomers()
	createSales()

	// load all customer recs into global customers Table from database
	customers = NewTable(customerFlds, NotShared, "customers")
	customers.Load()

	// get sales recs for customer with id="0001"
	// custNdx records are keys only
	// the key is a combination of custId + salesId
	// to select desired sales recs, we need the salesId portion of the key
	// ex key: 00010000002   (cust 0001, salesid 0000002)
	salesCustNdx := NewTable(FldMap{}, NotShared, "sales", "cust_ndx")
	count := salesCustNdx.LoadPrefix("0001") // load all recs where key begins with "0001"

	salesIds := make([]string, count) // container for ids for all sale recs for this customer
	i := 0
	salesCustNdx.Loop(func(key string, rec *Rec) {
		salesId := key[custKeySize:]
		salesIds[i] = salesId
		i++
	})
	sales := NewTable(salesFlds, NotShared, "sales")
	sales.LoadSome(salesIds)
	sales.CreateOrderBy("amt", "amt:d") // sort by amt descending
	sales.Loop(showSale, sales.OrderBy["amt"])

	// Output:
	// ....... Example2 ..........
	// Customer: Lanco, Lisa -- Amt: 350.49 Date: Aug 22, 2016
	// Customer: Lanco, Lisa -- Amt: 35.72 Date: Sep 23, 2017
}

// display info for a sale, including name of related customer
func showSale(key string, salesRec *Rec) {
	customer := customers.GetRec(salesRec.Get("custId"))
	saleDate := salesRec.GetDate("date")
	line := "Customer: %s -- Amt: %.2f Date: %s\n"
	fmt.Printf(line, customer.Get("name"), salesRec.GetFloat("amt"), saleDate.Format("Jan 02, 2006"))
}

// load some customer data into database
func createCustomers() {
	type input struct { // simulated input from external source
		id   string
		name string
	}
	custs := NewTable(customerFlds, NotShared, "customers")
	custs.SetKeySize(custKeySize)
	custs.CreateRecMap()

	data := []input{
		{id: custs.GetNextKey(), name: "Lanco, Lisa"},
		{id: custs.GetNextKey(), name: "Neely, Ned"},
	}
	for _, v := range data {
		custs.AddRec(v.id, ValMap{"id": v.id, "name": v.name})
	}
	tx := StartDBWrite()
	custs.Save(tx)
	CommitDBWrite(tx)
}

// load some sales data into database
func createSales() {
	type input struct { // simulated input from external source
		id     string
		custId string
		date   time.Time
		amt    float64
	}
	sales := NewTable(salesFlds, NotShared, "sales")
	sales.SetKeySize(salesKeySize)
	sales.CreateRecMap()

	// cust_ndx bucket is container for secondary index records
	// key only, no fields (empty FldMap)
	// key is custId + salesId
	custNdx := NewTable(FldMap{}, NotShared, "sales", "cust_ndx")
	custNdx.CreateRecMap()

	saleDate, _ := time.Parse(DateFormat, "2016-08-22")
	data := []input{
		{
			id:     sales.GetNextKey(),
			custId: "0001",
			date:   saleDate,
			amt:    350.49,
		}, {
			id:     sales.GetNextKey(),
			custId: "0001",
			date:   saleDate.AddDate(1, 1, 1),
			amt:    35.72,
		}, {
			id:     sales.GetNextKey(),
			custId: "0002",
			date:   saleDate.AddDate(0, 2, 0),
			amt:    88.33,
		},
	}
	for _, v := range data {
		sales.AddRec(v.id, ValMap{
			"id":     v.id,
			"custId": v.custId,
			"date":   DateToStr(v.date),
			"amt":    FloatToStr(v.amt),
		})
		// add associated custNdx record
		key := v.custId + v.id
		custNdx.AddRec(key)
	}
	tx := StartDBWrite()
	sales.Save(tx)
	custNdx.Save(tx)
	CommitDBWrite(tx)
}
