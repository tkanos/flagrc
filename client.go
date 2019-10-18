package flagrc

import (
	"context"
	"net/http"
	"time"

	"github.com/checkr/flagr/pkg/config"
	"github.com/checkr/flagr/pkg/handler"
	"github.com/checkr/flagr/swagger_gen/models"
	"github.com/checkr/goflagr"
)

type Evaluator interface {
	PostEvaluation(ctx context.Context, body goflagr.EvalContext) (goflagr.EvalResult, *http.Response, error)
	PostEvaluationBatch(ctx context.Context, body goflagr.EvaluationBatchRequest) (goflagr.EvaluationBatchResponse, *http.Response, error)

	withConfig
}

type withConfig interface {
	WithCacheTimeout(timeout time.Duration)
}

type evaluator struct {
	client *goflagr.APIClient
}

func NewClient(cfg *goflagr.Configuration) Evaluator {

	config.Config.EvalOnlyMode = true
	config.Config.DBDriver = "json_http"
	config.Config.DBConnectionStr = cfg.BasePath + "/flags?preload=true&enabled=true"

	ec := handler.GetEvalCache()
	ec.Start()

	e := evaluator{
		client: goflagr.NewAPIClient(cfg),
	}

	return &e
}

func (e *evaluator) WithCacheTimeout(timeout time.Duration) {
	config.Config.EvalCacheRefreshTimeout = timeout
}

func (e *evaluator) PostEvaluation(ctx context.Context, body goflagr.EvalContext) (goflagr.EvalResult, *http.Response, error) {
	if body.EntityID != "" {
		return e.client.EvaluationApi.PostEvaluation(ctx, body)
	}

	// Evaluate locally
	//https://github.com/checkr/flagr/blob/master/pkg/handler/eval.go

	evalContext := models.EvalContext{
		EnableDebug:   body.EnableDebug,
		EntityContext: body.EntityContext,
		EntityID:      body.EntityID,
		EntityType:    body.EntityType,
		FlagID:        body.FlagID,
		FlagKey:       body.FlagKey,
	}

	evalResult := evalFlag(evalContext)

	return toGloflagrEvalResult(evalResult), nil, nil

}

func (e *evaluator) PostEvaluationBatch(ctx context.Context, body goflagr.EvaluationBatchRequest) (goflagr.EvaluationBatchResponse, *http.Response, error) {
	for _, entity := range body.Entities {
		if entity.EntityID != "" {
			return e.client.EvaluationApi.PostEvaluationBatch(ctx, body)
		}
	}

	// EvaluateBatch locally
	//https://github.com/checkr/flagr/blob/master/pkg/handler/eval.go

	entities := body.Entities
	flagIDs := body.FlagIDs
	flagKeys := body.FlagKeys
	results := &goflagr.EvaluationBatchResponse{}

	// TODO make it concurrent
	for _, entity := range entities {
		for _, flagID := range flagIDs {
			evalContext := models.EvalContext{
				EnableDebug:   body.EnableDebug,
				EntityContext: entity.EntityContext,
				EntityID:      entity.EntityID,
				EntityType:    entity.EntityType,
				FlagID:        flagID,
			}
			evalResult := evalFlag(evalContext)
			results.EvaluationResults = append(results.EvaluationResults, toGloflagrEvalResult(evalResult))
		}
		for _, flagKey := range flagKeys {
			evalContext := models.EvalContext{
				EnableDebug:   body.EnableDebug,
				EntityContext: entity.EntityContext,
				EntityID:      entity.EntityID,
				EntityType:    entity.EntityType,
				FlagKey:       flagKey,
			}
			evalResult := evalFlag(evalContext)
			results.EvaluationResults = append(results.EvaluationResults, toGloflagrEvalResult(evalResult))
		}
	}

	return goflagr.EvaluationBatchResponse{}, nil, nil
}

func toGloflagrEvalResult(evalResult *models.EvalResult) goflagr.EvalResult {
	return goflagr.EvalResult{
		FlagID:            evalResult.FlagID,
		FlagKey:           evalResult.FlagKey,
		FlagSnapshotID:    evalResult.FlagSnapshotID,
		SegmentID:         evalResult.SegmentID,
		VariantID:         evalResult.VariantID,
		VariantKey:        evalResult.VariantKey,
		VariantAttachment: &evalResult.VariantAttachment,
		Timestamp:         evalResult.Timestamp,
	}
}
