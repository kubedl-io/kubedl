package workloadgate

import (
	"flag"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

const (
	envWorkloadEnable = "WORKLOADS_ENABLE"
	autoDetectOption  = "auto"
)

var (
	workloads       = flag.String("workloads", autoDetectOption, "List of workloads to be enabled in cluster, `*` indicates enables all, and `-foo` indicates disable workload foo.")
	discoveryClient discovery.DiscoveryInterface
)

func IsWorkloadEnable(workload runtime.Object, scheme *runtime.Scheme) (kind string, enabled bool) {
	gvk, err := apiutil.GVKForObject(workload, scheme)
	if err != nil {
		klog.Warningf("unrecognized workload object %+v in scheme: %v", gvk, err)
		return "", false
	}

	var (
		workloadKind = gvk.Kind
		enables      = make(map[string]bool)
		enableAll    = false
	)

	// Parse enabling from cmd args settings firstly.
	if workloads != nil {
		if (*workloads) == autoDetectOption {
			return workloadKind, workloadCRDInstalled(gvk)
		}
		enables, enableAll = parseWorkloadsEnabled(*workloads)
	}

	// Env settings prioritized, if env has set it will override cmd settings.
	envWorkloads := os.Getenv(envWorkloadEnable)
	if envWorkloads != "" {
		if envWorkloads == autoDetectOption {
			return workloadKind, workloadCRDInstalled(gvk)
		}
		enables, enableAll = parseWorkloadsEnabled(envWorkloads)
	}

	if enableAll {
		return workloadKind, true
	}
	_, enable := enables[workloadKind]
	return workloadKind, enable
}

func parseWorkloadsEnabled(workloads string) (map[string]bool, bool) {
	enableAll := false
	enables := make(map[string]bool)
	for _, workload := range strings.Split(workloads, ",") {
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

func workloadCRDInstalled(gvk schema.GroupVersionKind) bool {
	if discoveryClient == nil {
		return true
	}
	crdList, err := discoveryClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		klog.Warningf("workload CRD %s not found in discovery, err: %s", gvk, err)
		return false
	}
	for _, crd := range crdList.APIResources {
		if crd.Kind == gvk.Kind {
			klog.Infof("workload CRD %s found.", crd.Kind)
			return true
		}
	}
	return false
}

func init() {
	if os.Getenv("KUBEDL_CI") != "true" {
		discoveryClient = discovery.NewDiscoveryClientForConfigOrDie(ctrl.GetConfigOrDie())
	}
}
