/*
Copyright 2019 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"os"

	"k8s.io/apimachinery/pkg/util/net"

	"github.com/alibaba/kubedl/pkg/features"

	"github.com/spf13/pflag"

	"github.com/alibaba/kubedl/apis"
	"github.com/alibaba/kubedl/cmd/options"
	"github.com/alibaba/kubedl/controllers"
	"github.com/alibaba/kubedl/controllers/persist"
	cacheregistry "github.com/alibaba/kubedl/pkg/cache_backend/registry"
	"github.com/alibaba/kubedl/pkg/gang_schedule/registry"
	"github.com/alibaba/kubedl/pkg/metrics"
	backendregistry "github.com/alibaba/kubedl/pkg/storage/backends/registry"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
}

func main() {
	var (
		enableLeaderElection bool
		metricsAddr          int
		ctrlMetricsAddr      string
		hostPortRange        string
	)
	pflag.StringVar(&ctrlMetricsAddr, "controller-metrics-addr", ":8080", "The address the controller metric endpoint binds to.")
	pflag.IntVar(&metricsAddr, "metrics-addr", 8443, "The address the default endpoints binds to.")
	pflag.BoolVar(&enableLeaderElection, "enable-leader-election", true,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	pflag.StringVar(&options.CtrlConfig.GangSchedulerName, "gang-scheduler-name", "", "specify the name of gang scheduler")
	pflag.IntVar(&options.CtrlConfig.MaxConcurrentReconciles, "max-reconciles", 1, "specify the number of max concurrent reconciles of each controller")
	pflag.StringVar(&options.CtrlConfig.ModelImageBuilder, "model-image-builder", "kubedl/kaniko:latest", "The image name of container builder for building the model image")
	pflag.StringVar(&hostPortRange, "hostnetwork-port-range", "20000-30000", "hostnetwork port range for hostnetwork-enabled jobs")
	features.KubeDLFeatureGates.AddFlag(pflag.CommandLine)
	pflag.Parse()

	options.CtrlConfig.EnableGangScheduling = options.CtrlConfig.GangSchedulerName != ""
	options.CtrlConfig.HostNetworkPortRange = *net.ParsePortRangeOrDie(hostPortRange)

	if options.CtrlConfig.MaxConcurrentReconciles <= 0 {
		options.CtrlConfig.MaxConcurrentReconciles = 1
	}

	ctrl.SetLogger(zap.New(zap.UseDevMode(false)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: ctrlMetricsAddr,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "kubedl-election",
		Port:               9443,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	setupLog.Info("setting up scheme")
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		setupLog.Error(err, "unable to add APIs to scheme")
		os.Exit(1)
	}

	setupLog.Info("setting up gang schedulers")
	registry.RegisterGangSchedulers(mgr)

	setupLog.Info("setting up cache backends")
	cacheregistry.RegisterCacheBackends(mgr.GetClient())

	// Setup all controllers with provided manager.
	if err = controllers.SetupWithManager(mgr, options.CtrlConfig); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "KubeDL")
		os.Exit(1)
	}

	setupLog.Info("setting up storage backends")
	backendregistry.RegisterStorageBackends()

	// Setup persist controllers if storage backends are specified.
	if err = persist.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to setup persist controllers")
		os.Exit(1)
	}

	// Start monitoring for default registry.
	metrics.StartMonitoringForDefaultRegistry(metricsAddr)

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
