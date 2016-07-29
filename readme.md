#Bo - A Frontend For BoltDB

**status**  
Design looks solid. Hope others will give it a try and provide feedback. Testing modules are a bit of a mess and will be cleaning those up.  
  
**shout out**  
This readme.md file was created with CuteMarkEd, a free Windows app. It sure made the editing process a lot easier.  

**one more thing before getting started**  
Bo normally Panics on error (a reason is displayed). Errors are not generally returned. This decision has good and bad points. Bo requires using lots of method calls. Checking for and handling an error on each one would add significant lines of code to your app. There is some comfort in knowing the program will abort rather than continuing with an unhandled error. Also single return values allow chaining and embedding. I know some apps cannot live with this approach.

[BoltDB](https://github.com/boltdb/bolt) is a simple, fast, reliable key:value database. It is incredibly easy to get up and running, but using it can be a tad tedious. With Bo you can get a lot done with very little code. Its focus is on speed of development. The goal is to reduce stress on the man(or woman) while maybe adding a little more work for the machine. If your creating an app for the world, no Bo. If your creating an app for the neighborhood, Go Bo. 
  
**Snippet**

	members := NewTable(memberFlds, NotShared, "members")
	members.Load()

These 2 lines load all the records in the members bucket into a Table object.  Other methods load a selection of records. Once in a Table, it is very easy to sort, iterate through, add, change, and delete records. The Save method writes changes to the database (unchanged recs are ignored). Related data in other Tables are also easily accessed.
  

##Overview

Bo stores data from BoltDB buckets in objects called Tables. Tables contain a RecMap, map[string]*Rec (map key is db rec key), plus assorted attributes and methods. All data is stored as Recs which are maps with string keys and string values,  map[string]string (map key is field name, map val is string value of field). Rec objects have methods to retrieve/save values in the desired type (converting from/to string if needed).Tables have methods to load and save records from/to a database bucket. Tools are provided to access a table's recs in sorted order which can be based on mutiple fields and include ascending and descending order.
  
##Short Example

	database, _ := bolt.Open("test.db", 0600, nil)
	bo.Setdb(database)

	var circleFlds = bo.FldMap{
		"color":   "str",
		"radius":  "float",
	}
	// shapes bucket is root level, circles bucket is inside shapes
	tblCircles := bo.NewTable(circleFlds, bo.NotShared, "shapes", "circles")
	tblCircles.Load()  // tblCircles now contains all recs in shapes/circles bkt

	// convert radius from inches to centimeters
	var converter float64 = 2.54
	tblCircles.Loop(func(key string, rec *bo.Rec) {
		r := rec.GetFloat("radius") * converter
		rec.SetFloat("radius", r) 	
	})
	tx := bo.StartDBWrite()
	tblCircles.Save(tx)
	bo.CommitDBWrite(tx)

	// show values in order by color, radius (larger values 1st)
	tblCircles.CreateOrderBy("byColorRadius", "color", "radius:d")
	tblCircles.Loop(showCircleVals, "byColorRadius")
	...	
	func showCircleVals(key string, rec *bo.Rec) {
		color := rec.Get("color")
		radius := rec.GetFloat("radius")
		fmt.Println(key, color, radius)
	}

**See following modules for more complete examples** 
 
* Example1_test.go
	* stores and retrieves weather and accident data by location
	* each location has its own database bucket
* Example2_test.go
	* stores and retrieves sales data by customer
	* a single database bucket contains all sales records
	* a separate bucket contains a secondary index for accesssing sales data by customer
  
##A Few Words About BoltDB

* All data records and keys are stored as a slice of bytes, []byte.
* There is no schema or insert/add commands.
	* Simply set a value associated with a key.
	* If a record with the same key exists, the value is replaced.
* Data can be organized into buckets. A bucket can be inside another bucket.
* Records are accessed via the key.
	* use exact match to get a single record
	* use a cursor to establish position (partial key ok) and iterate through records
* Reads are very fast.
* Writes are done as a transaction, with all or none (if error) committed to disk when complete.
* Uses operating system file memory mapping to keep as much of db as possible in memory.

##Here's basically how Bo works: 

1. Create a Table, an in memory object for working with a set of records.
2. Load all or a selection of records from a database bucket into the Table.
3. Access those records in various ways including in sorted order.
4. Add, Change, Delete records in Table.
5. Call Table's Save method to update database with changes.
6. Features are provided to simplify concurrent access to Tables.

To understand Bo, you only have to learn 2 elements.  

1. Table - a container of Recs with associated methods
2. Rec - a container of values for a specific record

##Let's look at Table first. Here is the definition of Table:  
    type FldMap map[string]string
    type Table struct {
        Lock        sync.RWMutex
        Shared      bool 
        KeySize     string
        BktPath     []string
        Flds        FldMap          
        RecMap      map[string]*Rec  
        OrderBy     map[string][]string
    }  

* Lock: used only for shared tables
* Shared: indicates if this table is shared
	* set true if more than 1 goroutine can access simultaneously and 1 is performing writes
	* if all are read only, Shared should be false
* KeySize: used by GetNextKey method
	* contains the number of digits in the key returned by GetNextKey
	* global var DefaultKeySize is used by NewTable() to set the table's keysize
	* to change keysize, use SetKeySize(size int) method
	* value is a string format as used by fmt.Sprintf, ex. "%08d"
	* the app can set a Rec's key to any value, this feature is just for convenience
	* for ordering purposes keys should be 001, 002, 010 not 1, 2, 10
	* all records in a database bucket should have the same KeySize
* BktPath: hierarchy of bucket names leading to desired bucket
	* if root level bucket, then only 1 name is in BktPath
	* used by Load, Save, GetNextKey methods
* Flds: fields defined for recs stored in Table's.RecMap
	* a map where key is field name and val is field's value type
	* used by Rec's Get/Set methods to validate field name
	* used by CreateOrderBy to convert stored string value to appropriate type
	* value types: str, int, float, date, dateTime, bool, bytes
		* int is int64, float is float64
		* date uses var DateFormat (yyyy-mm-dd) when converting between string and time.Time
		* dateTime uses var DateTimeFormat (yyyy-mm-dd hh:mm:ss)
* RecMap: record container
	* key is record's database key value
	* value is pointer to a Rec object
* OrderBy: map of slices of primary key values, each in a particular order
	* key is a name indicating the sort order
	* val is []string where each entry is the key of a Rec in RecMap
	* normally loaded with Table's CreateOrderBy method
  
###Table's Methods

* CreateRecMap()
	* creates RecMap and OrderBy attributes of Table
	* all Load methods automatically perform this function
	* only call this method before manually adding recs to an empty table
	* func NewTable could create the maps, but they are thrown away when a Load is run
* SetKeySize(size int)
	* sets the number of digits in value returned by GetNextKey()
	* ex. KeySize of 8, GetNextKey would return value like: "00000123"
* Load() int
	* loads all recs from a database bucket using tbl.BktPath
	* Table's RecMap and Orderby are recreated; any previous values are lost
	* returns count of recs loaded
	* automatically locks & unlocks table if shared
	* creates tbl.OrderBy["byKey"] for reading RecMap in key order
* LoadPrefix(prefix string) int
	* same as Load, except only loads recs where beginning of key matches prefix  
* LoadRange(start, end string) int
	* same as Load, except only loads recs where key is between start & end
* Load1(key string) int
	* same as Load, except only loads 1 record where db key matches key
	* if key not found, returns 0
	* does not create tbl.OrderBy["key"]	
* LoadSome(keys []string) int
	* same as Load, except loads records where db key matches a value in keys
	* does not create tbl.OrderBy["key"]	
* Save(tx *bolt.Tx) int
	* saves added, changed, deleted records to database using BktPath
	* returns number of records added, changed, or deleted
	* an active bolt transaction must be provided  
	* call StartDBWrite() before 1st Save in transaction and CommitDBWrite after last Save  
	* automatically locks/unlocks table if shared  
	* deleted records are removed from table
	* changed records are no longer marked as changed
* Loop(func(key string, rec *Rec), orderBy string)  
	* reads every record in Table.RecMap, calling func for each one  
	* recs marked as deleted are skipped  
	* optional orderBy specifies key in OrderBy map containing keys in order to be read  
* CreateOrderBy(orderByName string, sortBy ...string)  
	* creates slice of rec keys in order based on sortBy   
	* orderByName is used as OrderBy map key to identify the sort order  
	* sortBy is variable number of field names records are to be sorted by
	* see *More on Sorting* section below  
* GetNextKey() string
	* get a new unique key, used when adding records   
	* returns the next sequential number for BktPath bucket using Bolt's bucket.NextSequence()  
	* Table.KeySize determines length of key (ex. "00001" or "00000001")  
	* returned value is a string number with leading zeros
	* cannot be called inside a StartDBWrite/CommitDBWrite process (creates its own transaction)
	* GetNextKey does not access the Table just the database bucket associated with it
* GetNextKeys(count int) []string
	* same as GetNextKey except returns multiple keys (number determined by count)
	* GetNextKey is fairly expensive process since it performs a database write
	* more efficient than getting 1 key at a time
	* keys returned will never be used again, so don't request large count unless needed
* GetRec(key string) *Rec
	* returns pointer to Rec where key matches key  
	* if key does not match existing key, nil is returned
* AddRec(key string, valMap ...ValMap)
	* add Rec to Table.RecMap
	* key is database key
	* optional valMap contains record values
		* a map[string]string, key is field name, value is string version of value
	* if new empty table, must call CreateRecMap before adding first record 
* DeleteRec(key string)
	* mark record for deletion
	* Loop method will skip over deleted records
* SetBktPath(bktPath ...string)
	* change value of BktPath
	* SetBktPath("root bkt", "level 2 bkt", "target bkt")
* StartRead, EndRead, StartWrite, EndWrite
	* used for Shared tables
	* see section below on Shared tables  
  
**Func NewTable(flds FldMap, shared bool, bktPath ...string) *Table**  
	* returns pointer to a new Table object  
	* bktPath is optional, but must be set before using Load, Save, GetNextKey methods  
	* constants bools Shared, NotShared should be used for shared value  
	* sets KeySize to value of global var DefaultKeySize

##Now Let's See Details on Rec. Here is the definition.
  
    type ValMap map[string]string
    type Rec struct {
        Tbl  *Table
        Vals ValMap
    }

* Tbl - pointer to Table containing this instance of Rec
	* Rec Get/Set Methods access Tbl.Flds to validate fieldnames
	* Set by Table's AddRec method
* Vals - record's values
	* ValMap, key = field name, val = string value
	* All values are stored in Rec's as strings

###Rec Methods

**Get Methods**: Get, GetInt, GetFloat, GetDate, GetDateTime, GetBool, GetBytes

* func signature: func Get???(fld string, defaultVal ...type) type
* fld is the field name of value to be returned
	* fld must be in Tbl.Flds
* defaultVal is optional and is returned if there is no entry for fld
	* its type is the same as the return type
	* fields for which no value has been provided do not have to be loaded
	* if defaultVal is omitted, a reasonable value is returned (ie. 0 for numbers)
* Return type matches the method name (GetInt:int64, GetFloat:float64)
* Get returns the string value as stored
* All others convert the string value to the requested type.  
* GetDate uses value of global var DateFormat to convert string to time.Time value.  
* GetDateTime uses value of global var DateTimeFormat to convert string to time.Time value.  
* GetBytes is for complex values and is discussed below.  

**Set Methods** work pretty much like Get methods. There is a corresponding Set for every Get.

* func signature: Set???(fld string, val type)
* val's type matches the method name
	* SetDate, SetDateTime val is time.Time
	* SetInt, val is int64; SetFloat, val is float64
* all convert val to a string.  
* The existing rec value for fld is replaced with the new string value.  

##More on Sorting

Table's CreateOrderBy method provides a means to access records in sorted order. It creates a slice containing the keys of the records in RecMap in order based on the values of particular fields in the records. A variable number of sortBy fields can be specified. By default, values are sorted in ascending order. To sort a specific field in descending order append ":d" to the field name.  

	Example: CreateOrderBy("bySeverityDate", "severity", "date:d")

Creates Table.OrderBy["bySeverityDate"] containing keys sorted by flds severity and date. Records with the same severity are shown with most recent dates first.

##NOTE

When a table is loaded, the "byKey" orderBy is automatically created. To access records in key order use Table.OrderBy["byKey"].

##Table's Loop Method

There are a couple of ways to read through a Table's records.  
Using RecMap (order is random):  

    for key, rec := range tbl.RecMap {
        fmt.Println(key, rec.Get("name"), rec.GetBool("member"))
    }

Deleted but not saved records will be included.  
Using the Loop method:  

    tbl.Loop(func(key string, rec *Rec) {
        fmt.Println(key, rec.Get("name"), rec.GetBool("member"))
    }, "byKey")
This example reads records in key order.   
func is called once for every record, it can be inline (like example) or a separate function.  
If orderBy is omitted, order is random.
Loop skips over records that have been deleted, but not saved to database.
    
##Storing and Retrieving Complex Types  

Values with complex types such as maps, slices, structs can be stored and retrieved using Bo. For these types Rec GetBytes & SetBytes methods are used.  
Here are the steps to store:

1. Convert value to []bytes using json.Marshal
2. Call SetBytes(fld, bytesVal) - converts to string using base64.EncodeToString  

Here are the steps to retrieve:

* bytesVal = GetBytes(fld) - converts string to []bytes using base64.DecodeString
* convert bytesVal to desired type using json.Unmarshal

##Other Functions, Values & Types

* Setdb(database *bolt.db) - tells Bo what database to use
* StartDBWrite() *bolt.Tx - call before 1st Save in transaction
* CommitDBWrite(tx *bolt.Tx) - call after last Save in transaction
* CreateBucket(bktPath ...string) - creates a new bucket, higher level buckets in path must exist
* BucketExists(bktPath ...string) - returns true if bucket already exists
* ShowTable(tbl *Table, heading string) - displays contents of tbl, preceded with heading
* Funcs to convert non strings to strings (useful when loading a new record's ValMap)
	* BytesToStr, IntToStr, FloatToStr, DateToStr, DateTimeToStr, BoolToStr
	* all use signature: func ???ToStr(val ?type) string
	* returns passed value as a string
* Funcs to convert strings to other types
	* StrToBytes, StrToInt, StrToFloat, StrToDate, StrToDateTime, StrToBool
	* all use signature: func StrTo???(val string) ?type
	* returns passed string val as requested type
	* Date, DateTime both return time.Time, but expect input to match DateFormat or DateTimeFormat
* global Date formats used to parse strings to time.Time values (in bo.go module)
	* var DateFormat = "2006-01-02"
	* var DateTimeFormat = "2006-01-02 15:04:05"
	* string values of dates in these formats will sort properly
* Sequence type - convenient way to get string values in range "0001" to "9999"
	* var lineNo bo.Sequence
	* lineNo.Next() returns next value
	* to reset, lineNo = 0
	* handy for multipart keys, ex. detail records for an order, where key begins with orderId
	* not for concurrent use

##Shared Tables

Tables that can be accessed by more than 1 goroutine at the same time and at least 1 of them may be performing writes should have the Shared attribute = true.

Note: If Shared attribute is not true, locking methods below do nothing when called.

Care must be taken to avoid causing a deadlock. Here are the rules for shared tables:  
For read operations:

 
* before 1st access, call tbl.StartRead() - puts a read lock on table
* after last access, call tbl.EndRead() - removes read lock  

For write operations

* before 1st access, call tbl.StartWrite() - puts a write lock on table
* after last access, call tbl.EndWrite() - removes write lock

##Important Info on Locking Shared Tables
  
Following methods call Table StartWrite / EndWrite methods. These methods cannot be called after the app has begun a StartRead/Write and before the corresponding EndRead/Write or a deadlock will occur.

* CreateOrderBy
* All Load methods
* Save  
  
Following methods do not call Table StartRead/Write & EndRead/Write. **The app is responsible for calling locking methods before and after using them.**  

* Loop
* AddRec
* DeleteRec
* GetRec
* All Rec Get/Set methods 
  
If your code is accessing a shared Table it should be locked before beginning and unlocked when complete (using StartRead or StartWrite, EndRead or EndWrite).  

Go Locking:

* Setting a lock does not prevent other goroutines from accessing data.
* It prevents other goroutines from setting the same lock (unless all are read locks).
* StartRead/Write methods block your code until the lock can be set.

##Database Transactions

* BoltDB allows concurrent read transactions.
* BoltDB does not allow concurrent write transactions.
	* The active one blocks until complete.
* Table Load methods create their own read transaction.
* Table Save method must be passed a write transaction to use.
	* This approach allows multiple Save's to be executed in a single transaction.
	* With Bolt, all writes inside a transaction are committed or none are committed.
	* Call StartDBWrite which returns a *bolt.Tx (Bolt write transaction).
	* Call CommitDBWrite(tx) after all saves in transaction.
* Table GetNextKey creates its own write transaction (bucket sequence number is updated)
* CreateBucket func creates is own write transaction
  
##Performance

* When reading/writing to db, all data is converted between map[string]string and []bytes (json marshal/unmarshal).
* Custom marshal, unmarshal methods are used which are fast.
* Rec Get, GetInt, GetDate, ... methods are used to retrieve a field's value.
	* string values are returned as stored
	* the string value of any type can also be returned as stored
	* returning number and date types requires a conversion step (from string to type)
* Values do not have to be loaded for every field.
	* if there is no entry for a requested field, a default is returned (can specify default)
	* the default value is of the requested type, so no conversion is required
	* for example, number fields that default to 0 (zero)
* Using maps for records provides simplicity and flexibility, but is definitely less efficient than structs.


##Errors

Bo funcs and methods do not generally return error values.  If a problem is detected and a single return value cannot reasonably communicate the error, then Bo calls log.Panic, displaying a message reflecting the reason for abort. With this approach the point of error is clear and removes the possibility of a program continuing to run until a less understandable crash occurs because an error return value was not checked. Methods like Table.GetRec(key) will not abort if the record key is not found, but return a nil pointer value. Methods like Rec.GetInt(fld) will abort if the stored database value cannot be converted to an integer. Beware that some middleware like http/net will recover on panic and the app will keep running. From the Go documentation:  
  
> Panic is a built-in function that stops the ordinary flow of control and begins panicking.  
> When the function F calls panic, execution of F stops, any deferred functions in F are executed normally, and then F returns to its caller.  
> To the caller, F then behaves like a call to panic.  
> The process continues up the stack until all functions in the current goroutine have returned, at which point the program crashes.  

  
##Database Design Strategies

Generally speaking, don't create lots of buckets with a few records in each. For exampe a separate bucket for every order. Keys can be composed of multiple parts which makes it easy to get a set of records with the same key prefix. Example:

* Orders bucket contains 1 rec per order, key is sequential # or customerId + sequence #
* Order_Details bucket contains 1 rec for each item in an order.
	* key is composed of orderId and lineNo
	* ex. 000002330005 - orderId=00000233, lineNo=0005
	* design makes it easy to get order details for an order
* bo.Sequence type is convenient way to get sequence values (see Other funcs, vals, types above)

Be creative. The very simple design of BoltDB may seem limiting, but it encourages creative solutions.

