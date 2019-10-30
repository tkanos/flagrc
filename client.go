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
}

type ClientOptions struct {
	EvalCacheRefreshTimeout time.Duration
}

func NewClient(cfg *goflagr.Configuration, options ...func(t *ClientOptions)) Evaluator {

	clienConfig := &ClientOptions{}

	for _, option := range options {
		option(clienConfig)
	}

	config.Config.EvalOnlyMode = true
	config.Config.DBDriver = "json_http"
	config.Config.DBConnectionStr = cfg.BasePath + "/export/eval_cache/json"

	if clienConfig.EvalCacheRefreshTimeout != 0 {
		config.Config.EvalCacheRefreshTimeout = clienConfig.EvalCacheRefreshTimeout
	}

	ec := handler.GetEvalCache()
	ec.Start()

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{
			Timeout: config.Config.EvalCacheRefreshTimeout,
		}
	}

	e := evaluator{
		client: goflagr.NewAPIClient(cfg),
	}

	return &e
}

type evaluator struct {
	client *goflagr.APIClient
}

func (e *evaluator) PostEvaluation(ctx context.Context, body goflagr.EvalContext) (goflagr.EvalResult, *http.Response, error) {
	// Evaluate locally
	//https://github.com/checkr/flagr/blob/master/pkg/handler/eval.go

	evalContext := models.EvalContext{
		EnableDebug:   body.EnableDebug,
		EntityContext: *body.EntityContext,
		EntityID:      body.EntityID,
		EntityType:    body.EntityType,
		FlagID:        body.FlagID,
		FlagKey:       body.FlagKey,
	}

	evalResult := handler.EvalFlag(evalContext)

	return toGloflagrEvalResult(evalResult, body), nil, nil

}

func (e *evaluator) PostEvaluationBatch(ctx context.Context, body goflagr.EvaluationBatchRequest) (goflagr.EvaluationBatchResponse, *http.Response, error) {
	// EvaluateBatch locally
	//https://github.com/checkr/flagr/blob/master/pkg/handler/eval.go

	entities := body.Entities
	flagIDs := body.FlagIDs
	flagKeys := body.FlagKeys
	results := goflagr.EvaluationBatchResponse{}

	// TODO make it concurrent
	for _, entity := range entities {
		for _, flagID := range flagIDs {
			evalContext := models.EvalContext{
				EnableDebug:   body.EnableDebug,
				EntityContext: *entity.EntityContext,
				EntityID:      entity.EntityID,
				EntityType:    entity.EntityType,
				FlagID:        flagID,
			}
			evalResult := handler.EvalFlag(evalContext)
			results.EvaluationResults = append(results.EvaluationResults, toGloflagrEvalResult(evalResult, goflagr.EvalContext{
				EntityID:      entity.EntityID,
				EntityType:    entity.EntityType,
				EntityContext: entity.EntityContext,
				EnableDebug:   body.EnableDebug,
				FlagID:        flagID,
			}))
		}
		for _, flagKey := range flagKeys {
			evalContext := models.EvalContext{
				EnableDebug:   body.EnableDebug,
				EntityContext: *entity.EntityContext,
				EntityID:      entity.EntityID,
				EntityType:    entity.EntityType,
				FlagKey:       flagKey,
			}
			evalResult := handler.EvalFlag(evalContext)
			results.EvaluationResults = append(results.EvaluationResults, toGloflagrEvalResult(evalResult, goflagr.EvalContext{
				EntityID:      entity.EntityID,
				EntityType:    entity.EntityType,
				EntityContext: entity.EntityContext,
				EnableDebug:   body.EnableDebug,
				FlagKey:       flagKey,
			}))
		}
	}

	return results, nil, nil
}

func toGloflagrEvalResult(evalResult *models.EvalResult, goflagrContext goflagr.EvalContext) goflagr.EvalResult {
	if evalResult == nil {
		return goflagr.EvalResult{}
	}

	return goflagr.EvalResult{
		FlagID:            evalResult.FlagID,
		FlagKey:           evalResult.FlagKey,
		FlagSnapshotID:    evalResult.FlagSnapshotID,
		SegmentID:         evalResult.SegmentID,
		VariantID:         evalResult.VariantID,
		VariantKey:        evalResult.VariantKey,
		VariantAttachment: &evalResult.VariantAttachment,
		EvalContext:       &goflagrContext,
		Timestamp:         evalResult.Timestamp,
		EvalDebugLog:      toGloflagrEvalDebugLog(evalResult.EvalDebugLog),
	}
}

func toGloflagrEvalDebugLog(evalDebugLog *models.EvalDebugLog) *goflagr.EvalDebugLog {
	if evalDebugLog == nil {
		return nil
	}
	debugLog := goflagr.EvalDebugLog{
		Msg: evalDebugLog.Msg,
	}

	for _, v := range evalDebugLog.SegmentDebugLogs {
		debugLog.SegmentDebugLogs = append(debugLog.SegmentDebugLogs, goflagr.SegmentDebugLog{
			SegmentID: v.SegmentID,
			Msg:       v.Msg,
		})
	}

	return &debugLog
}
