package job_controller

import (
	"strconv"
	"strings"
	"testing"
	"context"

	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	testjobv1 "github.com/alibaba/kubedl/pkg/test_job/v1"
	testutilv1 "github.com/alibaba/kubedl/pkg/test_util/v1"
	"k8s.io/api/core/v1"
)

func TestSetRestartPolicy(t *testing.T) {
	type tc struct {
		testJob               *testjobv1.TestJob
		expectedRestartPolicy v1.RestartPolicy
		expectedType          apiv1.ReplicaType
	}
	testCase := []tc{
		func() tc {
			tj := testutilv1.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = apiv1.RestartPolicyExitCode
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyNever,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
		func() tc {
			tj := testutilv1.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = apiv1.RestartPolicyNever
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyNever,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
		func() tc {
			tj := testutilv1.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = apiv1.RestartPolicyAlways
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyAlways,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
		func() tc {
			tj := testutilv1.NewTestJob(2)
			tj.Spec.TestReplicaSpecs[testjobv1.TestReplicaTypeWorker].RestartPolicy = apiv1.RestartPolicyOnFailure
			return tc{
				testJob:               tj,
				expectedRestartPolicy: v1.RestartPolicyOnFailure,
				expectedType:          testjobv1.TestReplicaTypeWorker,
			}
		}(),
	}
	for _, c := range testCase {
		spec := c.testJob.Spec.TestReplicaSpecs[c.expectedType]
		podTemplate := spec.Template
		setRestartPolicy(&podTemplate, spec)
		if podTemplate.Spec.RestartPolicy != c.expectedRestartPolicy {
			t.Errorf("Expected %s, got %s", c.expectedRestartPolicy, podTemplate.Spec.RestartPolicy)
		}
	}
}

func TestCreateNewPod(t *testing.T) {
	type tc struct {

	}
	testCase := []tc{

	}

	jc := JobController{

	}

	ctx := context.WithValue(context.Background(), contextHostNetworkPorts, make(map[string]int32))

	rt := strings.ToLower(string(rtype))

	for _, c := range(testCase) {

		jc.createNewPod(ctx, job, rt, strconv.Itoa(index), spec, masterRole, replicas)
	}
}