package allude

import (
	"context"
	"fmt"
	"time"
)

type JobRunner struct {
	repository Repository
	service    *Service
	workerID   string
}

func NewJobRunner(repository Repository, service *Service, workerID string) *JobRunner {
	return &JobRunner{
		repository: repository,
		service:    service,
		workerID:   workerID,
	}
}

func (runner *JobRunner) Start(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = runner.ProcessNext(ctx)
		}
	}
}

func (runner *JobRunner) Drain(ctx context.Context, maxJobs int) error {
	for index := 0; index < maxJobs; index++ {
		processed, err := runner.ProcessNext(ctx)
		if err != nil {
			return err
		}
		if !processed {
			return nil
		}
	}
	return nil
}

func (runner *JobRunner) ProcessNext(_ context.Context) (bool, error) {
	job, err := runner.repository.LeasePendingJob(runner.workerID)
	if err != nil || job == nil {
		return false, err
	}

	if err := runner.handle(job); err != nil {
		_ = runner.repository.FailJob(job.ID, err.Error())
		return true, err
	}
	return true, runner.repository.CompleteJob(job.ID)
}

func (runner *JobRunner) handle(job *Job) error {
	switch job.Type {
	case JobEmbedThoughtVersion:
		versionID := job.Payload["thoughtVersionId"]
		versionThoughtID := job.Payload["thoughtId"]
		thought, err := runner.service.Thought(versionThoughtID)
		if err != nil {
			return err
		}
		var target *ThoughtVersion
		for _, version := range thought.Versions {
			if version.ID == versionID {
				target = version
				break
			}
		}
		if target == nil {
			return fmt.Errorf("version %s not found", versionID)
		}
		embedding, err := runner.service.ai.EmbedQuery(target.Content)
		if err != nil {
			return err
		}
		if _, err := runner.repository.SaveThoughtVersionEnrichment(versionID, embedding, nil, ProcessingProcessing, []string{"Embedding generated"}); err != nil {
			return err
		}
		_, err = runner.repository.EnqueueJob(&Job{
			Type:        JobExtractConcepts,
			EntityType:  "thought_version",
			EntityID:    versionID,
			Payload:     map[string]string{"thoughtVersionId": versionID, "thoughtId": versionThoughtID},
			MaxAttempts: 3,
		})
		return err
	case JobExtractConcepts:
		return runner.service.enrichThoughtVersion(job.Payload["thoughtVersionId"], job.Payload["thoughtId"])
	case JobLinkThought:
		return runner.service.linkThought(job.Payload["thoughtId"])
	case JobRefreshConceptSummary:
		return runner.service.refreshConceptSummary(job.Payload["conceptId"])
	default:
		return fmt.Errorf("unsupported job type %s", job.Type)
	}
}
