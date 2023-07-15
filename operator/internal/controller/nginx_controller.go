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

package controller

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	webserverv1 "github.com/skraga/go-k8s-demo/operator/api/v1"
)

// NginxReconciler reconciles a Nginx object
type NginxReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=webserver.demo.local,resources=nginxes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=webserver.demo.local,resources=nginxes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

func (r *NginxReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Nginx custom resource
	nginx := &webserverv1.Nginx{}
	err := r.Get(ctx, req.NamespacedName, nginx)
	if err != nil {
		// Handle the error if the custom resource is not found or any other error occurs
		if errors.IsNotFound(err) {
			// Nginx custom resource not found, check if the associated Deployment exists and delete it
			deployment := &appsv1.Deployment{}
			err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, deployment)
			if err != nil && errors.IsNotFound(err) {
				// Deployment not found, nothing to do
				return ctrl.Result{}, nil
			} else if err != nil {
				log.Error(err, "Failed to fetch Nginx deployment")
				return ctrl.Result{}, err
			}

			// Delete the Deployment
			err = r.Delete(ctx, deployment)
			if err != nil {
				log.Error(err, "Failed to delete Nginx deployment")
				return ctrl.Result{}, err
			}

			// Deployment deleted successfully, return Requeue to check status again later
			return ctrl.Result{Requeue: true}, nil
		}

		log.Error(err, "Failed to fetch Nginx custom resource")
		return ctrl.Result{}, err
	}

	// Check if the Nginx deployment already exists, create if not
	deployment := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: nginx.Name, Namespace: nginx.Namespace}, deployment)
	if err != nil && errors.IsNotFound(err) {
		// Nginx deployment not found, create a new one
		deployment := r.newDeploymentForNginx(nginx)
		err = r.Create(ctx, deployment)
		if err != nil {
			log.Error(err, "Failed to create Nginx deployment")
			return ctrl.Result{}, err
		}

		// Deployment created successfully, return Requeue to check status again later
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to fetch Nginx deployment")
		return ctrl.Result{}, err
	}

	// Check if the number of replicas in the deployment matches the desired replicas
	if *deployment.Spec.Replicas != nginx.Spec.Replicas {
		deployment.Spec.Replicas = &nginx.Spec.Replicas
		err = r.Update(ctx, deployment)
		if err != nil {
			log.Error(err, "Failed to update Nginx deployment")
			return ctrl.Result{}, err
		}

		// Deployment updated successfully, return Requeue to check status again later
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

func (r *NginxReconciler) newDeploymentForNginx(nginx *webserverv1.Nginx) *appsv1.Deployment {
	// Create a new Deployment object with the desired Nginx configuration
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nginx.Name,
			Namespace: nginx.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &nginx.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": nginx.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": nginx.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: nginx.Spec.Image,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: nginx.Spec.Port,
								},
							},
						},
					},
				},
			},
		},
	}

	// Set the owner reference to establish the relationship between the Nginx custom resource and the Deployment
	ctrl.SetControllerReference(nginx, deployment, r.Scheme)

	return deployment
}

// SetupWithManager sets up the controller with the Manager.
func (r *NginxReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webserverv1.Nginx{}).
		Complete(r)
}
