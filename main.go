package main

import (
	"context"
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

var _ sdk.DeploymentPlugin[config, deployTargetConfig, applicationConfig] = plugin{}

type plugin struct{}

const (
	stageDiff     = "FILE_DIFF"
	stageSync     = "FILE_SYNC"
	stageRollback = "FILE_ROLLBACK"
)

func (plugin) FetchDefinedStages() []string {
	return []string{
		stageDiff,
		stageSync,
		stageRollback,
	}
}

func (plugin) DetermineVersions(_ context.Context, _ *config, input *sdk.DetermineVersionsInput[applicationConfig]) (*sdk.DetermineVersionsResponse, error) {
	return &sdk.DetermineVersionsResponse{
		Versions: []sdk.ArtifactVersion{{Version: input.Request.DeploymentSource.CommitHash}},
	}, nil
}

func (plugin) DetermineStrategy(context.Context, *config, *sdk.DetermineStrategyInput[applicationConfig]) (*sdk.DetermineStrategyResponse, error) {
	panic("unimplemented")
}
func (plugin) BuildPipelineSyncStages(context.Context, *config, *sdk.BuildPipelineSyncStagesInput) (*sdk.BuildPipelineSyncStagesResponse, error) {
	panic("unimplemented")
}
func (plugin) BuildQuickSyncStages(context.Context, *config, *sdk.BuildQuickSyncStagesInput) (*sdk.BuildQuickSyncStagesResponse, error) {
	panic("unimplemented")
}
func (plugin) ExecuteStage(context.Context, *config, []*sdk.DeployTarget[deployTargetConfig], *sdk.ExecuteStageInput[applicationConfig]) (*sdk.ExecuteStageResponse, error) {
	panic("unimplemented")
}
