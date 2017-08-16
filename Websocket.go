package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"strings"
	"time"
  "strconv"
  "errors"
	"sync"
)

type Wsocket struct {
	stop           chan bool
	wg             sync.WaitGroup
	connectionsMap map[string][]*websocket.Conn
	mutex          sync.RWMutex
	mutexWriteJSON sync.Mutex
	sliceMutex     sync.RWMutex
}

type BasePacket struct {
  Type    string           `json:"type"`
  Data    json.RawMessage  `json:"data"`
  Cookie  *string          `json:"cookie"`
}

type ws_protocol_handler func (*string, BasePacket, *websocket.Conn) error

var ws_protocol_handler_dispatch_table = map[string]ws_protocol_handler {
  WS_MSG_UID:      ws_msg_uid_handler,
  WS_MSG_LOCK:     ws_msg_lock_handler,
  WS_MSG_UNLOCK:   ws_msg_lock_handler,
  WS_MSG_SENDMSG:  ws_msg_sendmsg_handler,
}

var upgrader = websocket.Upgrader{
  ReadBufferSize:    1024,
  WriteBufferSize:   1024,
  EnableCompression: true,
  CheckOrigin:       func(r *http.Request) bool { return true },
}

func NewWsocket() *Wsocket {
	return &Wsocket{
		stop:           make(chan bool),
		wg:             sync.WaitGroup{},
		connectionsMap: make(map[string][]*websocket.Conn, 1000), //同时支持1000个boss在线监控
	}
}

func (ws *Wsocket) Start() {
	fmt.Println("websocket listening in port", GetConfigInstance().WebsocketPort)
	go ws.webSocket()
}

func (ws *Wsocket) webSocket() {
	http.HandleFunc("/echo", ws.msgLoop)
	http.HandleFunc("/", ws.home)
	http.HandleFunc("/locktime", ws.wsLockTimePeriod)
	http.HandleFunc("/lock", ws.wslockDrone)
	service := fmt.Sprintf("0.0.0.0:%d", GetConfigInstance().WebsocketPort)
	http.ListenAndServe(service, nil)
}

func (ws *Wsocket) InitDroneInfo(c *websocket.Conn, uid string) error {
  var lockedInfo LockedInfo
  var err error

  if uid == _SUPER_UID_ {
    //如果登录者是超级用户
    lockedInfo, err = service.GetSuperDroneLockedInfo(uid)
  } else {
    //构建websocket初始化信息：locked数量，drone总的数量
    lockedInfo, err = service.GetDroneLockedInfo(uid)
  }

  if err != nil {
    fmt.Println("init information error", err)
    return err
  }

  var init_info struct {
    Type string     `json:"type"`
    Data LockedInfo `json:"data"`
  }
  init_info.Data.Drone_sum = lockedInfo.Drone_sum
  init_info.Data.Locked_drones = lockedInfo.Locked_drones
  init_info.Type = INIT
  err = ws.SafeWriteJSON(c, init_info)

  if err != nil {
    fmt.Println("write json init info failed", uid, err)
    return err
  }

  return nil
}

func check_login(uid string, cookie string, addr string) (bool, error) {
  if uid == "" || cookie == "" {
    return false, errors.New("null parameters")
  }

  token, err := service.djiGateWay.GetTokenByCookie(cookie, addr)
  if err != nil || len(token) == 0 {
    return false, err
  }

  userInfo, err := service.djiGateWay.CheckToken(string(token), addr)
  if err != nil || userInfo.Status != 0 || len(userInfo.Items) == 0 || userInfo.Items[0].Item.UserID == 0 {
    fmt.Println("app auth failed", cookie, err, userInfo)
    return false, err
  }

  if uid == strconv.FormatUint(userInfo.Items[0].Item.UserID, 10) {
    return true, nil
  }

  return false, errors.New("invalid uid")
}

func ws_msg_uid_handler(binding_uid *string, msg BasePacket, conn *websocket.Conn) error {
  uid := string(msg.Data)

  if result, err := check_login(uid, *msg.Cookie, conn.RemoteAddr().String()); !result {
    *binding_uid = ""
    fmt.Println("websocket check login failed", err)
    return err
  }

  err := ws.InitDroneInfo(conn, uid)
  if err != nil {
    return err
  }
  ws.SafeMapSet(ws.connectionsMap, uid, conn)

  *binding_uid = uid

  return nil
}

