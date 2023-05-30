# FLAGRC

Flagrc is a Flagr client for evaluator only. 
The main difference with https://github.com/checkr/goflagr is that,

flagrc act as a local evaluator allowing you to instead of calling flagr server for each request, It will resolve all your request locally.
In order to do that flagrc load in memory, each EvalCacheRefreshInterval (by default 3) second (in backgound) the configuration from the main server, then It can resolve/evaluate all request locally.

flagrc uses all flagr function to act as a a real flagr Evaluator.
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
BenchmarkFlagrc_WithEntityID-8            500000              3413 ns/op
PASS
ok      github.com/tkanos/flagrc        7.779s
```

Todo : 
- unit tests (how to mock client)
