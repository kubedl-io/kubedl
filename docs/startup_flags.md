### Kubedl-operator Startup Flags


| Flag Name   |   Description    | Default |
|------------- |-------------| -----|
|controller-metrics-addr|The prometheus metrics endpoint for job stats| 8088 
enable-leader-election | Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager. | false
gang-scheduler-name |  The name of gang scheduler, by default it is set to empty meaning not enalbing gang scheduling | ""
max-reconciles |  The number of max concurrent reconciles of each controller | 1