func ws_msg_lock_handler(binding_uid *string, msg BasePacket, conn *websocket.Conn) error {
  if *binding_uid == "" {
    return errors.New("websocket lock msg handler no binding uid")
  }

  var lock_msg struct {
    HardwareID string  `json:"hardware_id"`
  }

  err := json.Unmarshal(msg.Data, lock_msg)
  if err != nil {
    return err
  }

  if len(lock_msg.HardwareID) <= 0 {
    return errors.New("websocket lock msg handler empty hardware id")
  }

  if ok := ws.wsCheckIsBoss(lock_msg.HardwareID, *binding_uid); !ok {
    return errors.New("websocket lock msg handler no privilege")
  }

  ws.wsLockHardware(conn, lock_msg.HardwareID, LOCKED)

  return nil
}

func ws_msg_sendmsg_handler(binding_uid *string, msg BasePacket, conn *websocket.Conn) error {
  if *binding_uid == "" {
    return errors.New("websocket lock msg handler no binding uid")
  }

  var sendmsg_msg struct {
    HardwareID   string `json:"hardware_id"`
    Content      string `json:"content"`
    SendToAll    bool   `json:"send_to_all"`
  }

  err := json.Unmarshal(msg.Data, sendmsg_msg)
  if err != nil {
    return err
  }

  if ok := ws.wsCheckIsBoss(sendmsg_msg.HardwareID, *binding_uid); !ok {
    return errors.New("websocket sendmsg msg handler no privilege")
  }

  if sendmsg_msg.SendToAll {
    ws.wsSendMsgToAll(conn, sendmsg_msg.Content, *binding_uid)
  } else {
    ws.wsSendMsg(conn, sendmsg_msg.Content, *binding_uid)
  }

  return nil
}

func (ws *Wsocket) msgLoop(w http.ResponseWriter, r *http.Request) {
  c, err := upgrader.Upgrade(w, r, nil)
  if err != nil {
    fmt.Println("websocket echo error", err)
    return
  }

  var conn_binding_uid string

  defer func() {
    if c != nil {
      fmt.Println("websocket close connection")
      c.WriteMessage(websocket.CloseMessage, []byte{})
      c.Close()
    }

    if conn_binding_uid != "" {
      fmt.Println("websocket del connection info")
      ws.SafeMapDel(ws.connectionsMap, conn_binding_uid, c)
    }
  }()

  var msg BasePacket
	for {
		if err = c.ReadJSON(&msg); err != nil {
      fmt.Println("websocket read json error", err)
			return
		}

    handler, ok := ws_protocol_handler_dispatch_table[msg.Type]
    if ok {
      err = handler(&conn_binding_uid, msg, c)
      if err != nil {
        fmt.Println("process websocket msg '", msg.Type, "'error: ", err)
      }
    } else {
      fmt.Println("websocket unknown msg type", msg.Type)
    }
	}
}

func (ws *Wsocket) SafeMapSet(m map[string][]*websocket.Conn, k string, v *websocket.Conn) {
	ws.mutex.Lock()
	ws.sliceMutex.Lock()
	if _, ok := m[k]; !ok {
		m[k] = make([]*websocket.Conn, 0, 10)
	}
	m[k] = append(m[k], v)
	ws.sliceMutex.Unlock()
	ws.mutex.Unlock()
}

func (ws *Wsocket) SafeMapGet(m map[string][]*websocket.Conn, k string) ([]*websocket.Conn, bool) {
	ws.mutex.RLock()
	ws.sliceMutex.RLock()
	v, ok := m[k]
	if !ok {
		ws.sliceMutex.RUnlock()
		ws.mutex.RUnlock()
		return v, ok
	}
	tmp := make([]*websocket.Conn, len(v))
	copy(tmp, v)
	ws.sliceMutex.RUnlock()
	ws.mutex.RUnlock()
	return tmp, ok
}

func (ws *Wsocket) SafeMapDel(m map[string][]*websocket.Conn, k string, c *websocket.Conn) {
	ws.mutex.Lock()
	ws.sliceMutex.Lock()
	for n, v := range m[k] {
		if v == c {
			//https://github.com/golang/go/wiki/SliceTricks
			copy(m[k][n:], m[k][n+1:])
			m[k][len(m[k])-1] = nil // or the zero value of T
			m[k] = m[k][:len(m[k])-1]
			break
		}
	}
	ws.sliceMutex.Unlock()
	if len(m[k]) == 0 {
		delete(m, k)
	}
	ws.mutex.Unlock()
}

func (ws *Wsocket) SafeWriteJSON(c *websocket.Conn, json interface{}) error {
	ws.mutexWriteJSON.Lock()
	err := c.WriteJSON(json)
	ws.mutexWriteJSON.Unlock()
	return err
}

func (ws *Wsocket) wsCheckIsBoss(sn string, uid string) bool {
	tmpid, _ := service.GetBossIDBySn(sn)
	if uid == tmpid {
		return true
	}
	return false
}

