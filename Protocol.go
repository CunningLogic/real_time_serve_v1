package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
  "errors"
  "strings"
)

func packAckPackage(info PackageHeadInfo, cmd uint8, ack uint8) []byte {
  buf := new(bytes.Buffer)

  binary.Write(buf, binary.LittleEndian, info.SOF)
  vl := uint16(info.Ver << 10) | uint16(1 + 2 + 1 + 1 + 2)
  binary.Write(buf, binary.LittleEndian, vl)
  binary.Write(buf, binary.LittleEndian, cmd)
  binary.Write(buf, binary.LittleEndian, ack)
  crc16 := CRC16(buf.Bytes())
  binary.Write(buf, binary.LittleEndian, crc16)

  return buf.Bytes()
}

func writeAck(conn net.Conn, info PackageHeadInfo, cmd uint8, ack uint8) {
  ackPackage := packAckPackage(info, cmd, ack)
  conn.Write(ackPackage)
}

func packLockPackage(lock uint8, strSN string) []byte {
  buf := new(bytes.Buffer)

  var sn [16]byte
  copy(sn[:], strSN)

  vl := uint16(1<<10) | uint16(1+2+1+16+1+2)

  binary.Write(buf, binary.LittleEndian, PACKAGE_HEAD)
  binary.Write(buf, binary.LittleEndian, vl)
  binary.Write(buf, binary.LittleEndian, APP_SEND_LOCK_CMD)
  binary.Write(buf, binary.LittleEndian, sn)
  binary.Write(buf, binary.LittleEndian, lock)

  crc16 := CRC16(buf.Bytes())
  binary.Write(buf, binary.LittleEndian, crc16)

  return buf.Bytes()
}

func packMsgPackage(msg string, strSN string) []byte {
  buf := new(bytes.Buffer)

  vl := uint16(1<<10) | uint16(1+1+len(msg)+1+16+1+2)

  var sn [16]byte
  copy(sn[:], strSN)

  var tmp [128]byte
  copy(tmp[:], msg)
  content := bytes.TrimRight(tmp[:], string(0))

  binary.Write(buf, binary.LittleEndian, PACKAGE_HEAD)
  binary.Write(buf, binary.LittleEndian, vl)
  binary.Write(buf, binary.LittleEndian, APP_SEND_MSG_CMD)
  binary.Write(buf, binary.LittleEndian, sn)
  binary.Write(buf, binary.LittleEndian, content)

  crc16 := CRC16(buf.Bytes())
  binary.Write(buf, binary.LittleEndian, crc16)

  return buf.Bytes()
}

func cmd_login_handler(is_login *bool, flyer *FlyerInfo, head *PackageHeadInfo, buf []byte, conn net.Conn) error {
  token_len := head.DataLen - 4 - 16 - 2

  id := buf[:15]
  token := buf[16 : 16 + token_len]

  hardware_id := strings.TrimRight(string(id[0:]), string(0))
  if len(hardware_id) == 0 {
    writeAck(conn, *head, APP_AUTH_ACK_CMD, 0)
    return errors.New("empty hardware id")
  }

  hardwareInfo, err := service.dbMgr.getHardwareInfoBySn(hardware_id)
  if err != nil {
    writeAck(conn, *head, APP_AUTH_ACK_CMD, 0)
    return err
  }

  teamName, captain, err := service.dbMgr.getTeamNameById(hardwareInfo.TeamID)
  if err != nil {
    writeAck(conn, *head, APP_AUTH_ACK_CMD, 0)
    return err
  }

  managerUids, err := service.dbMgr.getManagerUidsByHardwareSN(hardware_id)
  if err != nil {
    writeAck(conn, *head, APP_AUTH_ACK_CMD, 0)
    return err
  }

  flyer.HardwareSN = hardware_id
  flyer.BossID = hardwareInfo.BossID
  flyer.BossName = hardwareInfo.BossName
  flyer.HardwareName = hardwareInfo.HardwareName
  flyer.TeamID = hardwareInfo.TeamID
  flyer.TeamName = teamName
  flyer.Captain = captain
  flyer.Managers = managerUids
  flyer.HardwareType = hardwareInfo.HardwareType
  flyer.HardwareLocked = hardwareInfo.HardwareLocked

  userInfo, err := service.djiGateWay.CheckToken(string(token), conn.RemoteAddr().String())
  fmt.Println(userInfo, err)

  if err != nil || userInfo.Status != 0 || len(userInfo.Items) == 0 || userInfo.Items[0].Item.UserID == 0 {
    fmt.Println("app auth failed", err, userInfo)
    writeAck(conn, *head, APP_AUTH_ACK_CMD, 0)
    return errors.New("app auth failed")
  } else {
    flyer.UserID = userInfo.Items[0].Item.UserID
    flyer.Email = userInfo.Items[0].Item.Email
    flyer.UserName, flyer.Job_level, _ = service.dbMgr.GetUserInfoByUid(flyer.UserID, flyer.BossID)
  }

  //查询flight表获得这架飞机当天的工作总量
  flyer.TodayWork, _ = service.dbMgr.GetDroneTodayWork(flyer.HardwareSN)
  flyer.Token = string(token)

  //service.dbMgr.RecordOnlineInfo(flyer)
  service.dbMgr.UpdateDroneStatus(flyer.HardwareSN, 1)

  info := service.InitConnectionInfo(conn, *flyer)

  go service.CheckAndSendLockTimeMsg(conn, *flyer, info) //时间段锁定
  go service.CheckAndSendLockMsg(conn, *flyer, info)     //实时锁定

  *is_login = true

  writeAck(conn, *head, APP_AUTH_ACK_CMD, 1)

  return nil
}

