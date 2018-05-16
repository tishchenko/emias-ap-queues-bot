package model

import (
	"time"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"strings"
)

const defModelFileName = "m.js"
const queuesDataStr = "queuesData = "

type Model struct {
	FileName        string                 `json:"-"`
	Progress        uint32                 `json:"progress"`
	NormalQueues    map[string][]QueueInfo `json:"normalQueues"`
	ExceptionQueues map[string][]QueueInfo `json:"exceptionQueues"`
}

type QueueInfo struct {
	DateTime       time.Time `json:"dateTime"`
	Length         *int `json:"length"`
}

func NewModel() *Model {
	return NewModelWithFileName("")
}

func NewModelWithFileName(fileName string) *Model {

	if fileName == "" {
		fileName = defModelFileName
	}

	m := &Model{}
	m.FileName = fileName
	return m
}

func (m *Model) Refresh() error {
	body, err := ioutil.ReadFile(m.FileName)
	if err != nil {
		return err
		fmt.Printf("File error: %v\n", err)
	}
	body = []byte(strings.TrimPrefix(string(body), queuesDataStr))
	json.Unmarshal(body, &m)

	return nil
}