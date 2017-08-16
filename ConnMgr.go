package main

import (
  "net"
  "sync"
)

type ConnectionInfo struct {
  conn           net.Conn
  hardwareID     string
  userID         uint64
  captain        string
  bossID         string
  lockStatus     chan uint8
  timelockStatus chan uint8
  msgStatus      chan uint8
}

type ConnMgr struct {
  lock               sync.Mutex
  connInfo           map[net.Conn]*ConnectionInfo
  hardwareConnInfo   map[string]*ConnectionInfo
}

func NewConnMgr() *ConnMgr {
  connMgr := new(ConnMgr)
  connMgr.connInfo = make(map[net.Conn]*ConnectionInfo)
  connMgr.hardwareConnInfo = make(map[string]*ConnectionInfo)

  return connMgr
}

func (c *ConnMgr) SetConnInfo(hardwareID string, conn net.Conn, info *ConnectionInfo) {
  c.lock.Lock()
  defer c.lock.Unlock()

  c.connInfo[conn] = info
  c.hardwareConnInfo[hardwareID] = info
}

func (c *ConnMgr) GetConnInfoByConn(conn net.Conn) (*ConnectionInfo, bool) {
  c.lock.Lock()
  defer c.lock.Unlock()

  info, ok := c.connInfo[conn]
  return info, ok
}

func (c *ConnMgr) GetConnInfoByHardwareID(hardwareID string) (*ConnectionInfo, bool) {
  c.lock.Lock()
  defer c.lock.Unlock()

  info, ok := c.hardwareConnInfo[hardwareID]

  return info, ok
}

func (c *ConnMgr) DelConnInfoByConn(conn net.Conn) {
  c.lock.Lock()
  defer c.lock.Unlock()

  info, ok := c.connInfo[conn]
  if ok {
    delete(c.hardwareConnInfo, info.hardwareID)
    delete(c.connInfo, conn)
  }
}

func (c *ConnMgr) DelConnInfoByHardwareID(hardwareID string) {
  c.lock.Lock()
  defer c.lock.Unlock()

  info, ok := c.hardwareConnInfo[hardwareID]
  if ok {
    delete(c.connInfo, info.conn)
    delete(c.hardwareConnInfo, info.hardwareID)
  }
}
