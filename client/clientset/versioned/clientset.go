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
// Code generated by client-gen. DO NOT EDIT.

package versioned

import (
	kubeflowv1 "github.com/alibaba/kubedl/client/clientset/versioned/typed/pytorch/v1"
	kubeflowv1 "github.com/alibaba/kubedl/client/clientset/versioned/typed/tensorflow/v1"
	xdlv1alpha1 "github.com/alibaba/kubedl/client/clientset/versioned/typed/xdl/v1alpha1"
	xgboostjobv1alpha1 "github.com/alibaba/kubedl/client/clientset/versioned/typed/xgboost/v1alpha1"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

type Interface interface {
	Discovery() discovery.DiscoveryInterface
	KubeflowV1() kubeflowv1.KubeflowV1Interface
	KubeflowV1() kubeflowv1.KubeflowV1Interface
	XdlV1alpha1() xdlv1alpha1.XdlV1alpha1Interface
	XgboostjobV1alpha1() xgboostjobv1alpha1.XgboostjobV1alpha1Interface
}

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	kubeflowV1         *kubeflowv1.KubeflowV1Client
	kubeflowV1         *kubeflowv1.KubeflowV1Client
	xdlV1alpha1        *xdlv1alpha1.XdlV1alpha1Client
	xgboostjobV1alpha1 *xgboostjobv1alpha1.XgboostjobV1alpha1Client
}

// KubeflowV1 retrieves the KubeflowV1Client
func (c *Clientset) KubeflowV1() kubeflowv1.KubeflowV1Interface {
	return c.kubeflowV1
}

// KubeflowV1 retrieves the KubeflowV1Client
func (c *Clientset) KubeflowV1() kubeflowv1.KubeflowV1Interface {
	return c.kubeflowV1
}

// XdlV1alpha1 retrieves the XdlV1alpha1Client
func (c *Clientset) XdlV1alpha1() xdlv1alpha1.XdlV1alpha1Interface {
	return c.xdlV1alpha1
}

// XgboostjobV1alpha1 retrieves the XgboostjobV1alpha1Client
func (c *Clientset) XgboostjobV1alpha1() xgboostjobv1alpha1.XgboostjobV1alpha1Interface {
	return c.xgboostjobV1alpha1
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	cs.kubeflowV1, err = kubeflowv1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.kubeflowV1, err = kubeflowv1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.xdlV1alpha1, err = xdlv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	cs.xgboostjobV1alpha1, err = xgboostjobv1alpha1.NewForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}

// NewForConfigOrDie creates a new Clientset for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *Clientset {
	var cs Clientset
	cs.kubeflowV1 = kubeflowv1.NewForConfigOrDie(c)
	cs.kubeflowV1 = kubeflowv1.NewForConfigOrDie(c)
	cs.xdlV1alpha1 = xdlv1alpha1.NewForConfigOrDie(c)
	cs.xgboostjobV1alpha1 = xgboostjobv1alpha1.NewForConfigOrDie(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClientForConfigOrDie(c)
	return &cs
}

// New creates a new Clientset for the given RESTClient.
func New(c rest.Interface) *Clientset {
	var cs Clientset
	cs.kubeflowV1 = kubeflowv1.New(c)
	cs.kubeflowV1 = kubeflowv1.New(c)
	cs.xdlV1alpha1 = xdlv1alpha1.New(c)
	cs.xgboostjobV1alpha1 = xgboostjobv1alpha1.New(c)

	cs.DiscoveryClient = discovery.NewDiscoveryClient(c)
	return &cs
}
