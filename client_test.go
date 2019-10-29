//+build bench

package flagrc

import (
	"context"
	"testing"

	"github.com/checkr/goflagr"
)

// Todo the bench you need to set up a Flag on your localhost flagr server
var Result string

var goflagrClient = goflagr.NewAPIClient(&goflagr.Configuration{
	BasePath: "http://localhost:18000/api/v1",
})

func BenchmarkGoFlagr_WithoutEntityID(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var ec interface{}
		ec = map[string]string{"country": "ca"}
		result, _, _ := goflagrClient.EvaluationApi.PostEvaluation(context.Background(), goflagr.EvalContext{FlagKey: "FirstFlag", EntityContext: &ec})
		Result = result.VariantKey
	}
}

func BenchmarkGoFlagr_WithEntityID(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var ec interface{}
		ec = map[string]string{"country": "ca"}
		result, _, _ := goflagrClient.EvaluationApi.PostEvaluation(context.Background(), goflagr.EvalContext{FlagKey: "FirstFlag", EntityContext: &ec, EntityID: "123"})
		Result = result.VariantKey
	}
}

var flagrcClient = NewClient(&goflagr.Configuration{
	BasePath: "http://localhost:18000/api/v1",
})

func BenchmarkFlagrc_WithoutEntityID(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var ec interface{}
		ec = map[string]interface{}{"country": "ca"}
		result, _, _ := flagrcClient.PostEvaluation(context.Background(), goflagr.EvalContext{FlagKey: "FirstFlag", EntityContext: &ec})
		Result = result.VariantKey
	}
}

func BenchmarkFlagrc_WithEntityID(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var ec interface{}
		ec = map[string]interface{}{"country": "ca"}
		result, _, _ := flagrcClient.PostEvaluation(context.Background(), goflagr.EvalContext{FlagKey: "FirstFlag", EntityContext: &ec, EntityID: "456"})
		Result = result.VariantKey
	}
}
