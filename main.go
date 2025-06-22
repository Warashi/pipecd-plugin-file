package main

import (
	"context"
	"fmt"
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
	return nil, nil
}

func (plugin) BuildPipelineSyncStages(_ context.Context, _ *config, input *sdk.BuildPipelineSyncStagesInput) (*sdk.BuildPipelineSyncStagesResponse, error) {
	if len(input.Request.Stages) == 0 {
		return nil, fmt.Errorf("no stages defined in the request")
	}

	stages := make([]sdk.PipelineStage, 0, len(input.Request.Stages)+1) // +1 for the rollback stage
	for _, s := range input.Request.Stages {
		switch s.Name {
		case stageDiff:
			stages = append(stages, sdk.PipelineStage{
				Index: s.Index,
				Name:  stageDiff,
			})
		case stageSync:
			stages = append(stages, sdk.PipelineStage{
				Index: s.Index,
				Name:  stageSync,
			})
		default:
			return nil, fmt.Errorf("unknown stage: %s", s.Name)
		}
	}

	if input.Request.Rollback {
		// Find the minimum index from the defined stages to set the rollback stage index.
		// The rollback stages will be executed the order of the indexes when the pipeline is failed or canceled.
		// In this case, we want to execute the rollback stage in the order of the each first stage of the plugins.
		// So we need to find the minimum index from the defined stages.
		idx := input.Request.Stages[0].Index
		for _, s := range input.Request.Stages[1:] {
			if s.Index < idx {
				idx = s.Index
			}
		}
		stages = append(stages, sdk.PipelineStage{
			Index:    idx,
			Name:     stageRollback,
			Rollback: true,
		})
	}

	return &sdk.BuildPipelineSyncStagesResponse{
		Stages: stages,
	}, nil
}

func (plugin) BuildQuickSyncStages(_ context.Context, _ *config, input *sdk.BuildQuickSyncStagesInput) (*sdk.BuildQuickSyncStagesResponse, error) {
	stages := make([]sdk.QuickSyncStage, 0, 2)

	stages = append(stages, sdk.QuickSyncStage{
		Name:        stageSync,
		Description: "Sync stage", // Description is displayed in the UI.
	})

	if input.Request.Rollback {
		stages = append(stages, sdk.QuickSyncStage{
			Name:        stageRollback,
			Description: "Rollback stage",
			Rollback:    true,
		})
	}

	return &sdk.BuildQuickSyncStagesResponse{
		Stages: stages,
	}, nil
}

func (plugin) ExecuteStage(context.Context, *config, []*sdk.DeployTarget[deployTargetConfig], *sdk.ExecuteStageInput[applicationConfig]) (*sdk.ExecuteStageResponse, error) {
	panic("unimplemented")
}
