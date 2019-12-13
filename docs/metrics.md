## Expand Metrics

This metric package provide expanded and practical prometheus metrics for job 
workloads, these metrics are necessary for workloads monitoring but raw metrics
collecting methods are hard to achieve.

#### Job Running

`JobRunning` metric gauges for current number of running jobs in cluster, each 
workload should have its own gauger. Gauge will be triggered at most `interval`
seconds, and will try to trigger when new workloads created/deleted.

```go
// Example:
tfJobRunningGauge = NewTFJobRunningGauge(
    prometheus.GaugeOpts{
    		Name: "tf_operator_jobs_running",
    		Help: "Number of TF jobs running in cluster",
    },
    clientSet,
    30*time.Seconds,
)

// ...

tfJobRunningGauge.Gauge()
``` 
 
#### Launch Time

`LaunchTime` metric gauge for launch time of each job instance. Launch time stands
for `CondititionLastTransitionTime.Time - status.StartTime.Time`. Launch time can
reflect the duration from job submitted to job actually launched in cluster, indirectly
reflect the scheduling throughput and resource allocation efficiency.
Each job instance will be gauged one time in whole lifecycle.

```go
// Example:
tfJobLaunchTimeGauge = NewLaunchTimeGauge(
    prometheus.GaugeOpts{
    		Name: "tf_operator_jobs_running",
    		Help: "Number of TF jobs running in cluster",
    },
)

// ...

tfJobLauncTimeGauge.Gauge(tfjob, tfjob.status, "Tensorflow")
```
