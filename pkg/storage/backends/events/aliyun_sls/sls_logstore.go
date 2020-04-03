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

package aliyun_sls

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alibaba/kubedl/pkg/storage/backends"
	"github.com/alibaba/kubedl/pkg/storage/dmo"
	"github.com/alibaba/kubedl/pkg/util"
	sls "github.com/aliyun/aliyun-log-go-sdk"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
)

const (
	defaultSLSRetryTimes           = 10
	quotaExceedErrorHoldupDuration = 800 * time.Millisecond
	serverErrorHoldupDuration      = 200 * time.Millisecond

	slsMaxLineNum int64 = 100
)

func NewSLSEventBackend() backends.EventStorageBackend {
	return &slsEventBackend{initialized: 0}
}

var _ backends.EventStorageBackend = &slsEventBackend{}

type slsEventBackend struct {
	slsClient   *sls.Client
	projectName string
	logStore    string
	initialized int32
}

func (s *slsEventBackend) Initialize() error {
	if atomic.LoadInt32(&s.initialized) == 1 {
		return nil
	}
	if err := s.init(); err != nil {
		return err
	}
	atomic.StoreInt32(&s.initialized, 1)
	return nil
}

func (s *slsEventBackend) Close() error {
	if s.slsClient == nil {
		return nil
	}
	return s.slsClient.Close()
}

func (s *slsEventBackend) Name() string {
	return "aliyun-sls"
}

func (s *slsEventBackend) SaveEvent(event *corev1.Event, region string) error {
	lg := s.wrapSLSLog(event, region)
	var err error
	for retryTimes := 0; retryTimes < defaultSLSRetryTimes; retryTimes++ {
		err = s.slsClient.PutLogs(s.projectName, s.logStore, lg)
		if err == nil {
			return nil
		}
		// Handle sls quota-exceed exceptions here, hold for a while and
		// retry.
		if strings.Contains(err.Error(), sls.WRITE_QUOTA_EXCEED) ||
			strings.Contains(err.Error(), sls.PROJECT_QUOTA_EXCEED) ||
			strings.Contains(err.Error(), sls.SHARD_WRITE_QUOTA_EXCEED) {
			time.Sleep(quotaExceedErrorHoldupDuration)
			continue
		}
		// Handle sls server exceptions here, hold for a while and retry.
		if strings.Contains(err.Error(), sls.INTERNAL_SERVER_ERROR) ||
			strings.Contains(err.Error(), sls.SERVER_BUSY) {
			time.Sleep(serverErrorHoldupDuration)
		}
	}
	klog.Errorf("SLS PutLogs failed after retry 10 times, err: %v\n", err)
	return err
}
func (s *slsEventBackend) ListEvent(jobNamespace, jobName string, from, to time.Time) ([]*dmo.Event, error) {
	query := fmt.Sprintf("%s AND %s", jobNamespace, jobName)
	// Get histogram statistical information of logs satisfied with query.
	hist, err := s.slsClient.GetHistograms(s.projectName, s.logStore, "", from.Unix(), to.Unix(), query)
	if err != nil {
		return nil, err
	}

	var (
		logsCnt            = hist.Count
		offset       int64 = 0
		eventsBuffer       = make([]*dmo.Event, logsCnt)
	)

	// SLS allows client gets maximum 100 segments of logs onetime, so we'd try to pull slsMaxLineNum log
	// once a time and finally aggregate them into logs buffer.
	for logsCnt > 0 {
		getCnt := slsMaxLineNum
		if logsCnt < slsMaxLineNum {
			getCnt = logsCnt
		}
		logResp, err := s.slsClient.GetLogs(s.projectName, s.logStore, "", from.Unix(), to.Unix(), query, getCnt, offset, false)
		if err != nil {
			return nil, err
		}
		events, err := s.unwrapSLSLogs(logResp)
		if err != nil {
			return nil, err
		}

		// Copy fetched logs and update progress flags.
		copy(eventsBuffer[offset:offset+logResp.Count], events)
		logsCnt -= slsMaxLineNum
		offset += slsMaxLineNum
	}

	return eventsBuffer, nil
}

