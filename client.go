package flagrc

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/checkr/goflagr"
	"github.com/openflagr/flagr/pkg/config"
	"github.com/openflagr/flagr/pkg/handler"
	"github.com/openflagr/flagr/swagger_gen/models"
)

type Evaluator interface {
	PostEvaluation(ctx context.Context, body goflagr.EvalContext) (goflagr.EvalResult, *http.Response, error)
	PostEvaluationBatch(ctx context.Context, body goflagr.EvaluationBatchRequest) (goflagr.EvaluationBatchResponse, *http.Response, error)
}

type ClientOptions struct {
	EvalCacheRefreshInterval time.Duration
	LogrusLevel              string
	LogrusFormat             string
	EvalLoggingEnabled       bool
}

type singleton struct {
	Evaluator
}

var once sync.Once
var onceUponATime sync.Once
var instance *singleton

func NewClient(cfg *goflagr.Configuration, options ...func(t *ClientOptions)) (ev Evaluator) {
	defer func() {
		if r := recover(); r != nil {
			once.Do(func() {
				instance = &singleton{
					Evaluator: &defaultEvaluator{
						cfg: *cfg,
					},
				}
			})

			ev = instance

			go tryToConnect(cfg)

		}
	}()

	clienConfig := &ClientOptions{}

	for _, option := range options {
		option(clienConfig)
	}

	config.Config.PProfEnabled = false
	config.Config.EvalOnlyMode = true
	config.Config.DBDriver = "json_http"
	config.Config.DBConnectionStr = cfg.BasePath + "/export/eval_cache/json"
	config.Config.EvalLoggingEnabled = clienConfig.EvalLoggingEnabled

	if clienConfig.EvalCacheRefreshInterval != 0 {
		config.Config.EvalCacheRefreshInterval = clienConfig.EvalCacheRefreshInterval
	}

	if clienConfig.LogrusLevel != "" {
		config.Config.LogrusLevel = clienConfig.LogrusLevel
	}

	if clienConfig.LogrusFormat != "" {
		config.Config.LogrusFormat = clienConfig.LogrusFormat
	}

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{
			Timeout: config.Config.EvalCacheRefreshTimeout,
		}
	}

	e := startEvaluation(cfg)

	once.Do(func() {
		instance = &singleton{
			Evaluator: e,
		}
	})

	ev = instance

	return
}

func startEvaluation(cfg *goflagr.Configuration) *evaluator {
	ec := handler.GetEvalCache()
	ec.Start()

	e := evaluator{
		client: goflagr.NewAPIClient(cfg),
	}

	return &e
}

func tryToConnect(cfg *goflagr.Configuration) {

	//Create Ticker
	ticker := time.NewTicker(config.Config.EvalCacheRefreshInterval)

	//Create a function that will startEvaluation and recover if panic
	f := func(cfg *goflagr.Configuration) (b bool) {
		defer func() {
			if r := recover(); r != nil {
				b = false
			}
		}()

		e := startEvaluation(cfg)
		if e != nil { // if the evaluation works, we set the new evaluator
			onceUponATime.Do(func() {
				instance.Evaluator = e
			})
			b = true
		}
		return
	}

	// Tick and call f functions that will try to connect to the client
	go func() {
		for {
			select {
			case _ = <-ticker.C:
				b := f(cfg)
				if b == true {
					//if we succeed to connect we close the go routine
					return
				}
			}
		}
	}()
}

type evaluator struct {
	client *goflagr.APIClient
}

func (e evaluator) PostEvaluation(ctx context.Context, body goflagr.EvalContext) (goflagr.EvalResult, *http.Response, error) {
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

func (e evaluator) PostEvaluationBatch(ctx context.Context, body goflagr.EvaluationBatchRequest) (goflagr.EvaluationBatchResponse, *http.Response, error) {
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

type defaultEvaluator struct {
	cfg goflagr.Configuration
}

var ErrNoServerAvailable error = errors.New("Server is not available")

func (e defaultEvaluator) PostEvaluation(ctx context.Context, body goflagr.EvalContext) (goflagr.EvalResult, *http.Response, error) {
	return goflagr.EvalResult{}, nil, ErrNoServerAvailable
}

func (e defaultEvaluator) PostEvaluationBatch(ctx context.Context, body goflagr.EvaluationBatchRequest) (goflagr.EvaluationBatchResponse, *http.Response, error) {
	return goflagr.EvaluationBatchResponse{}, nil, ErrNoServerAvailable
}
