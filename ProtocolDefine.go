package main

import (
  "encoding/binary"
  "strings"
  "fmt"
  "time"
  "github.com/satori/go.uuid"
  "strconv"
)

type PackageHeadInfo struct {
  SOF     uint8
  Ver     uint16
  DataLen uint16
  CmdID   uint8
}

type GpsDataV1 struct {
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
  FlightVersion [8]uint8
  Plant         uint8
  TeamID        uint32
  WorkArea      uint16
  CRC           uint16
}

type GpsDataV2 struct {
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
  FlightVersion [8]uint8
  Plant         uint8
  TeamID        uint32
  WorkArea      uint16
  FlowSpeed     uint16   // v2 add
  CRC           uint16
}

type GpsDataV3 struct {
  GpsTimestamp  uint64
  Lng           float64
  Lat           float64
  Alt           float32
  HardwareId    [16]uint8
  SprayFlag     uint8
  MotorStatus   uint8
  RadarHeight   uint16
  VelocityX     float32
  VelocityY     float32
  FarmDeltaY    float32
  FarmMode      uint8
  PilotUid      uint64
  SessionNum    [16]uint8
  FrameIndex    uint16
  FlightVersion [8]uint8
  Plant         uint8
  TeamId        uint32
  WorkArea      uint32
  FlowSpeed     uint16
  CRC           uint16
}

type OnceWorkInfo struct {
  WorkArea      float64
  SessionNum    [16]byte
  CRC           uint16
}

func (gpsData GpsDataV1) ConvertToDBInfo(flyer FlyerInfo) IuavFlightData {
  sn := string(gpsData.ProductId[0:])
  sn = strings.TrimRight(sn, string(0))

  b := gpsData.FlightVersion
  flightVersion := fmt.Sprintf("%d.%d.%d.%d",
    binary.LittleEndian.Uint16(b[0:2]),
    binary.LittleEndian.Uint16(b[2:4]),
    binary.LittleEndian.Uint16(b[4:6]),
    binary.LittleEndian.Uint16(b[6:]))

  return IuavFlightData{
    BossID:        flyer.BossID,
    UserID:        flyer.UserID,
    TeamID:        gpsData.TeamID,
    PVersion:      1,
    Timestamp:     gpsData.NTimeStamp,
    Longi:         gpsData.Longi,
    Lati:          gpsData.Lati,
    Alti:          gpsData.Alti,
    ProductId:     sn,
    SprayFlag:     gpsData.SprayFlag,
    MotorStatus:   gpsData.MotorStatus,
    RadarHeight:   int(gpsData.RadarHeight),
    VelocityX:     gpsData.VelocityX,
    VelocityY:     gpsData.VelocityY,
    FarmDeltaY:    gpsData.FarmDeltaY,
    FarmMode:      gpsData.FarmMode,
    Pilotnum:      gpsData.Pilotnum,
    Sessionnum:    strconv.FormatUint(gpsData.Sessionnum, 10),
    FrameIndex:    uint16(gpsData.FrameIndex & 0x7F),
    FrameFlag:     gpsData.FrameIndex >> 7,
    FlightVersion: flightVersion,
    WorkArea:      uint32(gpsData.WorkArea),
    Plant:         gpsData.Plant,
    CreateTime:    time.Now().Unix(),
  }
}

func (gpsData GpsDataV2) ConvertToDBInfo(flyer FlyerInfo) IuavFlightData {
  sn := string(gpsData.ProductId[0:])
  sn = strings.TrimRight(sn, string(0))

  b := gpsData.FlightVersion
  flightVersion := fmt.Sprintf("%d.%d.%d.%d",
    binary.LittleEndian.Uint16(b[0:2]),
    binary.LittleEndian.Uint16(b[2:4]),
    binary.LittleEndian.Uint16(b[4:6]),
    binary.LittleEndian.Uint16(b[6:]))

  return IuavFlightData{
    BossID:        flyer.BossID,
    UserID:        flyer.UserID,
    TeamID:        gpsData.TeamID,
    PVersion:      2,
    Timestamp:     gpsData.NTimeStamp,
    Longi:         gpsData.Longi,
    Lati:          gpsData.Lati,
    Alti:          gpsData.Alti,
    ProductId:     sn,
    SprayFlag:     gpsData.SprayFlag,
    MotorStatus:   gpsData.MotorStatus,
    RadarHeight:   int(gpsData.RadarHeight),
    VelocityX:     gpsData.VelocityX,
    VelocityY:     gpsData.VelocityY,
    FarmDeltaY:    gpsData.FarmDeltaY,
    FarmMode:      gpsData.FarmMode,
    Pilotnum:      gpsData.Pilotnum,
    Sessionnum:    strconv.FormatUint(gpsData.Sessionnum, 10),
    FrameIndex:    uint16(gpsData.FrameIndex & 0x7F),
    FrameFlag:     gpsData.FrameIndex >> 7,
    FlightVersion: flightVersion,
    WorkArea:      uint32(gpsData.WorkArea),
    Plant:         gpsData.Plant,
    FlowSpeed:     gpsData.FlowSpeed,
    CreateTime:    time.Now().Unix(),
  }
}

func (gpsData GpsDataV3) ConvertToDBInfo(flyer FlyerInfo) IuavFlightData {
  sn := string(gpsData.HardwareId[0:])
  sn = strings.TrimRight(sn, string(0))

  b := gpsData.FlightVersion
  flightVersion := fmt.Sprintf("%d.%d.%d.%d",
    binary.LittleEndian.Uint16(b[0:2]),
    binary.LittleEndian.Uint16(b[2:4]),
    binary.LittleEndian.Uint16(b[4:6]),
    binary.LittleEndian.Uint16(b[6:]))

  SessionId := uuid.FromBytesOrNil(gpsData.SessionNum[0:])

  return IuavFlightData{
    BossID:        flyer.BossID,
    UserID:        flyer.UserID,
    TeamID:        gpsData.TeamId,
    PVersion:      3,
    Timestamp:     gpsData.GpsTimestamp,
    Longi:         gpsData.Lng,
    Lati:          gpsData.Lat,
    Alti:          gpsData.Alt,
    ProductId:     sn,
    SprayFlag:     gpsData.SprayFlag,
    MotorStatus:   gpsData.MotorStatus,
    RadarHeight:   int(gpsData.RadarHeight),
    VelocityX:     gpsData.VelocityX,
    VelocityY:     gpsData.VelocityY,
    FarmDeltaY:    gpsData.FarmDeltaY,
    FarmMode:      gpsData.FarmMode,
    Pilotnum:      gpsData.PilotUid,
    Sessionnum:    SessionId.String(),
    FrameIndex:    gpsData.FrameIndex & 0x7FFF,
    FrameFlag:     uint8(gpsData.FrameIndex >> 15),
    FlightVersion: flightVersion,
    WorkArea:      gpsData.WorkArea,
    Plant:         gpsData.Plant,
    FlowSpeed:     gpsData.FlowSpeed,
    CreateTime:    time.Now().Unix(),
  }
}