func (s *slsEventBackend) wrapSLSLog(event *corev1.Event, region string) *sls.LogGroup {
	content := make([]*sls.LogContent, 0, 9)
	content = append(content, &sls.LogContent{
		Key:   pointer.StringPtr("FirstTimestamp"),
		Value: pointer.StringPtr(event.FirstTimestamp.Format(time.RFC3339)),
	})
	content = append(content, &sls.LogContent{
		Key:   pointer.StringPtr("LastTimestamp"),
		Value: pointer.StringPtr(event.LastTimestamp.Format(time.RFC3339)),
	})
	content = append(content, &sls.LogContent{
		Key:   pointer.StringPtr("Count"),
		Value: pointer.StringPtr(strconv.Itoa(int(event.Count))),
	})
	content = append(content, &sls.LogContent{
		Key:   pointer.StringPtr("Name"),
		Value: pointer.StringPtr(event.Name),
	})
	content = append(content, &sls.LogContent{
		Key:   pointer.StringPtr("Kind"),
		Value: pointer.StringPtr(event.InvolvedObject.Kind),
	})
	content = append(content, &sls.LogContent{
		Key:   pointer.StringPtr("ObjUID"),
		Value: pointer.StringPtr(string(event.InvolvedObject.UID)),
	})
	content = append(content, &sls.LogContent{
		Key:   pointer.StringPtr("ObjNamespace"),
		Value: pointer.StringPtr(event.InvolvedObject.Namespace),
	})
	content = append(content, &sls.LogContent{
		Key:   pointer.StringPtr("ObjName"),
		Value: pointer.StringPtr(event.InvolvedObject.Name),
	})
	content = append(content, &sls.LogContent{
		Key:   pointer.StringPtr("Type"),
		Value: pointer.StringPtr(event.Type),
	})
	content = append(content, &sls.LogContent{
		Key:   pointer.StringPtr("Reason"),
		Value: pointer.StringPtr(event.Reason),
	})
	content = append(content, &sls.LogContent{
		Key:   pointer.StringPtr("Message"),
		Value: pointer.StringPtr(event.Message),
	})
	if region != "" {
		content = append(content, &sls.LogContent{
			Key:   pointer.StringPtr("Region"),
			Value: pointer.StringPtr(region),
		})
	}
	return &sls.LogGroup{
		Topic:  pointer.StringPtr(""),
		Source: pointer.StringPtr(event.Source.Component + "/" + event.Source.Host),
		Logs: []*sls.Log{
			{
				Time:     util.UInt32Ptr(uint32(event.LastTimestamp.Time.Unix())),
				Contents: content,
			},
		},
	}
}

func (s *slsEventBackend) unwrapSLSLogs(logRsp *sls.GetLogsResponse) ([]*dmo.Event, error) {
	events := make([]*dmo.Event, 0, logRsp.Count)
	// Sort logs by first timestamp.
	sort.Slice(logRsp.Logs, func(i, j int) bool {
		ti, ok := logRsp.Logs[i]["FirstTimestamp"]
		if !ok {
			return false
		}
		tj, ok := logRsp.Logs[j]["FirstTimestamp"]
		if !ok {
			return false
		}
		return strings.Compare(ti, tj) < 0
	})

	// Deduplicate logs by involved object uid.
	dup := make(map[string]bool)
	for _, item := range logRsp.Logs {
		e, err := unwrapSLSLog(item)
		if err != nil {
			return nil, err
		}
		if !dup[e.ObjUID] {
			events = append(events, e)
			dup[e.ObjUID] = true
		}
	}
	return events, nil
}

func unwrapSLSLog(rawLog map[string]string) (*dmo.Event, error) {
	e := &dmo.Event{
		Name:         rawLog["Name"],
		Kind:         rawLog["Kind"],
		Type:         rawLog["Type"],
		ObjNamespace: rawLog["ObjNamespace"],
		ObjName:      rawLog["ObjName"],
		ObjUID:       rawLog["ObjUID"],
		Reason:       rawLog["Reason"],
		Message:      rawLog["Message"],
	}
	count, err := strconv.Atoi(rawLog["Count"])
	if err != nil {
		return nil, err
	}
	e.Count = int32(count)
	if region, ok := rawLog["Region"]; ok {
		e.Region = &region
	}
	first, err := time.Parse(time.RFC3339, rawLog["FirstTimestamp"])
	if err != nil {
		return nil, err
	}
	e.FirstTimestamp = first
	last, err := time.Parse(time.RFC3339, rawLog["LastTimestamp"])
	if err != nil {
		return nil, err
	}
	e.LastTimestamp = last
	return e, nil
}

func (s *slsEventBackend) init() error {
	slsClient, project, logStore, err := GetSLSClient()
	if err != nil {
		return err
	}
	s.slsClient = slsClient
	s.projectName = project
	s.logStore = logStore
	return nil
}
