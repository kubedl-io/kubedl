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

package apiserver

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"time"

	controllerruntime "sigs.k8s.io/controller-runtime"

	clientmgr "github.com/alibaba/kubedl/console/backend/pkg/client"

	corev1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/alibaba/kubedl/pkg/storage/backends"
	"github.com/alibaba/kubedl/pkg/storage/dmo"
	"github.com/alibaba/kubedl/pkg/storage/dmo/converters"
)

func NewAPIServerEventBackend() backends.EventStorageBackend {
	return &apiServerEventBackend{ctrlClient: clientmgr.GetCtrlClient()}
}

var _ backends.EventStorageBackend = &apiServerEventBackend{}

type apiServerEventBackend struct {
	ctrlClient client.Client
	kubeClient clientset.Interface
}

func (a *apiServerEventBackend) Initialize() error {
	if a.kubeClient != nil {
		return nil
	}
	cfg := controllerruntime.GetConfigOrDie()
	a.kubeClient = clientset.NewForConfigOrDie(cfg)
	return nil
}

func (a *apiServerEventBackend) Close() error {
	return nil
}

func (a *apiServerEventBackend) Name() string {
	return "apiserver"
}

func (a *apiServerEventBackend) SaveEvent(event *corev1.Event, region string) error {
	return nil
}

func (a *apiServerEventBackend) ListEvent(jobNamespace, jobName, uid string, from, to time.Time) ([]*dmo.Event, error) {
	events := &corev1.EventList{}
	if err := a.ctrlClient.List(context.TODO(), events, &client.ListOptions{Namespace: jobNamespace}); err != nil {
		return nil, err
	}
	var ret []*dmo.Event
	sort.SliceStable(events.Items, func(i, j int) bool {
		return events.Items[i].ResourceVersion < events.Items[j].ResourceVersion
	})
	for _, ev := range events.Items {
		if ev.InvolvedObject.Name != jobName {
			continue
		}
		if len(uid) > 0 && string(ev.InvolvedObject.UID) != uid {
			continue
		}
		if ev.LastTimestamp.Time.Before(from) || ev.FirstTimestamp.Time.After(to) {
			continue
		}
		dmoEvents, _ := converters.ConvertEventToDMOEvent(ev, "")
		ret = append(ret, dmoEvents)
	}
	return ret, nil
}

func (a *apiServerEventBackend) ListLogs(namespace, name, uid string, maxLine int64, from, to time.Time) ([]string, error) {
	var tail *int64
	if maxLine > 0 {
		tail = &maxLine
	}
	req := a.kubeClient.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{TailLines: tail})
	podLogs, err := req.Stream(context.Background())
	if err != nil {
		klog.Errorf("list %v/%v logs error: %v", namespace, name, err)
		return []string{}, nil
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return nil, err
	}
	return strings.Split(buf.String(), "\n"), nil
}
