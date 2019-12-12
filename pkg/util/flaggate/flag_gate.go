package flaggate

import (
	"flag"
	"strings"
)

var (
	workloads = flag.String("workloads", "*", "List of workloads to be enabled in cluster, `*` indicates enables all, and `-foo` indicates disable workload foo.")
)

func IsWorkloadEnable(workload string) bool {
	enables, enableAll := parseWorkloadList(workloads)
	if enableAll {
		return enableAll
	}
	_, enable := enables[workload]
	return enable
}

func parseWorkloadList(workloads *string) (map[string]bool, bool) {
	if workloads == nil {
		return nil, false
	}
	enableAll := false
	enables := make(map[string]bool)
	for _, workload := range strings.Split(*workloads, ",") {
		workload = strings.TrimSpace(workload)
		enable := true
		// workload flag starts with `-` indicates disabling.
		if strings.HasPrefix(workload, "-") {
			enable = false
			workload = workload[1:]
		}
		// `*` indicates enable all supported workloads.
		if workload == "*" {
			// If enable==false, which means the original flag is `-*`, indicates disabling.
			if enable {
				enableAll = true
			}
			continue
		}
		if workload == "" {
			continue
		}
		enables[workload] = enable
	}
	return enables, enableAll
}
