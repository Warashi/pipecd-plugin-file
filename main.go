package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"maps"
	"os"
	"slices"

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

func (p plugin) ExecuteStage(ctx context.Context, _ *config, _ []*sdk.DeployTarget[deployTargetConfig], input *sdk.ExecuteStageInput[applicationConfig]) (*sdk.ExecuteStageResponse, error) {
	switch input.Request.StageName {
	case stageDiff:
		return p.executeStageDiff(ctx, input)
	case stageSync:
		return p.executeStageSync(ctx, input)
	case stageRollback:
		return p.executeStageRollback(ctx, input)
	default:
		return nil, fmt.Errorf("unknown stage: %s", input.Request.StageName)
	}
}

func (plugin) executeStageDiff(ctx context.Context, input *sdk.ExecuteStageInput[applicationConfig]) (*sdk.ExecuteStageResponse, error) {
	lp := input.Client.LogPersister()

	lp.Info("Listing files in the git repository...")
	sourceFiles, err := listFiles(os.DirFS(input.Request.TargetDeploymentSource.ApplicationDirectory))
	if err != nil {
		return nil, fmt.Errorf("error listing files: %w", err)
	}

	delete(sourceFiles, filepath.Base(input.Request.TargetDeploymentSource.ApplicationConfigFilename))

	lp.Info("Listing files in the target directory...")
	targetFiles, err := listFiles(os.DirFS(input.Request.TargetDeploymentSource.ApplicationConfig.Spec.Path))
	if err != nil {
		return nil, fmt.Errorf("error listing files: %w", err)
	}

	addedFiles := differenceFiles(sourceFiles, targetFiles)
	removedFiles := differenceFiles(targetFiles, sourceFiles)

	mergedFiles := maps.Clone(sourceFiles)
	maps.Copy(mergedFiles, targetFiles)

	diffFiles := make(map[string]struct{})
	for path := range mergedFiles {
		if _, ok := addedFiles[path]; ok {
			continue
		}

		if _, ok := removedFiles[path]; ok {
			continue
		}

		different, err := isFileContentDifferent(os.DirFS(input.Request.TargetDeploymentSource.ApplicationDirectory), os.DirFS(input.Request.TargetDeploymentSource.ApplicationConfig.Spec.Path), path)
		if err != nil {
			return nil, fmt.Errorf("error checking if file content is different: %w", err)
		}

		if different {
			diffFiles[path] = struct{}{}
		}
	}

	lp.Info("Summary of the file diff:")
	lp.Info("--------------------------------")
	lp.Info("Added files:")
	for _, path := range slices.Sorted(maps.Keys(addedFiles)) {
		lp.Info(path)
	}

	lp.Info("--------------------------------")
	lp.Info("Removed files:")
	for _, path := range slices.Sorted(maps.Keys(removedFiles)) {
		lp.Info(path)
	}

	lp.Info("--------------------------------")
	lp.Info("Changed files:")
	for _, path := range slices.Sorted(maps.Keys(diffFiles)) {
		lp.Info(path)
	}

	lp.Info("--------------------------------")

	lp.Success("File diff completed")

	return &sdk.ExecuteStageResponse{
		Status: sdk.StageStatusSuccess,
	}, nil
}

func (plugin) executeStageSync(ctx context.Context, input *sdk.ExecuteStageInput[applicationConfig]) (*sdk.ExecuteStageResponse, error) {
	panic("unimplemented")
}

func (plugin) executeStageRollback(ctx context.Context, input *sdk.ExecuteStageInput[applicationConfig]) (*sdk.ExecuteStageResponse, error) {
	panic("unimplemented")
}

func listFiles(f fs.FS) (map[string]struct{}, error) {
	files := make(map[string]struct{})

	if err := fs.WalkDir(f, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files[path] = struct{}{}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("error walking through files: %w", err)
	}

	return files, nil
}

// differenceFiles returns the files that are in a but not in b.
func differenceFiles(a, b map[string]struct{}) map[string]struct{} {
	differences := make(map[string]struct{})

	for path := range a {
		if _, ok := b[path]; !ok {
			differences[path] = struct{}{}
		}
	}

	return differences
}

// isFileContentDifferent returns true if the content of the file is different between a and b.
func isFileContentDifferent(a, b fs.FS, path string) (bool, error) {
	aFile, err := a.Open(path)
	if err != nil {
		return false, fmt.Errorf("error opening file %s: %w", path, err)
	}
	defer aFile.Close()

	bFile, err := b.Open(path)
	if err != nil {
		return false, fmt.Errorf("error opening file %s: %w", path, err)
	}
	defer bFile.Close()

	aContent, err := io.ReadAll(aFile)
	if err != nil {
		return false, fmt.Errorf("error reading file %s: %w", path, err)
	}

	bContent, err := io.ReadAll(bFile)
	if err != nil {
		return false, fmt.Errorf("error reading file %s: %w", path, err)
	}

	return !bytes.Equal(aContent, bContent), nil
}
