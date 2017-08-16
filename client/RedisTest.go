package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/redis.v3"
)

var pool_size, work_cnt, t, user, count int

func init() {
	flag.IntVar(&t, "type", 0, "0=lpush, 1=rpop, 2=go chan")
	flag.IntVar(&work_cnt, "worker", 20, "worker count")
	flag.IntVar(&user, "user", 1, "concurrence wirte count")
	flag.IntVar(&count, "count", 1, "how many write time each concurrence")
	flag.IntVar(&pool_size, "pool", 20, "redis pool size")
}

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

func initData() string {
	var info GprsDBInfo
	info.CreateTime = time.Now().Unix()
	info.UserID = 1111111
	info.FlightVersion = "1.0.0.1"
	info.Lati = 1.011111
	info.Longi = 2.01111
	info.VelocityX = 0.1111
	info.VelocityY = 0.2222
	m, _ := json.Marshal(info)
	return string(m)
}

func initObj() GprsDBInfo {
	var info GprsDBInfo
	info.CreateTime = time.Now().Unix()
	info.UserID = 1111111
	info.FlightVersion = "1.0.0.1"
	info.Lati = 1.011111
	info.Longi = 2.01111
	info.VelocityX = 0.1111
	info.VelocityY = 0.2222
	return info
}

var g_failed_cnt int
var g_wg sync.WaitGroup

func Writer(c *redis.Client, count int) {
	defer g_wg.Done()
	for i := 0; i < count; i++ {
		s := initData()
		queueName := fmt.Sprintf("test_queue_%d", i%work_cnt)
		err := c.LPush(queueName, s).Err()
		if err != nil {
			g_failed_cnt++
		}
	}
}

func Reader(c *redis.Client, count int) {
	defer g_wg.Done()
	for i := 0; i < count; i++ {
		queueName := fmt.Sprintf("test_queue_%d", count%work_cnt)
		_, err := c.RPop(queueName).Result()
		if err != nil {
			g_failed_cnt++
		}
	}
}

func ChanWriter(count int) {
	defer g_wg.Done()
	for i := 0; i < count; i++ {
		obj := initObj()
		g_list <- obj
	}
}

func ChanReader() {
	defer g_wg.Done()
	for {
		select {
		case <-g_list:
			atomic.AddInt64(&g_process_cnt, 1)
		case <-time.After(time.Second * 1):
			return
		}
	}
}

var g_list chan GprsDBInfo
var g_process_cnt int64

func main() {
	flag.Parse()
	//connect redis
	redisServer := os.Getenv("IUAV_REDIS_SERVER")
	redisPass := os.Getenv("IUVA_REDIS_PASS")
	client := redis.NewClient(&redis.Options{
		Addr:     redisServer,
		Password: redisPass,
		PoolSize: pool_size,
		DB:       0,
	})
	_, err := client.Ping().Result()
	if err != nil {
		log.Fatal(err)
	}
	g_wg.Add(user)
	st := time.Now()
	if t == 0 {
		for i := 0; i < user; i++ {
			go Writer(client, count)
		}
	} else if t == 1 {
		for i := 0; i < user; i++ {
			go Reader(client, count)
		}
	} else {
		g_list = make(chan GprsDBInfo, 30000)
		for i := 0; i < user; i++ {
			go ChanWriter(count)
		}

		g_wg.Add(work_cnt)
		for j := 0; j < work_cnt; j++ {
			go ChanReader()
		}
	}
	g_wg.Wait()
	use := time.Now().Sub(st).Nanoseconds()
	ms := use / (1000 * 1000)
	sec := use / (1000 * 1000 * 1000)
	fmt.Println("chan count:", g_process_cnt)
	fmt.Println("user:", user, "count:", count, "all:", user*count, "failed:", g_failed_cnt, "time:", ms, "ms", "speed:", int64(user*count)/ms, "count/ms", int64(user*count)/sec, "count/second")
}
