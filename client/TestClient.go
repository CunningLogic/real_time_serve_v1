package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	//"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	//"strings"
	//"strconv"
	"sync"
	"time"
	"math/rand"
)

func loginPackage() []byte {
	token := []byte("21a73750b4ae42ec972540ac68332d9847734451")
	buf := new(bytes.Buffer)
	nLen := uint16(1 + 2 + 1 + 16 + len(token) + 2)
	vl := uint16(1<<10 | nLen)
	sn := []uint8{48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 0, 0, 0, 0, 0, 0}
	//sn := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	binary.Write(buf, binary.LittleEndian, uint8(0x55))
	binary.Write(buf, binary.LittleEndian, vl)
	binary.Write(buf, binary.LittleEndian, uint8(0x01))
	binary.Write(buf, binary.LittleEndian, sn)
	binary.Write(buf, binary.LittleEndian, token)
	binary.Write(buf, binary.LittleEndian, uint16(0xaa))
	//fmt.Println("loginPackage, len: ", len(buf.Bytes()), buf.Bytes())
	return buf.Bytes()
}

func initData(frameindex uint8, latitude float64, longitude float64, start uint64) []byte {
	buf := new(bytes.Buffer)
	vl := uint16(1<<10 | 0x65)
	t := uint64(time.Now().Unix() * 1000)
	var data = []interface{}{
		uint8(0x55),
		vl,
		uint8(0x05),
		t,
		longitude,
		latitude,
		float32(88.33333),
		//[]uint8{65, 66, 67, 68, 69, 70, 71, 72, 73, 73, 74, 75, 76, 77, 79, 78},
		[]uint8{48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 0, 0, 0, 0, 0, 0},
		uint8(0x1),
		uint8(0x0),
		uint16(0x2),
		float32(1.111111),
		float32(2.222222),
		float32(3.33333333),
		uint8(0),
		uint64(0x6666),
		start,
		uint8(frameindex),
		[]uint8{0, 1, 0, 2, 0, 3, 0, 4},
		uint8(0x06),
		uint32(0x37),
		uint16(100),
		uint16(100),
		uint16(0xaa),
	}
	for _, v := range data {
		err := binary.Write(buf, binary.LittleEndian, v)
		if err != nil {
			fmt.Println("binary.Write failed:", err)
		}
	}
	return buf.Bytes()
}

/*type GprsData struct {
	Sof         uint8
	VerLen      uint16
	CmdId       uint8
	NTimeStamp  uint64
	Longi       float64
	Lati        float64
	Alti        float32
	ProductId   [16]uint8
	SprayFlag   uint8
	MotorStatus uint8
	RadarHeight uint16
	VelocityX   float32
	VelocityY   float32
	FarmDeltaY  float32
	FarmMode    uint8
	Pilotnum    uint32
	Sessionnum  uint8
	FrameIndex  uint8
	Reserved    [8]uint8
	Plant       uint8
	CRC         uint16
}*/
type GprsData struct {
	Sof           uint8
	VerLen        uint16
	CmdId         uint8
	NTimeStamp    uint64
	Longi         float64
	Lati          float64
	Alti          float32
	ProductId     [16]uint8
	SprayFlag     uint8
	MotorStatus   uint8
	RadarHeight   uint16
	VelocityX     float32
	VelocityY     float32
	FarmDeltaY    float32
	FarmMode      uint8
	Pilotnum      uint64
	Sessionnum    uint64
	FrameIndex    uint8
	FLightVersion [8]uint8
	Plant         uint8
	TeamID        uint32
	WorkArea      uint16
	FlowSpeed     uint16
	CRC           uint16
}

var wg sync.WaitGroup

