package main

import (
  "fmt"
  _ "github.com/go-sql-driver/mysql"
  "os"
  "os/signal"
  "syscall"
  "flag"
)

var service *Service
var ws      *Wsocket

var VERSION = "test version"

func main() {
  showVersion := flag.Bool("version", false, "show version")
  flag.Parse()

  if *showVersion {
    fmt.Println(VERSION)
    return
  }

  GetConfigInstance().init()
  
  service = NewService()
  service.Start()

  ws = NewWsocket()
  ws.Start()

  ch := make(chan os.Signal)
  signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
  fmt.Println(<-ch)
  fmt.Println("IUAVRealtime Server stop")
  service.Stop()
}
