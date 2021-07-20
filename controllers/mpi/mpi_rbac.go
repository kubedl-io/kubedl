package mpi

import (
	"context"
	"fmt"
	training "github.com/alibaba/kubedl/apis/training/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *MPIJobReconciler) getOrCreateLauncherServiceAccount(mpiJob *training.MPIJob) (*corev1.ServiceAccount, error) {
	sa := corev1.ServiceAccount{}
	err := r.Client.Get(context.Background(), types.NamespacedName{
		Namespace: mpiJob.Namespace,
		Name: mpiJob.Name + launcherSuffix,
	}, &sa)
	if err != nil && errors.IsNotFound(err) {
		// If the ServiceAccount doesn't exist, we'll create it.
		err = r.Client.Create(context.Background(), newLauncherServiceAccount(mpiJob))
	}

	if err != nil {
		return nil, err
	}
	return &sa, nil
}

// newLauncherServiceAccount creates a new launcher ServiceAccount for an MPIJob
// resource. It also sets the appropriate OwnerReferences on the resource so
// handleObject can discover the MPIJob resource that 'owns' it.
func newLauncherServiceAccount(mpiJob *training.MPIJob) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mpiJob.Name + launcherSuffix,
			Namespace: mpiJob.Namespace,
			Labels: map[string]string{
				"app": mpiJob.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(mpiJob, training.SchemeGroupVersion.WithKind(training.MPIJobKind)),
			},
		},
	}
}

func (r *MPIJobReconciler) getOrCreateLauncherRole(mpiJob *training.MPIJob, workerReplicas int32) (*rbacv1.Role, error) {
	role := rbacv1.Role{}
	err := r.Client.Get(context.Background(), types.NamespacedName{
		Namespace: mpiJob.Namespace,
		Name: mpiJob.Name + launcherSuffix,
	}, &role)
	if err != nil && errors.IsNotFound(err) {
		// If the Role doesn't exist, we'll create it.
		err = r.Client.Create(context.Background(), newLauncherRole(mpiJob, workerReplicas))
	}

	if err != nil {
		return nil, err
	}
	return &role, nil
}

// newLauncherRole creates a new launcher Role for an MPIJob resource. It also
// sets the appropriate OwnerReferences on the resource so handleObject can
// discover the MPIJob resource that 'owns' it.
func newLauncherRole(mpiJob *training.MPIJob, workerReplicas int32) *rbacv1.Role {
	var podNames []string
	for i := 0; i < int(workerReplicas); i++ {
		podNames = append(podNames, fmt.Sprintf("%s%s-%d", mpiJob.Name, workerSuffix, i))
	}
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mpiJob.Name + launcherSuffix,
			Namespace: mpiJob.Namespace,
			Labels: map[string]string{
				"app": mpiJob.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(mpiJob, training.SchemeGroupVersion.WithKind(training.MPIJobKind)),
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "list", "watch"},
				APIGroups: []string{""},
				Resources: []string{"pods"},
			},
			{
				Verbs:         []string{"create"},
				APIGroups:     []string{""},
				Resources:     []string{"pods/exec"},
				ResourceNames: podNames,
			},
		},
	}
}


func (r *MPIJobReconciler) getLauncherRoleBinding(mpiJob *training.MPIJob) (*rbacv1.RoleBinding, error) {
	rb := rbacv1.RoleBinding{}
	err := r.Client.Get(context.Background(), types.NamespacedName{
		Namespace: mpiJob.Namespace,
		Name: mpiJob.Name + launcherSuffix,
	}, &rb)
	if err != nil && errors.IsNotFound(err) {
		// If the RoleBinding doesn't exist, we'll create it.
		err = r.Client.Create(context.Background(), newLauncherRoleBinding(mpiJob))
	}

	if err != nil {
		return nil, err
	}
	return &rb, nil
}


// newLauncherRoleBinding creates a new launcher RoleBinding for an MPIJob
// resource. It also sets the appropriate OwnerReferences on the resource so
// handleObject can discover the MPIJob resource that 'owns' it.
func newLauncherRoleBinding(mpiJob *training.MPIJob) *rbacv1.RoleBinding {
	launcherName := mpiJob.Name + launcherSuffix
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      launcherName,
			Namespace: mpiJob.Namespace,
			Labels: map[string]string{
				"app": mpiJob.Name,
			},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(mpiJob, training.SchemeGroupVersion.WithKind(training.MPIJobKind)),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      launcherName,
				Namespace: mpiJob.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     launcherName,
		},
	}
}