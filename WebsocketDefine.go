package main

type LockedInfo struct {
  Locked_drones []string `json:"lock"`
  Drone_sum     int64    `json:"count"`
}

type LockResult struct {
  Status     int    `json:"status"`
  BossName   string `json:"bossName"`
  BossID     string `json:"bossID"`
  SN         string `json:"sn"`
  Cmd        string `json:"cmd"`
  Lock_begin string `json:"lock_begin"`
  Lock_end   string `json:"lock_end"`
  ErrMsg     string `json:"error"`
  LockStatus uint8  `json:"lock_status"`
}

type Sendmsg struct {
  HardwareId string `json:"hardware_id"`
  Content    string `json:"content"`
  SendToAll  bool   `json:"send_to_all"`
}

//在线飞行器发送的消息
type GpsInfo struct {
  Type       string `json:"type"`
  Point_data Data   `json:"data"`
}

type Data struct {
  Machine_info Machine     `json:"machine_info"`
  Flyer_info   Flyer       `json:"flyer_info"`
  Point_info   interface{} `json:"point_info"`
}

type Machine struct {
  Hardware_id string `json:"hardware_id"`
  Nickname    string `json:"nickname"`
  Type        string `json:"type"`
  Locked      uint8  `json:"locked"`
}

type Flyer struct {
  Uid       uint64 `json:"id"`
  Realname  string `json:"realname"`
  Avatar    string `json:"avatar"`
  Job_level uint8  `json:"job_level"`
  Team_info Team   `json:"team_info"`
}

type Team struct {
  Id   uint32 `json:"id"`
  Name string `json:"name"`
}

type Point struct {
  Lati         float64 `json:"lati"`
  Longi        float64 `json:"longi"`
  Plant        uint8   `json:"plant"`
  Velocity_x   float32 `json:"velocity_x"`
  Velocity_y   float32 `json:"velocity_y"`
  Radar_height int     `json:"radar_height"`
  Farm_delta_y float32 `json:"interval"`
  WorkArea     uint32  `json:"work_area"`
  TodayWork    uint32  `json:"today_work"`
  FlowSpeed    uint16  `json:"flow_speed"`
}

//resonce struct
type RsponceResult struct {
  Type string      `json:"type"`
  Data RsponceData `json:"data"`
}

type RsponceData struct {
  Type       string `json:"type"`
  HardwareId string `json:"hardware_id"`
  Status     uint   `json:"status"`
}

//发送信息的回复
type RsponceMsg struct {
  Type string         `json:"type"`
  Data RsponceMsgData `json:"data"`
}

type RsponceMsgData struct {
  Type       string `json:"type"`
  HardwareId string `json:"hardware_id"`
  Status     uint   `json:"status"`
  Msg_sum    uint8  `json:"available_msg_count"`
}
