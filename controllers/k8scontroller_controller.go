/*
Copyright 2023.

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
	"strconv"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	apiv1alpha1 "github.com/lazyboson/k8s-controller/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// K8scontrollerReconciler reconciles a K8scontroller object
type K8scontrollerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

const (
	podDeletionCostAnnotation = "controller.kubernetes.io/pod-deletion-cost"
)

//+kubebuilder:rbac:groups=api.lazyboson.ai,resources=k8scontrollers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=api.lazyboson.ai,resources=k8scontrollers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=api.lazyboson.ai,resources=k8scontrollers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the K8scontroller object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *K8scontrollerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("k8scontroller", req.NamespacedName)

	// Create the nginx deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webserver",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(5),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "nginx",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "nginx",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx",
						},
					},
				},
			},
		},
	}

	// Check if the deployment already exists
	var existingDeployment appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, &existingDeployment)
	if err != nil && errors.IsNotFound(err) {
		// Deployment does not exist, create it
		if err := r.Create(ctx, deployment); err != nil {
			return ctrl.Result{}, err
		}
	} else if err != nil {
		// Error getting the deployment
		return ctrl.Result{}, err
	} else {
		// Deployment exists, update it
		existingDeployment.Spec = deployment.Spec
		if err := r.Update(ctx, &existingDeployment); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Get the pods for the deployment
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace("default"), client.MatchingLabels(deployment.Spec.Selector.MatchLabels)); err != nil {
		return reconcile.Result{}, err
	}

	time.Sleep(30 * time.Second)
	val := int64(-1000)
	inc := int64(1)
	for _, pod := range podList.Items {
		deletionCost := val + inc
		pod.Annotations = map[string]string{
			podDeletionCostAnnotation: strconv.FormatInt(deletionCost, 10),
		}
		if err := r.Client.Update(ctx, &pod); err != nil {
			log.Error(err, "unable to update Pod.")
			return ctrl.Result{}, err
		}
		log.Info(fmt.Sprintf("Pod: %s, pod Deletion Cost: %d", pod.Name, deletionCost))

		inc += 10
	}

	time.Sleep(60 * time.Second)

	deployment.Spec.Replicas = pointer.Int32(2)
	err = r.Client.Update(context.Background(), deployment)
	if err != nil {
		return ctrl.Result{}, err
	}

	time.Sleep(60 * time.Second)

	return ctrl.Result{RequeueAfter: time.Duration(180 * time.Second)}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8scontrollerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.K8scontroller{}).
		Complete(r)
}

// getPodDeletionCost retrieves the deletion cost of a pod
func getPodDeletionCost(pod corev1.Pod) int {
	if val, ok := pod.Annotations[podDeletionCostAnnotation]; ok {
		if cost, err := strconv.Atoi(val); err == nil {
			return cost
		}
	}
	return 0
}