func (ws *Wsocket) wsLockHardware(c *websocket.Conn, sn string, cmd uint8) {
	var responce RsponceResult
	responce.Type = RESPONCE

	if cmd == LOCKED {
		responce.Data.Type = WS_MSG_LOCK
	} else if cmd == UNLOCKED {
		responce.Data.Type = WS_MSG_UNLOCK
	}
	responce.Data.HardwareId = sn

	info, ok := service.connMgr.GetConnInfoByHardwareID(sn)

	if !ok {
		responce.Data.Status = ERR_NOT_ONLINE
	} else {
		p := packLockPackage(cmd, sn)

		_, err := info.conn.Write(p)
		if err != nil {
			fmt.Println("SendManagerCmd failed", cmd, sn, err)
			responce.Data.Status = ERR_SEND_APP_FAILED
		}

		select {
		case <-time.After(time.Duration(3) * time.Second):
			responce.Data.Status = ERR_APP_RESPONE_TIMEOUT
		case status := <-info.lockStatus:
			if status == 1 { //锁定成功
				responce.Data.Status = uint(STATUS_OK) //给网页发送的状态码统一使用0表示成功

        service.dbMgr.dbmap.Exec("update agro_active_info set locked=1,locked_notice=1 where hardware_id=?", sn)
			} else {
				responce.Data.Status = ERR_SEND_APP_FAILED
			}
		}
	}

	ws.SafeWriteJSON(c, responce)
}

func (ws *Wsocket) wsSendMsg(c *websocket.Conn, sn string, msg string) {
	var responce RsponceMsg
	responce.Type = RESPONCE
	responce.Data.Type = WS_MSG_SENDMSG
	responce.Data.HardwareId = sn

  info, ok := service.connMgr.GetConnInfoByHardwareID(sn)
  if !ok {
    responce.Data.Status = ERR_NOT_ONLINE
  } else {
    p := packMsgPackage(msg, sn)
    _, err := info.conn.Write(p)
    if err != nil {
      fmt.Println("SendManagerMsg failed", msg, sn, err)
      responce.Data.Status = ERR_SEND_APP_FAILED
    }
    select {
    case <-time.After(time.Duration(3) * time.Second):
      responce.Data.Status = ERR_APP_RESPONE_TIMEOUT
    case status := <-info.msgStatus:
      if status == 0 {
        responce.Data.Status = uint(STATUS_OK)
        query := fmt.Sprintf("update %s set msg_sum = %d where hardware_id = '%s' and deleted = 0", IUAV_AGRO_TABLE_NAME, 100, sn)
        _, err = service.dbMgr.dbmap.Exec(query)
        if err == nil {
          responce.Data.Msg_sum = 100
        } else {
          responce.Data.Status = ERR_SAVE_TODB_FAILED
        }
      } else {
        responce.Data.Status = ERR_SEND_APP_FAILED
      }
    }
  }

	ws.SafeWriteJSON(c, responce)
}

func (ws *Wsocket) wsSendMsgToAll(c *websocket.Conn, msg string, bossID string) {
	drones, _ := service.QueryOnlineDroneInfo(bossID) //查找出所有在线飞行器
	for _, v := range drones {
		go ws.wsSendMsg(c, v, msg)
	}
}

func (ws *Wsocket) wsLockTimePeriod(w http.ResponseWriter, r *http.Request) {
	var result LockResult

	auth := r.Header.Get("AuthToken")
	result.SN = SafeParams(r, "sn")
	result.Cmd = SafeParams(r, "cmd")
	result.Lock_begin = SafeParams(r, "lock_begin")
	result.Lock_end = SafeParams(r, "lock_end")
	result.BossName = SafeParams(r, "bossname")
	result.BossID = SafeParams(r, "bossid")
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	forwarded := r.Header.Get("X-FORWARDED-FOR")

	clientInfo := fmt.Sprintf("[remote:%s][forwarded:%s][auth:%s]", ip, forwarded, auth)
	fmt.Println("HandleLockTimePeriod start", result, clientInfo)

	//send to app
	info, ok := service.connMgr.GetConnInfoByHardwareID(result.SN)
	if !ok {
		result.Status = ERR_NOT_ONLINE
	} else if strings.Compare(auth, GetConfigInstance().WebsocketAuthToken) == 0 {
		if result.Cmd == WS_MSG_LOCK || result.Cmd == WS_MSG_UNLOCK { //锁定
			if err := service.SendLockTimePeriod(info.conn, result); err != nil { //发给app
				result.Status = ERR_SEND_APP_FAILED
			} else {
				//wait response
				select {
				case <-time.After(time.Duration(3) * time.Second):
					result.Status = ERR_APP_RESPONE_TIMEOUT
				case status := <-info.timelockStatus:
					result.LockStatus = status
					result.Status = int(STATUS_OK) //回复时间段锁定已经成功
				}
			}
		} else {
			result.Status = ERR_PARAMS_ERR
		}
	} else {
		result.Status = ERR_NOT_AUTH
	}
	fmt.Println("HandleLockTimePeriod end", result, clientInfo)

	b, _ := json.Marshal(&result)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b) //回复http请求
}

