package config

import (
	"os"
	"encoding/json"
)

const defConfigFileName = "config.json"

type Config struct {
	TelApiToken   string            `json:"token"`
	Proxy         *Proxy            `json:"proxy,omitempty"`
	ModelFileName string            `json:"modelFileName"`
	AlarmLogic    *QueuesAlarmLogic `json:"alarmLogic,omitempty"`
}

type Proxy struct {
	IpAddr   string `json:"ip"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type QueuesAlarmLogic struct {
	NormalQueues    *map[string]int `json:"normalQueues"`
	ExceptionQueues *map[string]int `json:"exceptionQueues"`
}

func NewConfig() *Config {
	return NewConfigWithCustomFile("")
}

func NewConfigWithCustomFile(fileName string) *Config {

	if fileName == "" {
		fileName = defConfigFileName
	}

	c := &Config{}

	file, err := os.Open(fileName)
	if err != nil {
		return c
	}

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&c)
	if err != nil {
		return c
	}
	return c
}
