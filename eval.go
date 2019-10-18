package flagrc

import (
	"fmt"
	"math/rand"

	"github.com/checkr/flagr/pkg/config"
	"github.com/checkr/flagr/pkg/entity"
	"github.com/checkr/flagr/pkg/handler"
	"github.com/checkr/flagr/pkg/util"
	"github.com/checkr/flagr/swagger_gen/models"
	"github.com/davecgh/go-spew/spew"

	"github.com/jinzhu/gorm"
	"github.com/zhouzhuojie/conditions"
)

var evalFlag = func(evalContext models.EvalContext) *models.EvalResult {
	cache := handler.GetEvalCache()
	flagID := util.SafeUint(evalContext.FlagID)
	flagKey := util.SafeString(evalContext.FlagKey)
	f := cache.GetByFlagKeyOrID(flagID)
	if f == nil {
		f = cache.GetByFlagKeyOrID(flagKey)
	}

	if f == nil {
		emptyFlag := &entity.Flag{Model: gorm.Model{ID: flagID}, Key: flagKey}
		return handler.BlankResult(emptyFlag, evalContext, fmt.Sprintf("flagID %v not found or deleted", flagID))
	}

	if !f.Enabled {
		return handler.BlankResult(f, evalContext, fmt.Sprintf("flagID %v is not enabled", f.ID))
	}

	if len(f.Segments) == 0 {
		return handler.BlankResult(f, evalContext, fmt.Sprintf("flagID %v has no segments", f.ID))
	}

	if evalContext.EntityID == "" {
		evalContext.EntityID = fmt.Sprintf("randomly_generated_%d", rand.Int31())
	}

	if f.EntityType != "" {
		evalContext.EntityType = f.EntityType
	}

	logs := []*models.SegmentDebugLog{}
	var vID int64
	var sID int64

	for _, segment := range f.Segments {
		sID = int64(segment.ID)
		variantID, log, evalNextSegment := evalSegment(f.ID, evalContext, segment)
		if config.Config.EvalDebugEnabled && evalContext.EnableDebug {
			logs = append(logs, log)
		}
		if variantID != nil {
			vID = int64(*variantID)
		}
		if !evalNextSegment {
			break
		}
	}
	evalResult := handler.BlankResult(f, evalContext, "")
	evalResult.EvalDebugLog.SegmentDebugLogs = logs
	evalResult.SegmentID = sID
	evalResult.VariantID = vID
	v := f.FlagEvaluation.VariantsMap[util.SafeUint(vID)]
	if v != nil {
		evalResult.VariantAttachment = v.Attachment
		evalResult.VariantKey = v.Key
	}

	//logEvalResult(evalResult, f.DataRecordsEnabled)
	return evalResult
}

var evalSegment = func(
	flagID uint,
	evalContext models.EvalContext,
	segment entity.Segment,
) (
	vID *uint, // returns VariantID
	log *models.SegmentDebugLog,
	evalNextSegment bool,
) {
	if len(segment.Constraints) != 0 {
		m, ok := evalContext.EntityContext.(map[string]interface{})
		if !ok {
			log = &models.SegmentDebugLog{
				Msg:       fmt.Sprintf("constraints are present in the segment_id %v, but got invalid entity_context: %s.", segment.ID, spew.Sdump(evalContext.EntityContext)),
				SegmentID: int64(segment.ID),
			}
			return nil, log, true
		}

		expr := segment.SegmentEvaluation.ConditionsExpr
		match, err := conditions.Evaluate(expr, m)
		if err != nil {
			log = &models.SegmentDebugLog{
				Msg:       err.Error(),
				SegmentID: int64(segment.ID),
			}
			return nil, log, true
		}
		if !match {
			log = &models.SegmentDebugLog{
				//Msg:       debugConstraintMsg(evalContext.EnableDebug, expr, m),
				Msg:       "constraint not match",
				SegmentID: int64(segment.ID),
			}
			return nil, log, true
		}
	}

	vID, debugMsg := segment.SegmentEvaluation.DistributionArray.Rollout(
		evalContext.EntityID,
		fmt.Sprint(flagID), // default use the flagID as salt
		segment.RolloutPercent,
	)

	log = &models.SegmentDebugLog{
		Msg:       "matched all constraints. " + debugMsg,
		SegmentID: int64(segment.ID),
	}

	// at this point, all constraints are matched, so we shouldn't go to next segment
	// thus setting evalNextSegment = false
	return vID, log, false
}
