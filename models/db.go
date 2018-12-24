package models

import (
	"fmt"
	"log"
	"time"

	"github.com/influxdata/influxdb/client/v2"
)

//declares all constants related to the influx database
const (
	DB         = "pw"
	DBusername = "calceamenta"
	DBpassword = "personal_website"
)

//declares all variables related to the influx database
var (
	C client.Client
)

//DBinit initializes the database
func DBinit() {
	fmt.Println("db init function called")
	var err error
	C, err = client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://localhost:8086",
		Username: DBusername,
		Password: DBpassword,
	})
	if err != nil {
		log.Println(err)
	}
}

//DBquery runs a query and returns the response
func DBquery(cmd string) (res []client.Result, err error) {
	q := client.Query{
		Command:  cmd,
		Database: DB,
	}
	if response, err := C.Query(q); err == nil {
		if response.Error() != nil {
			return res, response.Error()
		}
		res = response.Results
	} else {
		return res, err
	}
	return res, nil
}

//DBwrite writes a point to the database
func DBwrite(measurement string, tags map[string]string, fields map[string]interface{}, t time.Time) {
	bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database: DB,
	})
	pt, err := client.NewPoint(measurement, tags, fields, t)
	bp.AddPoint(pt)
	err = C.Write(bp)
	if err != nil {
		log.Println("failed to write a point to the database: ", err.Error())
	}
}
