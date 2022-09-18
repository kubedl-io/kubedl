package torchelastic

import (
	"bufio"
	"context"
	"fmt"
	logger "github.com/sirupsen/logrus"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"

	"regexp"
	"strconv"
	"strings"
)

// Construct a request for getting the logs for a pod and retrieves the logs.
func readRawLogs(client kubernetes.Interface, namespace, podID string, logOptions *v1.PodLogOptions) (string, error) {
	readCloser, err := openStream(client, namespace, podID, logOptions)
	if err != nil {
		return err.Error(), nil
	}

	defer func(readCloser io.ReadCloser) {
		err := readCloser.Close()
		if err != nil {
			return
		}
	}(readCloser)

	reader := bufio.NewReader(readCloser)
	line, err := reader.ReadString('\n')
	if err != nil {

		return err.Error(), nil
	}

	return line, nil
}

func openStream(client kubernetes.Interface, namespace, podID string, logOptions *v1.PodLogOptions) (io.ReadCloser, error) {
	return client.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Name(podID).
		Resource("pods").
		SubResource("log").
		VersionedParams(logOptions, scheme.ParameterCodec).Stream(context.TODO())
}

func podRunning(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodRunning
}

func GetDefaultWorkerName(pytorchJobName string) string {
	return pytorchJobName + "-" + "worker" + "-" + "0"
}

func read(client *kubernetes.Clientset, namespace, name string) (MetricObservation, error) {
	lines := int64(1)
	opts := &v1.PodLogOptions{
		TailLines: &lines,
		Follow:    true,
	}

	//Read raw pod log.
	detail, err := readRawLogs(client, namespace, name, opts)
	if err != nil {
		return MetricObservation{}, err
	}
	//Extract training metrics from raw log.
	rawLog := strings.Split(detail, "\t")
	epochRule := regexp.MustCompile(`[0-9]{1,2}`)
	batchRule := regexp.MustCompile(`[0-9]{2,4}`)
	trainRule := regexp.MustCompile(`[0-9]{1,2}.[0-9]{3}`)
	accRule := regexp.MustCompile(`[0-9]{1,2}.[0-9]{1,2}`)
	matchTrain, err := regexp.MatchString(`Epoch`, rawLog[0])

	if err != nil {
		return MetricObservation{}, err
	}
	// If current log is a training log.
	if matchTrain {
		epochNum, _ := strconv.Atoi(epochRule.FindStringSubmatch(rawLog[0])[0])
		batchNum, _ := strconv.Atoi(batchRule.FindStringSubmatch(rawLog[0])[0])
		trainTime, _ := strconv.ParseFloat(trainRule.FindStringSubmatch(rawLog[1])[0], 64)
		accuracy, _ := strconv.ParseFloat(accRule.FindStringSubmatch(rawLog[5])[0], 64)

		observation := MetricObservation{
			Accuracy: accuracy,
			Epoch:    int32(epochNum),
			Latency:  trainTime,
			Batch:    int32(batchNum),
		}
		// drop the inaccurate train data
		if trainTime > 1 {
			return MetricObservation{}, fmt.Errorf("drop the inaccurate train data")
		}

		logger.Infof("epoch: %d batch: %d train_time: %f accuracy: %f", epochNum, batchNum, trainTime, accuracy)
		return observation, nil
	}

	return MetricObservation{}, fmt.Errorf("current log is not a training log")
}
