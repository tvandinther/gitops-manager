package main

import (
	"github.com/tvandinther/gitops-manager/pkg/server"
)

func main() {
	server := server.New().WithDefaultLogger()
	server.Run()
}