func main() {
	//frameindex := 0xf3
	//fmt.Printf("%b, %b, %b", frameindex, frameindex&0x7F, frameindex>>7)

	//frameindex = 0x73
	//fmt.Printf("\n%b, %b, %b\n", frameindex, frameindex&0x7F, frameindex>>7)

	//tsn := []uint8{48, 54, 55, 48, 49, 49, 51, 49, 52, 48, 0, 0, 0, 0, 0, 0}
	//sn := strings.TrimRight(string(tsn), string(0))
	//fmt.Println([]byte(sn), sn)
	if len(os.Args) < 2 {
		fmt.Println("Usage: ", os.Args[0], "host:port [user] [count] [sleepCnt]")
		os.Exit(1)
	}
	var user, count int
	if len(os.Args) > 3 {
		user, _ = strconv.Atoi(os.Args[2])
		count, _ = strconv.Atoi(os.Args[3])
	} else {
		user = 1
		count = 1
	}

	sleepCnt := 1000
	if len(os.Args) > 4 {
		sleepCnt, _ = strconv.Atoi(os.Args[4])
	}
	//fmt.Println("3 seconds", time.Now().Add(time.Second*3).Unix()-time.Now().Unix())
	/*	buf := initData(uint8(1))
		reader := bytes.NewBuffer(buf)
		var s GprsData
		err := binary.Read(reader, binary.LittleEndian, &s)
		fmt.Println("test....")
		fmt.Println(err, s)
		//m, _ := json.Marshal(&s)
		json.Marshal(&s)*/
	//Sfmt.Println(string(m))

	//fmt.Printf("%#v\n", s)
	service := os.Args[1]
	i := 0
	fmt.Println("user: ", user)
	wg.Add(user)
	for i < user {
		go Sender(service, i, count, sleepCnt)
		i++
		time.Sleep(time.Millisecond * 1000)
	}
	wg.Wait()
	fmt.Println("sender user:", user, ", count:", count)
	os.Exit(0)
}
func Reader(conn net.Conn) {
	defer conn.Close()

	var ack [128]byte
	for {
		_, err := conn.Read(ack[0:])
		if err != nil {
			fmt.Println("readError...")
			return
		}
		fmt.Println("reader: ", bytes.TrimRight(ack[:], "\x00"))
		copy(ack[:], " ")
	}
}

func updateLocation(latitude float64, longitude float64) (float64, float64) {
	STEP := 0.00002
	if (rand.Float32() > 0.5) {
		latitude += STEP
	} else {
		latitude -= STEP
	}

	if (rand.Float32() > 0.5) {
		longitude += STEP
	} else {
		longitude -= STEP
	}

	return latitude, longitude
}

func Sender(service string, i, count, sleepCnt int) {
	defer wg.Done()
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", service, conf)
	if err != nil {
		fmt.Println(err, i)
		return
	}
	fmt.Println("connet...", i)
	//checkError(err)
	var ack [16]byte
	lp := loginPackage()
	fmt.Println("loginPackage: ", lp)
	conn.Write(lp)

	conn.Read(ack[0:])
	fmt.Println("loginPackage ack: ", ack, err)

	go Reader(conn) //多个goruntine可以在同一个Conn中调用方法

	start := uint64(time.Now().Unix() * 1000)
	latitude := 22.5370300000
	longitude := 113.9529200000

	for n := 0; n < count-1; n++ {
		//var buf [512]byte
		latitude, longitude = updateLocation(latitude, longitude)
		w := initData(uint8(n+1)%128, latitude, longitude, start)

		var d GprsData
		newbuf := bytes.NewReader(w)
		err = binary.Read(newbuf, binary.LittleEndian, &d)
		//fmt.Println(d)
		//fmt.Println(w)
		_, err := conn.Write(w)
		if err != nil {
			fmt.Println("disconnet...")
			conn.Close()
			break
		}
		//conn.Read(buf[0:])
		//n, err := conn.Read(buf[0:])
		//fmt.Println("receive:", len(buf), buf[0:n], err)
		time.Sleep(time.Millisecond * time.Duration(sleepCnt))
	}
	//发送最后一帧数据
	w := initData(uint8(128+count), latitude, longitude, start)
	var d GprsData
	newbuf := bytes.NewReader(w)
	err = binary.Read(newbuf, binary.LittleEndian, &d)
	fmt.Println(d)
	fmt.Println(w)
	conn.Write(w)
}

func ReadFile(path string) ([]byte, error) {
	fi, err := os.Open(path)
	if err == nil {
		defer fi.Close()
		fc, err := ioutil.ReadAll(fi)
		return fc, err
	} else {
		return []byte(""), err
	}
}
func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
