# FLAGRC

Flagrc is a Flagr client for evaluator only. 
The main difference with https://github.com/checkr/goflagr is that, flagrc only calls the /api/v1/evaluation endpoints 
if your request has an EntityId on it.

If you use an EntityID, flagr will make sure that this ID will always receive the same variant. 
So flagrc ask to the server.
Otherwise flagrc, will act as a flagr evaluator :
- Loading in Memory all flags
- Do the evaluation of the request locally.
- Refresh each EvalCacheRefreshInterval the Flags for computation

flagrc uses all flagr function to act as a a real flagr Evaluator (except for EntityID).
flagrc uses all goflagr signature.


## Example 

```
    local := flagrc.NewClient(&goflagr.Configuration{
		BasePath: "http://localhost:18000/api/v1",
	})

    var ec interface{}
	ec = map[string]interface{}{"country": "ca"}
	result, _, err := local.PostEvaluation(ctx, goflagr.EvalContext{FlagKey: "HelloWorldFlag", EntityContext: &ec})

	if err == nil && result.VariantKey == "A" {
		fmt.Println("Hello Canada")
	}
```

## Benchmark

Comparison between flagrc and goflagr :

```
$ go test -bench=.
goos: linux
goarch: amd64
pkg: github.com/tkanos/flagrc
BenchmarkGoFlagr_WithoutEntityID-8           100          19656778 ns/op
BenchmarkGoFlagr_WithEntityID-8              100          19679028 ns/op
BenchmarkFlagrc_WithoutEntityID-8         500000              3393 ns/op
BenchmarkFlagrc_WithEntityID-8               100          19624108 ns/op
PASS
ok      github.com/tkanos/flagrc        7.779s
```

Todo : 
- unit tests (how to mock client)
- cache in a map the entityId response - enabled by config (default false)
