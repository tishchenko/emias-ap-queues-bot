package main

import (
	"github.com/tishchenko/emias-ap-queues-bot/config"
)

func main() {
	apBot := NewApBot(config.NewConfig())
	apBot.Run()
}