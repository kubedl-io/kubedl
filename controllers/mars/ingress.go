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

package controllers

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/alibaba/kubedl/api/marsjob/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	v1 "k8s.io/api/core/v1"
	networkingv1beta "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *MarsJobReconciler) reconcileIngressForWebservice(marsJob *v1alpha1.MarsJob) error {
	// Reconcile ingresses only when external host set.
	if marsJob.Spec.WebHost == nil {
		return nil
	}

	labels := r.ctrl.GenLabels(marsJob.GetName())
	labels[apiv1.ReplicaTypeLabel] = strings.ToLower(string(v1alpha1.MarsReplicaTypeWebService))
	selector, _ := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{MatchLabels: labels})

	// List active services for mars webservice instances.
	svcs := v1.ServiceList{}
	err := r.Client.List(context.Background(), &svcs, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     marsJob.Namespace,
	})
	if err != nil {
		return err
	}

	// List active ingresses already exist with same label-selector, number of ingresses and
	// services should be 1:1.
	ingresses := networkingv1beta.IngressList{}
	if err = r.Client.List(context.Background(), &ingresses, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     marsJob.Namespace,
	}); err != nil {
		return err
	}
	// Construct a ingress name set for fast lookup.
	ingressSet := sets.NewString()
	for i := range ingresses.Items {
		ingressSet.Insert(ingresses.Items[i].Name)
	}

	for i := range svcs.Items {
		svc := &svcs.Items[i]
		if ingressSet.Has(svc.Name) {
			continue
		}
		// Create new ingress instance and expose web service to external users.
		if err = r.createNewIngressForWebservice(svc, *marsJob.Spec.WebHost, labels); err != nil {
			return err
		}
	}
	return nil
}

func (r *MarsJobReconciler) createNewIngressForWebservice(svc *v1.Service, host string, labels map[string]string) error {
	port := r.findDefaultPort(svc)

	// Generate ingress instance by given service template, every ingress exposes a URL formatted as:
	// http://{host}/mars/{namespace}/{svc.Name}/
	// all traffic requests for this url will be forwarded to the backend service and its endpoint.
	ingress := networkingv1beta.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/$1",
				"nginx.org/websocket-services":               fmt.Sprintf(`"%s"`, svc.Name),
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         svc.APIVersion,
					Kind:               svc.Kind,
					Name:               svc.Name,
					UID:                svc.UID,
					Controller:         pointer.BoolPtr(true),
					BlockOwnerDeletion: pointer.BoolPtr(true),
				},
			},
		},
		Spec: networkingv1beta.IngressSpec{
			Rules: []networkingv1beta.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1beta.IngressRuleValue{
						HTTP: &networkingv1beta.HTTPIngressRuleValue{
							Paths: []networkingv1beta.HTTPIngressPath{
								{
									Path: path.Join("/", "mars", svc.Namespace, svc.Name+"/(.*)"),
									Backend: networkingv1beta.IngressBackend{
										ServiceName: svc.Name,
										ServicePort: intstr.IntOrString{IntVal: port},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return r.Client.Create(context.Background(), &ingress)
}

func (r *MarsJobReconciler) findDefaultPort(svc *v1.Service) int32 {
	if len(svc.Spec.Ports) == 0 {
		return v1alpha1.DefaultPort
	}
	port := svc.Spec.Ports[0].Port
	for i := 1; i < len(svc.Spec.Ports); i++ {
		if svc.Spec.Ports[i].Name == r.ctrl.Controller.GetDefaultContainerPortName() {
			port = svc.Spec.Ports[i].Port
			break
		}
	}
	return port
}