func cmd_msgrsp_handler(is_login *bool, flyer *FlyerInfo, head *PackageHeadInfo, buf []byte, conn net.Conn) error {
  if !*is_login {
    return errors.New("cmd_msgrsp_handler, not login")
  }

  msgAck := buf[0]

  ci, ok := service.connMgr.GetConnInfoByConn(conn)
  if !ok {
    return errors.New("cmd_msgrsp_handler, get conn info failed")
  }

  select {
  case <- ci.msgStatus:
  default:
  }

  ci.msgStatus <- msgAck

  return nil
}

func cmd_lockrsp_handler(is_login *bool, flyer *FlyerInfo, head *PackageHeadInfo, buf []byte, conn net.Conn) error {
  if !*is_login {
    return errors.New("cmd_lockrsp_handler, not login")
  }

	lockAck := buf[0]

  ci, ok := service.connMgr.GetConnInfoByConn(conn)
  if !ok {
    return errors.New("cmd_lockrsp_handler, get conn info failed")
  }

	select {
	  case <- ci.lockStatus:
	  default:
	}

	ci.lockStatus <- lockAck

  return nil
}

func cmd_timelockrsp_handler(is_login *bool, flyer *FlyerInfo, head *PackageHeadInfo, buf []byte, conn net.Conn) error {
  if !*is_login {
    return errors.New("cmd_timelockrsp_handler, not login")
  }

	lockAck := buf[0]

  ci, ok := service.connMgr.GetConnInfoByConn(conn)
  if !ok {
    return errors.New("cmd_timelockrsp_handler, get conn info failed")
  }

	select {
		case <- ci.timelockStatus:
		default:
	}

	ci.timelockStatus <- lockAck

  return nil
}

func cmd_upload_handler(is_login *bool, flyer *FlyerInfo, head *PackageHeadInfo, buf []byte, conn net.Conn) error {
  if !*is_login {
    return errors.New("cmd_upload_handler, not login")
  }

  var iuavFlightData IuavFlightData

  newbuf := bytes.NewReader(buf)

  switch head.Ver {
  case 1:
    var gpsData GpsDataV1
    err := binary.Read(newbuf, binary.LittleEndian, &gpsData)
    if err != nil {
      return errors.New("cmd_upload_handler, parse v1 packet failed")
    }
    iuavFlightData = gpsData.ConvertToDBInfo(*flyer)

  case 2:
    var gpsData GpsDataV2
    err := binary.Read(newbuf, binary.LittleEndian, &gpsData)
    if err != nil {
      return errors.New("cmd_upload_handler, parse v2 packet failed")
    }
    iuavFlightData = gpsData.ConvertToDBInfo(*flyer)

  case 3:
    var gpsData GpsDataV3
    err := binary.Read(newbuf, binary.LittleEndian, &gpsData)
    if err != nil {
      return errors.New("cmd_upload_handler, parse v3 packet failed")
    }
    iuavFlightData = gpsData.ConvertToDBInfo(*flyer)

  default:
    return errors.New("cmd_upload_handler, unknown version")
  }

  err := service.dbMgr.PushToIuavFlightData(iuavFlightData)
  if err != nil {
    fmt.Println("push to iuav flight data queue failed", err)
  }

  if iuavFlightData.FrameFlag == 1 {
    service.dbMgr.UpdateDroneStatus(flyer.HardwareSN, 0)
  } else {
    service.dbMgr.UpdateDroneStatus(flyer.HardwareSN, 2)
  }

  //如果连接了websocket则发送给websocket，实现实时监控，通过bossid来判断连接，每一秒都要判断
  v, ok := ws.SafeMapGet(ws.connectionsMap, flyer.BossID)
  if ok {
    //获取这个conn以后，在sendwebsocket的时候conn或许已经被改变了？？slice可能会变
    ws.WsocketSendIuavFlightData(v, *flyer, iuavFlightData)
  }

  if len(flyer.Captain) > 1 {
    //如果有队长登录则发送给队长
    if v, ok = ws.SafeMapGet(ws.connectionsMap, flyer.Captain); ok {
      ws.WsocketSendIuavFlightData(v, *flyer, iuavFlightData)
    }
  }

  //如果有超级账号登录则发送给超级账号
  if v, ok = ws.SafeMapGet(ws.connectionsMap, _SUPER_UID_); ok {
    ws.WsocketSendIuavFlightData(v, *flyer, iuavFlightData)
  }

  //如果有管理者登录则发送给管理者
  if len(flyer.Managers) > 0 {
    for _, managerUid := range flyer.Managers {
      if v, ok = ws.SafeMapGet(ws.connectionsMap, managerUid); ok {
        ws.WsocketSendIuavFlightData(v, *flyer, iuavFlightData)
      }
    }
  }

	if GetConfigInstance().ENV == "test" {
		//非线上环境,回ACK
		conn.Write(packAckPackage(*head, head.CmdID, uint8(iuavFlightData.FrameIndex)))
	}

	return nil
}

