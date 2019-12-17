## Expand Metrics

This file describes all built-in expanded and practical prometheus metrics for job workloads
in KubeDL, and difference kinds of workloads is classified by labels.

|    Metric Names     |   Description    |
|    ------------     |   -----------    |
|    kubedl_jobs_created{kind}  | Counts number of jobs created |
|    kubedl_jobs_deleted{kind}  | Counts number of jobs deleted |
|    kubedl_jobs_successful{kind}  |  Counts number of jobs successfully finished  |
|    kubedl_jobs_failed{kind}      |   Counts number of jobs failed  |
|    kubedl_jobs_restarted{kind}   |   Counts number of jobs restarted  |
|    kubedl_running_jobs{kind}     |   Counts number of jobs running now  |
|    kubedl_launch_time{kind, name, namespace, uid}  |  Gauge for launch time(from created to truly running) of each job  |