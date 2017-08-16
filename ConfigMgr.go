package main

import (
  "sync"
  "io/ioutil"
  "encoding/json"
)

type GlobalConfig struct {
  ENV                    string   `json:"env"`

  PublicPemPath          string   `json:"public_pem_path"`
  PrivatePemPath         string   `json:"private_pem_path"`

  ServerPort             uint16   `json:"server_port"`

  WebsocketPort          uint16   `json:"websocket_port"`

  WebsocketAuthToken     string   `json:"websocket_auth_token"`

  Gateway                string   `json:"gateway"`
  GatewayAppID           string   `json:"gateway_app_id"`
  GatewayAppKey          string   `json:"gateway_app_key"`

  Database               string   `json:"database"`
}

var instance *GlobalConfig
var once sync.Once

func GetConfigInstance() *GlobalConfig {
  once.Do(func() {
    instance = &GlobalConfig{}
  })

  return instance
}

func (config *GlobalConfig) init() error {
  buf, err := ioutil.ReadFile("config.json")

  if err != nil {
    return err
  }

  json.Unmarshal(buf, config)

  return nil
}
