package main

import (
  "net"
  "fmt"
  "encoding/binary"
  "crypto/tls"
  "sync"
  "errors"
  "time"
  "strconv"
)

type protocol_handler func (*bool, *FlyerInfo, *PackageHeadInfo, []byte, net.Conn) error

var protocol_handler_dispatch_table = map[string]protocol_handler {
  CMD_DISPATCH_ID_AUTH_V1:           cmd_login_handler,
  CMD_DISPATCH_ID_RECV_LOCK_V1:      cmd_lockrsp_handler,
  CMD_DISPATCH_ID_RECV_LOCK_V2:      cmd_lockrsp_handler,
  CMD_DISPATCH_ID_UPLOAD_V1:         cmd_upload_handler,
  CMD_DISPATCH_ID_UPLOAD_V2:         cmd_upload_handler,
  CMD_DISPATCH_ID_UPLOAD_V3:         cmd_upload_handler,
  CMD_DISPATCH_ID_RECV_MSG_V1:       cmd_msgrsp_handler,
  CMD_DISPATCH_ID_RECV_MSG_V2:       cmd_msgrsp_handler,
  CMD_DISPATCH_ID_RECV_TIMELOCK_V1:  cmd_timelockrsp_handler,
  CMD_DISPATCH_ID_RECV_TIMELOCK_V2:  cmd_timelockrsp_handler,
  CMD_DISPATCH_ID_ONCE_WORKINFO_V3:  cmd_once_workinfo_handler,
}

type Service struct {
  stop                  chan bool
  wg                    sync.WaitGroup
  tokenMap              map[string]interface{}

  connMgr               *ConnMgr
  dbMgr                 *DBManager

  mutex                 sync.Mutex
  djiGateWay            *DJIGateWay
}

func NewService() *Service {
  dbMgr := NewDBManager()

  gateway, err := NewDJIGateWay(GetConfigInstance().Gateway, GetConfigInstance().GatewayAppID, GetConfigInstance().GatewayAppKey)
  checkError(err)

  return &Service{
    stop:           make(chan bool),
    wg:             sync.WaitGroup{},
    connMgr:        NewConnMgr(),
    tokenMap:       make(map[string]interface{}, 1000),
    djiGateWay:     gateway,
    dbMgr:          dbMgr,
  }
}

func (s *Service) Start() {
  s.dbMgr.StartDBSaver()

  cert, err := tls.LoadX509KeyPair(GetConfigInstance().PublicPemPath, GetConfigInstance().PrivatePemPath)
  checkError(err)
  config := tls.Config{Certificates: []tls.Certificate{cert}}
  service := fmt.Sprintf("0.0.0.0:%d", GetConfigInstance().ServerPort)
  listener, err := tls.Listen("tcp", service, &config)
  checkError(err)

  fmt.Println("tls start listening in prot ", GetConfigInstance().ServerPort)

  go s.Serve(listener)
}

func (s *Service) Stop() {
  close(s.stop)
  s.wg.Wait()
  fmt.Println("stop DJIGateWay...")
  s.djiGateWay.Close()
  fmt.Println("Service stop, goodbye...")
}

func (s *Service) Serve(l net.Listener) {
  defer l.Close()

  for {
    conn, err := l.Accept()

    if err != nil {
      fmt.Println("Accept a client failed", conn.RemoteAddr(), err)
      continue
    }

    fmt.Println("Accept a client success", conn.RemoteAddr())

    go s.HandleClient(conn)
  }
}

func (s *Service) SafeMapSet(m map[string]interface{}, k string, v interface{}) {
  s.mutex.Lock()
  m[k] = v
  s.mutex.Unlock()
}

func (s *Service) SafeMapGet(m map[string]interface{}, k string) (interface{}, bool) {
  s.mutex.Lock()
  v, ok := m[k]
  s.mutex.Unlock()
  return v, ok
}

func (s *Service) SafeMapDel(m map[string]interface{}, k string) {
  s.mutex.Lock()
  delete(m, k)
  s.mutex.Unlock()
}

func (s *Service) InitConnectionInfo(conn net.Conn, flyer FlyerInfo) *ConnectionInfo {
  info := new(ConnectionInfo)
  info.conn = conn
  info.userID = flyer.UserID
  info.captain = flyer.Captain
  info.bossID = flyer.BossID
  info.lockStatus = make(chan uint8, 1)
  info.msgStatus = make(chan uint8, 1)
  info.timelockStatus = make(chan uint8, 1)

  s.connMgr.SetConnInfo(flyer.HardwareSN, conn, info)

  s.SafeMapSet(s.tokenMap, flyer.Token, flyer)

  time.AfterFunc(time.Minute*1, func() { //token cache 30 minutes
    s.SafeMapDel(s.tokenMap, flyer.Token)
  })

  return info
}

func (s *Service) RecvProtocolHead(conn net.Conn) (*PackageHeadInfo, error) {
  buf := make([]byte, 4)
  read_count, err := conn.Read(buf)

  if err != nil {
    return nil, err
  }

  if read_count != 4 {
    return nil, errors.New("read protocol head != header size")
  }

  var head_info PackageHeadInfo

  head_info.SOF = uint8(buf[0])
  vl := binary.LittleEndian.Uint16(buf[1:3])
  head_info.Ver = (vl & VERSION_FLAG) >> 10
  head_info.DataLen = vl & DATALEN_FLAG
  head_info.CmdID = uint8(buf[3])

  // TODO: common check

  return &head_info, nil
}

