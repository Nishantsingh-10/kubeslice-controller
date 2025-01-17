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

package v1alpha1

import (
	"context"

	"github.com/kubeslice/kubeslice-controller/util"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var sliceconfigurationlog = logf.Log.WithName("sliceconfig-resource")

type sliceConfigValidation func(ctx context.Context, sliceConfig *SliceConfig) error
type sliceConfigUpdateValidation func(ctx context.Context, sliceConfig *SliceConfig, old runtime.Object) error

var customSliceConfigCreateValidation func(ctx context.Context, sliceConfig *SliceConfig) error = nil
var customSliceConfigUpdateValidation func(ctx context.Context, sliceConfig *SliceConfig, old runtime.Object) error = nil
var customSliceConfigDeleteValidation func(ctx context.Context, sliceConfig *SliceConfig) error = nil
var sliceConfigWebhookClient client.Client

func (r *SliceConfig) SetupWebhookWithManager(mgr ctrl.Manager, validateCreate sliceConfigValidation, validateUpdate sliceConfigUpdateValidation, validateDelete sliceConfigValidation) error {
	sliceConfigWebhookClient = mgr.GetClient()
	customSliceConfigCreateValidation = validateCreate
	customSliceConfigUpdateValidation = validateUpdate
	customSliceConfigDeleteValidation = validateDelete
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-controller-kubeslice-io-v1alpha1-sliceconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=controller.kubeslice.io,resources=sliceconfigs,verbs=create;update,versions=v1alpha1,name=msliceconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &SliceConfig{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *SliceConfig) Default() {
	sliceconfigurationlog.Info("default", "name", r.Name)
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-controller-kubeslice-io-v1alpha1-sliceconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=controller.kubeslice.io,resources=sliceconfigs,verbs=create;update;delete,versions=v1alpha1,name=vsliceconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &SliceConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *SliceConfig) ValidateCreate() error {
	sliceconfigurationlog.Info("validate create", "name", r.Name)
	sliceConfigCtx := util.PrepareKubeSliceControllersRequestContext(context.Background(), sliceConfigWebhookClient, nil, "SliceConfigValidation")
	return customSliceConfigCreateValidation(sliceConfigCtx, r)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *SliceConfig) ValidateUpdate(old runtime.Object) error {
	sliceconfigurationlog.Info("validate update", "name", r.Name)
	sliceConfigCtx := util.PrepareKubeSliceControllersRequestContext(context.Background(), sliceConfigWebhookClient, nil, "SliceConfigValidation")

	return customSliceConfigUpdateValidation(sliceConfigCtx, r, old)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *SliceConfig) ValidateDelete() error {
	sliceconfigurationlog.Info("validate delete", "name", r.Name)
	sliceConfigCtx := util.PrepareKubeSliceControllersRequestContext(context.Background(), sliceConfigWebhookClient, nil, "SliceConfigValidation")

	return customSliceConfigDeleteValidation(sliceConfigCtx, r)
}
