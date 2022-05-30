/*
 * 	Copyright (c) 2022 Avesha, Inc. All rights reserved. # # SPDX-License-Identifier: Apache-2.0
 *
 * 	Licensed under the Apache License, Version 2.0 (the "License");
 * 	you may not use this file except in compliance with the License.
 * 	You may obtain a copy of the License at
 *
 * 	http://www.apache.org/licenses/LICENSE-2.0
 *
 * 	Unless required by applicable law or agreed to in writing, software
 * 	distributed under the License is distributed on an "AS IS" BASIS,
 * 	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * 	See the License for the specific language governing permissions and
 * 	limitations under the License.
 */

package controller

import (
	"context"

	"github.com/go-logr/logr"
	controllerv1alpha1 "github.com/kubeslice/apis/pkg/controller/v1alpha1"
	util "github.com/kubeslice/apis/pkg/util"
	"github.com/kubeslice/kubeslice-controller/service"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	ClusterService service.IClusterService
	Log            logr.Logger
}

// SetupWithManager sets up the controller with the Manager.
func (c *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&controllerv1alpha1.Cluster{}).
		Complete(c)
}

// Reconcile is a function to reconcile the cluster , ClusterReconciler implements it
func (c *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	kubeSliceCtx := util.PrepareKubeSliceControllersRequestContext(ctx, c.Client, c.Scheme, "ClusterController")
	return c.ClusterService.ReconcileCluster(kubeSliceCtx, req)
}
