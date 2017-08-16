package main

const (
	//超级管理员账号，可监控所有的飞机
	_SUPER_UID_ = "36234651029942" //正式环境
	//_SUPER_UID_ = "801690643807174656" //测试环境

	PACKAGE_HEAD                    = uint8(0x55)
	APP_AUTH_CMD                    = uint8(0x01) //app鉴权数据包
	APP_AUTH_ACK_CMD                = uint8(0x02)
	APP_SEND_LOCK_CMD               = uint8(0x03) //给app发送锁定指令
	APP_RECEIVE_LOCK_CMD            = uint8(0x04)
  APP_UPLOAD_CMD                  = uint8(0x05) //一秒一次航点数据
	APP_SEND_MSG_CMD                = uint8(0x06) //给app发送消息
	APP_RECEIVE_MSG_CMD             = uint8(0x07)
	APP_SEND_LOCKTIME_CMD           = uint8(0x08) //给app发送时间段锁定消息
	APP_RECEIVE_LOCKTIME_CMD        = uint8(0x09)
  APP_SEND_ONCE_WORKINFO_CMD      = uint8(0x0B) //一次飞行的作业亩数

	CMD_DISPATCH_ID_AUTH_V1               = "1_1"
	CMD_DISPATCH_ID_RECV_LOCK_V1          = "4_1"
	CMD_DISPATCH_ID_RECV_LOCK_V2          = "4_2"
	CMD_DISPATCH_ID_UPLOAD_V1             = "5_1"
	CMD_DISPATCH_ID_UPLOAD_V2             = "5_2"
	CMD_DISPATCH_ID_UPLOAD_V3             = "5_3"
	CMD_DISPATCH_ID_RECV_MSG_V1           = "7_1"
	CMD_DISPATCH_ID_RECV_MSG_V2           = "7_2"
	CMD_DISPATCH_ID_RECV_TIMELOCK_V1      = "9_1"
	CMD_DISPATCH_ID_RECV_TIMELOCK_V2      = "9_2"
  CMD_DISPATCH_ID_ONCE_WORKINFO_V3      = "11_3"

	VERSION_FLAG = 0xfc00 //二进制: 1111110000000000
	DATALEN_FLAG = 0x03ff //二进制: 0000001111111111

	MAX_TOKEN_LEN            = 512
	APP_MEMBER_CENTER        = "member_center"
	FLIGHT_QUEUE_NAME        = "iuav_flight_queue"
	FLIGHT_FAILED_QUEUE_NAME = "iuav_flight_failed_queue"
	IUAV_FLIGHT_TABLE_NAME   = "iuav_flight_data"
	IUAV_AGRO_TABLE_NAME     = "agro_active_info"
	IUAV_AGRO_FLYER          = "agro_flyer"
	IUAV_AGRO_TEAM           = "agro_team"
	IUAV_AGRO_MANAGER_DRONE  = "manager_drone"
	IUAV_AGRO_FLIGHT         = "agro_flight"

	//发给前端网页消息
	ONLINE    = "online"
	OFFLINE   = "offline"
	INIT      = "init"
	RESPONCE  = "responce"
	//接收到的消息类型
	WS_MSG_UID     = "uid"
	WS_MSG_LOCK    = "lock"
	WS_MSG_UNLOCK  = "unlock"
	WS_MSG_SENDMSG = "sendmsg"
	//发给app消息
	LOCKED   = uint8(0x01)
	UNLOCKED = uint8(0x00)

	//实时监控中允许的延迟时间
	DelayTime = 60000

	STATUS_OK                   = 0
	ERR_NOT_AUTH                = 1001
	ERR_DB_FAILED               = 1002
	ERR_PARAMS_ERR              = 1003
	ERR_NOT_ONLINE              = 1004
	ERR_REQ_REALTIME_SVR_FAILED = 1005
	ERR_SEND_APP_FAILED         = 1006
	ERR_APP_RESPONE_TIMEOUT     = 1007
	ERR_SAVE_TODB_FAILED        = 1008
	ERR_MSG_USE_UP              = 1009

	IUAV_ONLINE_INFO_TABLE_NAME = "iuav_online_info"
)

type AccountInfo struct {
	Item AccountItem `json:"account_info"`
}

type AccountItem struct {
	Email  string `json:"email"`
	UserID uint64 `json:"account_id"`
}

type UserInfoResult struct {
	Status  int
	Message string
	Items   []AccountInfo `json:"items"`
}

type FlyerInfo struct {
	UserName       string
	UserID         uint64
	Job_level      uint8
	Token          string
	Email          string
	BossID         string
	BossName       string
	HardwareSN     string
	HardwareName   string
	HardwareType   string
	HardwareLocked uint8
	TeamID         uint32
	TeamName       string
	Captain        string
	Managers       []string
	TodayWork      uint32
}
