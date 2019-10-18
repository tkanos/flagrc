# FLAGRC

Flagrc is a Flagr client for evaluator only. 
The main difference with https://github.com/checkr/goflagr is that, flagrc only calls the /api/v1/evaluation endpoints 
if your request has an EntityId on it.

If you use an EntityID, flagr will make sure that this ID will always receive the same variant. 
So flagrc ask to the server.
Otherwise flagrc, will act as an flagr evaluator :
- Loading in Memory all flags
- Do the evaluation of the request locally.
- Load each EvalCacheRefreshInterval the Flags for computation

flagrc uses all flagr function to act as a a real flagr Evaluator (except for EntityID).


Todo : 
- unit tests (how to mock client)
- ask goflagr guys to have a review.
- benchmark
- add a real client to goflagr (on config)
- cache in a map the entityId response - enabled by config (default false)
