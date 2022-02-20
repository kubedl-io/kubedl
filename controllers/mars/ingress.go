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
	"path"
	"strings"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/alibaba/kubedl/apis/training/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"

	v1 "k8s.io/api/core/v1"
	networkingv1beta "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	availableAddresses := make([]string, 0, len(svcs.Items))
	for i := range svcs.Items {
		svc := &svcs.Items[i]
		if ingressSet.Has(svc.Name) {
			continue
		}
		// Create new ingress instance and expose web service to external users.
		addr, err := r.createNewIngressForWebservice(svc, *marsJob.Spec.WebHost, labels)
		if err != nil {
			return err
		}
		availableAddresses = append(availableAddresses, addr)
	}

	marsJobCpy := marsJob.DeepCopy()
	marsJobCpy.Status.WebServiceAddresses = append(marsJob.Status.WebServiceAddresses, availableAddresses...)
	return r.Client.Status().Patch(context.Background(), marsJobCpy, client.MergeFrom(marsJob))
}

func (r *MarsJobReconciler) createNewIngressForWebservice(svc *v1.Service, host string, labels map[string]string) (string, error) {
	port := r.findDefaultPort(svc)
	regexAPIPath := path.Join("/", "mars", svc.Namespace, svc.Name+"/(.*)")
	indexPath := path.Join("/", "mars", svc.Namespace, svc.Name)
	prefixPathType := networkingv1.PathTypePrefix

	// Generate ingress instance by given service template, every ingress exposes a URL formatted as:
	// http://{host}/mars/{namespace}/{svc.Name}/
	// all traffic requests for this url will be forwarded to the backend service and its endpoint.
	ingress := networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/$1",
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
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     regexAPIPath,
									PathType: &prefixPathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: svc.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: port,
											},
										},
									},
								},
								{
									Path: indexPath,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: svc.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: port,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := r.Client.Create(context.Background(), &ingress); err != nil {
		return "", err
	}
	return path.Join(host, indexPath), nil
}

func (r *MarsJobReconciler) findDefaultPort(svc *v1.Service) int32 {
	if len(svc.Spec.Ports) == 0 {
		return v1alpha1.MarsJobDefaultPort
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
