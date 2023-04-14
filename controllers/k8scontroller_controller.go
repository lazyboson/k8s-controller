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
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"

	apiv1alpha1 "github.com/lazyboson/k8s-controller/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// K8scontrollerReconciler reconciles a K8scontroller object
type K8scontrollerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

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
	op := &apiv1alpha1.K8scontroller{}
	if err := r.Get(ctx, req.NamespacedName, op); err != nil {
		return ctrl.Result{}, err
	}

	currentHour := time.Now().UTC().Hour()
	fmt.Println("current UTC time", currentHour)
	if currentHour >= op.Spec.Start && currentHour <= op.Spec.End {
		fmt.Println("entered in scaling")
		for _, deployment := range op.Spec.Deployments {
			deploy := &v1.Deployment{}
			err := r.Get(ctx, types.NamespacedName{
				Namespace: deployment.Namespace,
				Name:      deployment.Name,
			}, deploy)
			if err != nil {
				return ctrl.Result{}, err
			}

			if deploy.Spec.Replicas != &op.Spec.Replicas {
				deploy.Spec.Replicas = &op.Spec.Replicas
				fmt.Printf("replicas %d", op.Spec.Replicas)
				err := r.Update(ctx, deploy)
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}

	return ctrl.Result{RequeueAfter: time.Duration(30 * time.Second)}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *K8scontrollerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.K8scontroller{}).
		Complete(r)
}
