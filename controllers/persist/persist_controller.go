/*
Copyright 2020 The Alibaba Authors.

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

package persist

import (
	"flag"
	"os"

	"github.com/alibaba/kubedl/controllers/persist/event"
	"github.com/alibaba/kubedl/controllers/persist/object/job"
	"github.com/alibaba/kubedl/controllers/persist/object/pod"

	ctrl "sigs.k8s.io/controller-runtime"
)

func init() {
	flag.StringVar(&region, "region", "", "region of kubedl deployed")
	flag.StringVar(&eventStorage, "event-storage", "", "event storage backend plugin name, persist events into backend if it's specified")
	flag.StringVar(&objectStorage, "object-storage", "", "object storage backend plugin name, persist jobs and pods into backend if it's specified")
}

var (
	region        string
	eventStorage  string
	objectStorage string
)

func SetupWithManager(mgr ctrl.Manager) error {
	if regionEnv, ok := os.LookupEnv("REGION"); ok {
		region = regionEnv
	}

	if eventStorage != "" {
		eventPersistController, err := event.NewEventPersistController(mgr, eventStorage, region)
		if err != nil {
			return err
		}
		if err = eventPersistController.SetupWithManager(mgr); err != nil {
			return err
		}
	}

	if objectStorage != "" {
		jobPersistController, err := job.NewJobPersistControllers(mgr, objectStorage, region)
		if err != nil {
			return err
		}
		if err = jobPersistController.SetupWithManager(mgr); err != nil {
			return err
		}
		podPersistController, err := pod.NewPodPersistController(mgr, objectStorage, region)
		if err != nil {
			return err
		}
		if err = podPersistController.SetupWithManager(mgr); err != nil {
			return err
		}
	}
	return nil
}