func cmd_once_workinfo_handler(is_login *bool, flyer *FlyerInfo, head *PackageHeadInfo, buf []byte, conn net.Conn) error {
  if !*is_login {
    return errors.New("cmd_once_workinfo_handler, not login")
  }

  var once_workinfo OnceWorkInfo

  newbuf := bytes.NewReader(buf)

  err := binary.Read(newbuf, binary.LittleEndian, &once_workinfo)
  if err != nil {
    return errors.New("cmd_upload_handler, parse v1 packet failed")
  }

  fmt.Println(once_workinfo)

  return nil
}

func (s *Service) LockTimePackage(info LockResult) ([]byte, int64) {
	buf := new(bytes.Buffer)
	var err error
	bwrite := func(v interface{}) {
		if err == nil {
			err = binary.Write(buf, binary.LittleEndian, v)
		}
	}
	var sn [16]byte
	copy(sn[:], info.SN)
	vl := uint16(1<<10) | uint16(1+1+17+len(info.BossName)+1+16+1+2)
	bwrite(PACKAGE_HEAD)
	bwrite(vl)
	bwrite(APP_SEND_LOCKTIME_CMD)
	bwrite(sn)
	//添加lock指令
	if info.Cmd == WS_MSG_LOCK {
		bwrite(LOCKED)
	} else {
		bwrite(UNLOCKED)
	}
	//添加锁定时间段 格式 1970年至今的毫秒数
	t1, _ := strconv.ParseInt(info.Lock_begin, 10, 64) //string to int64
	bwrite(t1)
	t2, _ := strconv.ParseInt(info.Lock_end, 10, 64) //string to int64
	bwrite(t2)

	var name [50]byte
	copy(name[:], info.BossName) //写入老板名称
	bossname := bytes.TrimRight(name[:], "\x00")
	bwrite(bossname)

	crc16 := CRC16(buf.Bytes())
	bwrite(crc16)
	if err != nil {
    fmt.Println("LockTimePackage failed", info.SN, info.Cmd, err)
	}
	return buf.Bytes(), t1
}

func (s *Service) SendLockTimePeriod(conn net.Conn, info LockResult) error {
	p, _ := s.LockTimePackage(info)
	/*	today := time.Now().Unix() * 1000
		if t > 0 && today > t { //如果时间段锁定从今天算起，那么今天就给实时监管发送lock信息
			if v, ok := ws.SafeMapGet(ws.connectionsMap, info.BossID); ok {
				for _, c := range v {
					ws.wsCmdHardware(c, info.SN, LOCK) //发给网页  delete
				}
			}
		}*/
	/*	if info.Cmd == WS_UNLOCK { //如果是解锁，给web发送解锁信息
		if v, ok := ws.SafeMapGet(ws.connectionsMap, info.BossID); ok {
			for _, c := range v {
				ws.wsCmdHardware(c, info.SN, UNLOCK) //发给网页
			}
		}
	}*/

	_, err := conn.Write(p) //发给app
	if err != nil {
    fmt.Println("SendLockTimePeriod failed", info.Cmd, info.SN, err)
	}
	return err
}

func (s *Service) SendLockMsg(conn net.Conn, info LockResult) error {
	var cmd uint8
	if info.Cmd == WS_MSG_LOCK {
		cmd = LOCKED
	} else {
		cmd = UNLOCKED
	}
	p := packLockPackage(cmd, info.SN)

	_, err := conn.Write(p) //发给app
	if err != nil {
    fmt.Println("SendLockTimePeriod failed", info.Cmd, info.SN, err)
	}
	return err
}