func (s *Service) RecvAndDispatchProtocol(is_login *bool, flyer_info *FlyerInfo, head *PackageHeadInfo, conn net.Conn) error {
  buf := make([]byte, head.DataLen - 4)
  read_count, err := conn.Read(buf)

  if err != nil {
    return err
  }

  if uint16(read_count) != (head.DataLen - 4) {
    return errors.New("read protocol body != body size")
  }

  handler_id := fmt.Sprintf("%v_%v", head.CmdID, head.Ver)
  handler, ok := protocol_handler_dispatch_table[handler_id]
  if ok {
    err = handler(is_login, flyer_info, head, buf, conn)
  } else {
    fmt.Println("no protocol handler")
    //err = errors.New("no protocol handler")
  }

  return err
}

func (s *Service) HandleClient(conn net.Conn) {
  s.wg.Add(1)

  key := fmt.Sprintf("%#p", conn)
  connInfo := fmt.Sprintf("[key:%s] [client:%s]", key, conn.RemoteAddr())

  var conn_binding_flyer_info FlyerInfo

  defer func() {
    s.connMgr.DelConnInfoByConn(conn)
    s.dbMgr.UpdateDroneStatus(conn_binding_flyer_info.HardwareSN, 0)
    conn.Close()
    s.wg.Done()
    fmt.Println("disconnect ", connInfo)
  }()

  ST_HEAD := 0
  ST_BODY := 1
  PROTOCOL_STATES := [2]int{ST_HEAD, ST_BODY}

  state_index := 0
  current_state := PROTOCOL_STATES[state_index]

  is_login := false

  var head_info *PackageHeadInfo

  for {
    select {
    case <-s.stop:
      fmt.Println("server stop to service ", connInfo)
      return
    default:
    }

    var err error

    if current_state == ST_HEAD {
      head_info, err = s.RecvProtocolHead(conn)
    } else if current_state == ST_BODY {
      err = s.RecvAndDispatchProtocol(&is_login, &conn_binding_flyer_info, head_info, conn)
    }

    if err != nil {
      fmt.Println("conn read error", err)
      return
    }

    state_index++
    state_index = state_index % len(PROTOCOL_STATES)
    current_state = PROTOCOL_STATES[state_index]
  }
}

func (s *Service) CheckAndSendLockMsg(conn net.Conn, flyer FlyerInfo, info *ConnectionInfo) {
  //app一旦登录则检查是否有实时段锁定信息需要发送

  var active ActiveDBInfo
  var result LockResult
  err := s.dbMgr.dbmap.SelectOne(&active, "select * from agro_active_info where hardware_id=?", flyer.HardwareSN)
  if err != nil {
    return
  }

  if active.Locked == 0 {
    result.Cmd = WS_MSG_UNLOCK
  } else {
    result.Cmd = WS_MSG_LOCK
  }

  if active.Locked_notice != 0 {
    return
  }

  result.SN = flyer.HardwareSN
  time.Sleep(time.Second * 1)
  if err := s.SendLockMsg(conn, result); err != nil {
    result.Status = ERR_SEND_APP_FAILED
  } else {
    //wait response
    select {
    case <-time.After(time.Duration(3) * time.Second):
      result.Status = ERR_APP_RESPONE_TIMEOUT
    case status := <-info.lockStatus:
      result.LockStatus = status
      active.Locked_notice = 1 //修改数据库标志
      s.dbMgr.dbmap.Update(&active)
    }
  }
}

func (s *Service) CheckAndSendLockTimeMsg(conn net.Conn, flyer FlyerInfo, info *ConnectionInfo) {
  //app一旦登录则检查是否有时间段锁定信息需要发送
  var active ActiveDBInfo
  var result LockResult

  err := s.dbMgr.dbmap.SelectOne(&active, "select * from agro_active_info where hardware_id=?", flyer.HardwareSN)
  if err != nil {
    return
  }

  if active.TimeLocked == 0 {
    result.Cmd = WS_MSG_UNLOCK
  } else {
    result.Cmd = WS_MSG_LOCK
  }

  if active.TimeLocked_notice != 0 {
    return
  }

  result.SN = flyer.HardwareSN
  if active.Lock_begin.Valid && len(active.Lock_end.String) == 10 {
    t1, _ := time.Parse("2006-01-02", active.Lock_begin.String)
    result.Lock_begin = strconv.FormatInt(t1.Unix()*1000, 10)
    t2, _ := time.Parse("2006-01-02", active.Lock_end.String)
    result.Lock_end = strconv.FormatInt(t2.Unix()*1000, 10)
  } else {
    result.Lock_begin = "0"
    result.Lock_end = "0"
  }

  result.BossName = flyer.BossName
  result.BossID = info.bossID
  time.Sleep(time.Second) //睡眠1s

  if err := s.SendLockTimePeriod(conn, result); err != nil {
    result.Status = ERR_SEND_APP_FAILED
  } else {
    //wait response
    select {
    case <-time.After(time.Duration(3) * time.Second):
      result.Status = ERR_APP_RESPONE_TIMEOUT
    case status := <-info.timelockStatus:
      result.LockStatus = status
      active.TimeLocked_notice = 1 //修改数据库标志
      s.dbMgr.dbmap.Update(&active)
    }
  }
}
