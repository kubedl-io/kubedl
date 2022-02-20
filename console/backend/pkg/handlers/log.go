package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/alibaba/kubedl/console/backend/pkg/storage"

	"github.com/alibaba/kubedl/console/backend/pkg/model"
	"github.com/alibaba/kubedl/pkg/storage/backends"
)

func NewLogHandler(eventStorage string) (*LogHandler, error) {
	eventBackend := storage.GetEventBackend(eventStorage)
	if eventBackend == nil {
		return nil, fmt.Errorf("no event backend storage named: %s", eventStorage)
	}
	err := eventBackend.Initialize()
	if err != nil {
		return nil, err
	}
	return &LogHandler{eventBackend: eventBackend}, nil
}

type LogHandler struct {
	eventBackend backends.EventStorageBackend
}

func (lh *LogHandler) GetLogs(namespace, podName, uid string, from, to time.Time) ([]string, error) {
	logs, err := lh.eventBackend.ListLogs(namespace, podName, uid, 2000, from, to)
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (lh *LogHandler) DownloadLogs(namespace, podName, uid string, from, to time.Time) ([]byte, error) {
	logs, err := lh.eventBackend.ListLogs(namespace, podName, uid, -1, from, to)
	if err != nil {
		return nil, err
	}
	return []byte(strings.Join(logs, "\r\n")), nil
}

func (lh *LogHandler) GetEvents(namespace, objName, uid string, from, to time.Time) ([]string, error) {
	list, err := lh.eventBackend.ListEvent(namespace, objName, uid, from, to)
	if err != nil {
		return nil, err
	}
	if len(list) > 2000 {
		list = list[:2000]
	}
	msg := []string{}
	for _, ev := range list {
		msg = append(msg, fmt.Sprintf("%s %s", ev.LastTimestamp.Format(model.JobInfoTimeFormat), ev.Message))
	}
	return msg, nil
}
