package main

import (
	"log"

	sdk "github.com/pipe-cd/piped-plugin-sdk-go"
)

type (
	// We don't need plugin-wide config, so we can use an empty struct.
	config struct{}
	// We don't need deploy target config, so we can use an empty struct.
	// When we need some configs like `targetMachine` or `targetKubernetesCluster`, we can add them here.
	deployTargetConfig struct{}
	// We can define the application config here.
	// This config will be used to configure the application that this plugin will deploy.
	applicationConfig struct {
		// Path is the path to the destination directory where the files will be copied.
		Path string `json:"path"`
	}
)

func main() {
	plugin, err := sdk.NewPlugin[config, deployTargetConfig, applicationConfig]("0.0.1")
	if err != nil {
		log.Fatalln(err)
	}

	if err := plugin.Run(); err != nil {
		log.Fatalln(err)
	}
}
