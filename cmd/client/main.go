package main

import "github.com/tvandinther/gitops-manager/pkg/client"

func main() {
	client := client.New()
	client.Run()
}
