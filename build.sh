#!/usr/bin/env bash
go build -ldflags "-X main.VERSION=`date +%Y%m%d.%H%M%S`" IUAVRealtimeServer.go ConfigMgr.go DataStruct.go DBModelDefine.go Protocol.go Util.go ConnMgr.go DBManager.go DJIGateWay.go ProtocolDefine.go Service.go Websocket.go WebsocketDefine.go