func (ws *Wsocket) wsCmdHardware(c *websocket.Conn, sn string, cmd string) {
	var responce RsponceResult
	responce.Type = RESPONCE

	type responceData struct {
		Hardware_id string `json:"hardware_id"`
	}
	type responceUnlock struct {
		Type string       `json:"type"`
		Data responceData `json:"data"`
	}

	var r responceUnlock

	r.Type = cmd

	r.Data.Hardware_id = sn

	ws.SafeWriteJSON(c, r) //发送给web
}

func (ws *Wsocket) wslockDrone(w http.ResponseWriter, r *http.Request) {
	auth       := r.Header.Get("AuthToken")
	sn         := SafeParams(r, "sn")
	cmd        := SafeParams(r, "cmd")
	bossid     := SafeParams(r, "bossid")

	//auth
	if strings.Compare(auth, GetConfigInstance().WebsocketAuthToken) != 0 {
		return
	}

  var resp LockResult
  resp.Cmd = cmd
  if cmd == WS_MSG_LOCK {
    resp.LockStatus = 1
  } else {
    resp.LockStatus = 0
  }

	info, ok := service.connMgr.GetConnInfoByHardwareID(sn)
	if !ok {
		resp.Status = ERR_NOT_ONLINE
	} else {
		p := packLockPackage(resp.LockStatus, sn)
		_, err := info.conn.Write(p) //发送给app
		if err != nil {
			fmt.Println("SendManagerCmd failed", cmd, sn, err)
			resp.Status = ERR_SEND_APP_FAILED
		}
		select {
		case <-time.After(time.Duration(3) * time.Second):
			resp.Status = ERR_APP_RESPONE_TIMEOUT
		case rec := <-info.lockStatus:
			rec = rec
			resp.Status = int(STATUS_OK)
		}
	}

	//2. send to web
	value, ok := ws.SafeMapGet(ws.connectionsMap, bossid)
	if ok {
		for _, v := range value {
			ws.wsCmdHardware(v, sn, cmd)
		}
	}

	//3. responce to the http
	b, _ := json.Marshal(&resp)
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)

	fmt.Println("HandlelockDrone end", b)
}

func (ws *Wsocket) home(w http.ResponseWriter, r *http.Request) {
  w.Write([]byte("working."))
}

func (ws *Wsocket) WsocketSendIuavFlightData(v []*websocket.Conn, flyer FlyerInfo, info IuavFlightData) {
  var gps_info GpsInfo
  var point Point

  if info.FrameFlag == 0 {
    gps_info.Type = ONLINE
  } else {
    gps_info.Type = OFFLINE
  }

  if now := time.Now().Unix() * 1000; info.Timestamp+DelayTime < uint64(now) {
    //处理缓存数据是否需要实时显示问题
    return
  }

  gps_info.Point_data.Machine_info.Hardware_id = flyer.HardwareSN
  gps_info.Point_data.Machine_info.Nickname = flyer.HardwareName
  gps_info.Point_data.Machine_info.Type = flyer.HardwareType
  gps_info.Point_data.Machine_info.Locked = flyer.HardwareLocked

  gps_info.Point_data.Flyer_info.Uid = flyer.UserID
  gps_info.Point_data.Flyer_info.Realname = flyer.UserName
  gps_info.Point_data.Flyer_info.Avatar = ""
  gps_info.Point_data.Flyer_info.Job_level = flyer.Job_level

  gps_info.Point_data.Flyer_info.Team_info.Id = flyer.TeamID
  gps_info.Point_data.Flyer_info.Team_info.Name = flyer.TeamName

	point.Lati = info.Lati
	point.Longi = info.Longi
	point.Plant = info.Plant
	point.Velocity_x = info.VelocityX
	point.Velocity_y = info.VelocityY
	point.Radar_height = info.RadarHeight
	point.Farm_delta_y = info.FarmDeltaY
	point.WorkArea = uint32(info.WorkArea) //增加已喷面积 2017.01.01 14:18
  flyer.TodayWork += uint32(info.WorkArea)
	point.TodayWork = flyer.TodayWork //增加当天作业总亩数
	point.FlowSpeed = info.FlowSpeed //协议1.1新增字段
  gps_info.Point_data.Point_info = point

  for _, c := range v {
    err := ws.SafeWriteJSON(c, gps_info)
    if err != nil {
      ws.SafeMapDel(ws.connectionsMap, flyer.BossID, c) //第一时间把map删除
    }
  }
  v = nil
}
