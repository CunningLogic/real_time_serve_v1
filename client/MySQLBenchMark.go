package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/gorp.v1"
)

//database struct
type GprsDBInfo struct {
	Id            uint64  `db:"id"`
	UserID        uint64  `db:"user_id"`
	TeamID        uint32  `db:"team_id"`
	ProVersion    uint16  `db:"version"`
	TimeStamp     uint64  `db:"timestamp"`
	Longi         float64 `db:"longi"`
	Lati          float64 `db:"lati"`
	Alti          float32 `db:"alti"`
	ProductId     string  `db:"product_sn"`
	SprayFlag     uint8   `db:"spray_flag"`
	MotorStatus   uint8   `db:"motor_status"`
	RadarHeight   int     `db:"radar_height"`
	VelocityX     float32 `db:"velocity_x"`
	VelocityY     float32 `db:"velocity_y"`
	FarmDeltaY    float32 `db:"farm_delta_y"`
	FarmMode      uint8   `db:"farm_mode"`
	Pilotnum      uint32  `db:"pilot_num"`
	Sessionnum    uint64  `db:"session_num"`
	FrameIndex    uint8   `db:"frame_index"`
	FlightVersion string  `db:"flight_version"`
	Plant         uint8   `db:"plant"`
	CreateTime    int64   `db:"create_time"`
	WorkArea      uint16  `db:"work_area"`
	Ext1          string  `db:"ext1"`
	Ext2          int64   `db:"ext2"`
}

var g_dbpoolcnt, user, count int

func init() {
	flag.IntVar(&user, "user", 1, "concurrence wirte count")
	flag.IntVar(&count, "count", 1, "how many write time each concurrence")
	flag.IntVar(&g_dbpoolcnt, "db_pool_cnt", 20, "how many connectionss to mysql db")
}

func GetFlightBMap(tableName string, db *sql.DB) *gorp.DbMap {
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.MySQLDialect{"InnoDB", "UTF8"}}
	dbmap.AddTableWithName(GprsDBInfo{}, tableName).SetKeys(true, "Id")
	return dbmap
}

func doInsert(db *sql.DB) error {
	var info GprsDBInfo
	info.CreateTime = time.Now().Unix()
	info.UserID = 1111111
	info.FlightVersion = "1.0.0.1"
	info.Lati = 1.011111
	info.Longi = 2.01111
	info.VelocityX = 0.1111
	info.VelocityY = 0.2222
	dbmp := GetFlightBMap("iuav_flight_data", db)
	return dbmp.Insert(&info)
}

func Writer(db *sql.DB, count int) {
	defer g_wg.Done()
	for i := 0; i < count; i++ {
		if doInsert(db) != nil {
			g_failed_cnt++
		}
	}
}

var g_wg sync.WaitGroup

func main() {
	flag.Parse()
	dbServer := os.Getenv("IUAV_DB_SERVER")
	flightDB, err := sql.Open("mysql", dbServer)
	if err != nil {
		log.Fatal(err)
	}
	err = flightDB.Ping()
	if err != nil {
		log.Fatal(err)
	}
	flightDB.SetMaxOpenConns(g_dbpoolcnt)
	flightDB.SetMaxIdleConns(g_dbpoolcnt/2 + 1)
	g_wg.Add(user)
	st := time.Now()
	for i := 0; i < user; i++ {
		go Writer(flightDB, count)
	}
	g_wg.Wait()
	use := time.Now().Sub(st).Nanoseconds() 
	ms := use/(1000 * 1000)
	sec := use/(1000 * 1000 * 1000)
	fmt.Println("user:", user, "count:", count, "all:", user*count, "failed:", g_failed_cnt, "time:", use, "ms", "speed:", int64(user*count)/ms, "count/ms", int64(user*count)/sec, "count/second")
}
