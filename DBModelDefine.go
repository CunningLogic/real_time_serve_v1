package main

import "database/sql"

type ActiveDBInfo struct {
  Id            uint64         `db:"id"`
  Orderid       sql.NullString `db:"order_id"`
  Polno         sql.NullString `db:"pol_no"`
  Exptm         sql.NullString `db:"exp_tm"`
  Queryflag     sql.NullBool   `db:"query_flag"`
  Activetm      sql.NullString `db:"active_tm"`
  Account       sql.NullString `db:"account"`
  Uid           sql.NullString `db:"uid"`
  Teamid        sql.NullInt64  `db:"team_id"`
  Idcard        string         `db:"idcard"`
  Phone         string         `db:"phone"`
  Bodycode      string         `db:"body_code"`
  Hardwareid    sql.NullString `db:"hardware_id"`
  Type          sql.NullString `db:"type"`
  Isactive      sql.NullBool   `db:"is_active"`
  Deleted       uint8          `db:"deleted"`
  Nickname      sql.NullString `db:"nickname"`
  Locked        uint8          `db:"locked"`
  Locked_notice uint8          `db:"locked_notice"`

  //timelock
  TimeLocked        uint8          `db:"timelocked"`
  TimeLocked_notice uint8          `db:"timelocked_notice"`
  Lock_begin        sql.NullString `db:"lock_begin"`
  Lock_end          sql.NullString `db:"lock_end"`

  Is_online       uint8          `db:"is_online"`
  Msg_sum         uint8          `db:"msg_sum"`
  Active_location sql.NullString `db:"active_location"`
  IP              sql.NullString `db:"ip"`
  Ext1            sql.NullString `db:"ext1"`
  Ext2            sql.NullString `db:"ext2"`
  Updatedat       sql.NullString `db:"updated_at"`
  Createdat       sql.NullString `db:"created_at"`
}

type HardwareInfo struct {
  BossID         string `db:"uid"`
  BossName       string
  HardwareName   string `db:"nickname"`
  HardwareType   string `db:"type"`
  TeamID         uint32 `db:"team_id"`
  HardwareLocked uint8  `db:"locked"`
}

//飞机连接信息
type OnlineInfo struct {
  Id         uint64 `db:"id"`
  SN         string `db:"sn"`
  UserID     uint64 `db:"user_id"`
  BossID     string `db:"boss_id"`
  Captain    string `db:"captain"`
  ServerInfo string `db:"server_info"`
  TimeStamp  int64  `db:"timestamp"`
  Ext1       string `db:"ext1"`
  Ext2       int64  `db:"ext2"`
}

type IuavFlightData struct {
  Id            uint64  `db:"id"`
  BossID        string  `db:"boss_id"`
  UserID        uint64  `db:"user_id"`
  TeamID        uint32  `db:"team_id"`
  PVersion      uint16  `db:"version"`
  Timestamp     uint64  `db:"timestamp"`
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
  Pilotnum      uint64  `db:"pilot_num"`
  Sessionnum    string  `db:"session_num"`
  FrameIndex    uint16  `db:"frame_index"`
  FrameFlag     uint8   `db:"frame_flag"`
  FlightVersion string  `db:"flight_version"`
  Plant         uint8   `db:"plant"`
  CreateTime    int64   `db:"create_time"`
  WorkArea      uint32  `db:"work_area"`
  FlowSpeed     uint16  `db:"flow_speed"`
  Ext1          string  `db:"ext1"`
  Ext2          int64   `db:"ext2"`
}
