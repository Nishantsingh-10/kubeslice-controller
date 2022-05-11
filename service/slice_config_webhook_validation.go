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

package service

import (
	"context"
	"fmt"
	"strings"

	controllerv1alpha1 "github.com/kubeslice/kubeslice-controller/apis/controller/v1alpha1"
	workerv1alpha1 "github.com/kubeslice/kubeslice-controller/apis/worker/v1alpha1"
	"github.com/kubeslice/kubeslice-controller/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// s is instance of SliceConfig schema
var s *controllerv1alpha1.SliceConfig = nil

// sliceConfigCtx is context var
var sliceConfigCtx context.Context = nil

// ValidateSliceConfigCreate is a function to verify the creation of slice config
func ValidateSliceConfigCreate(ctx context.Context, sliceConfig *controllerv1alpha1.SliceConfig) error {
	s = sliceConfig
	sliceConfigCtx = ctx
	var allErrs field.ErrorList
	if err := validateProjectNamespace(); err != nil {
		allErrs = append(allErrs, err)
	} else {
		if err = validateSliceSubnet(); err != nil {
			allErrs = append(allErrs, err)
		}
		if err = validateClusters(); err != nil {
			allErrs = append(allErrs, err)
		}
		if err = validateQosProfile(); err != nil {
			allErrs = append(allErrs, err)
		}
		if err = validateExternalGatewayConfig(); err != nil {
			allErrs = append(allErrs, err)
		}
		if err = validateNamespaces(); err != nil {
			allErrs = append(allErrs, err)
		}
	}
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(schema.GroupKind{Group: "controller.kubeslice.io", Kind: "SliceConfig"}, s.Name, allErrs)
}

// ValidateSliceConfigUpdate is function to verify the update of slice config
func ValidateSliceConfigUpdate(ctx context.Context, sliceConfig *controllerv1alpha1.SliceConfig) error {
	s = sliceConfig
	sliceConfigCtx = ctx
	var allErrs field.ErrorList
	if err := preventUpdate(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateClusters(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateQosProfile(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateExternalGatewayConfig(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateNamespaces(); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(schema.GroupKind{Group: "controller.kubeslice.io", Kind: "SliceConfig"}, s.Name, allErrs)
}

// ValidateSliceConfigDelete is function to validate the deletion of sliceConfig
func ValidateSliceConfigDelete(ctx context.Context, sliceConfig *controllerv1alpha1.SliceConfig) error {
	s = sliceConfig
	workerSliceConfigCtx = ctx

	var allErrs field.ErrorList
	if err := preventDeleteSliceConfig(); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(schema.GroupKind{Group: "controller.kubeslice.io", Kind: "SliceConfig"}, s.Name, allErrs)
}

func preventDeleteSliceConfig() *field.Error {
	workerSlices := &workerv1alpha1.WorkerSliceConfigList{}
	ownerLabel := map[string]string{
		"original-slice-name": s.Name,
	}
	err := util.ListResources(workerSliceConfigCtx, workerSlices, client.MatchingLabels(ownerLabel), client.InNamespace(s.Namespace))
	if err == nil && len(workerSlices.Items) > 0 {
		for _, slice := range workerSlices.Items {
			if len(slice.Spec.NamespaceIsolationProfile.ApplicationNamespaces) > 0 && len(slice.Spec.NamespaceIsolationProfile.AllowedNamespaces) > 0 {
				applicationNamespacesErr := &field.Error{
					Type:     field.ErrorTypeInternal,
					Field:    "Field: ApplicationNamespaces & AllowedNamespaces",
					BadValue: fmt.Sprintf("Number of ApplicationNamespaces: %d and AllowedNamespaces: %d", len(slice.Spec.NamespaceIsolationProfile.ApplicationNamespaces), len(slice.Spec.NamespaceIsolationProfile.AllowedNamespaces)),
					Detail:   fmt.Sprintf("%s", "Please deboard the namespace before deletion."),
				}
				return applicationNamespacesErr
			} else {
				if len(slice.Status.OnboardedNamespaces) > 0 {
					onboardNamespaceErr := &field.Error{
						Type:     field.ErrorTypeInternal,
						Field:    "Field: OnboardedNamespaces",
						BadValue: fmt.Sprintf("Number of onboarded namespaces: %d", len(slice.Status.OnboardedNamespaces)),
						Detail:   fmt.Sprintf("%s", "Deboarding of namespaces is in progress try after some time."),
					}
					return onboardNamespaceErr
				}
			}

		}
	}
	return nil
}

// validateSliceSubnet is function to validate the the subnet of slice
func validateSliceSubnet() *field.Error {
	if !util.IsPrivateSubnet(s.Spec.SliceSubnet) {
		return field.Invalid(field.NewPath("Spec").Child("sliceSubnet"), s.Spec.SliceSubnet, "must be a private subnet")
	}
	if !util.HasPrefix(s.Spec.SliceSubnet, "16") {
		return field.Invalid(field.NewPath("Spec").Child("sliceSubnet"), s.Spec.SliceSubnet, "prefix must be 16")
	}
	if !util.HasLastTwoOctetsZero(s.Spec.SliceSubnet) {
		return field.Invalid(field.NewPath("Spec").Child("sliceSubnet"), s.Spec.SliceSubnet, "third and fourth octets must be 0")
	}
	return nil
}

// validateProjectNamespace is a function to verify the namespace of project
func validateProjectNamespace() *field.Error {
	namespace := &corev1.Namespace{}
	exist, _ := util.GetResourceIfExist(sliceConfigCtx, client.ObjectKey{Name: s.Namespace}, namespace)
	if !exist || !checkForProjectNamespace(namespace) {
		return field.Invalid(field.NewPath("metadata").Child("namespace"), s.Namespace, "SliceConfig must be applied on project namespace")
	}
	return nil
}

// checkForProjectNamespace is a function to check namespace is in decided format
func checkForProjectNamespace(namespace *corev1.Namespace) bool {
	return namespace.Labels[util.LabelName] == fmt.Sprintf(util.LabelValue, "Project", namespace.Name)
}

// validateClusters is function to validate the cluster specification
func validateClusters() *field.Error {
	if duplicate, value := util.CheckDuplicateInArray(s.Spec.Clusters); duplicate {
		return field.Invalid(field.NewPath("Spec").Child("Clusters"), value, "clusters must be unique in slice config")
	}
	for _, clusterName := range s.Spec.Clusters {
		cluster := controllerv1alpha1.Cluster{}
		exist, _ := util.GetResourceIfExist(sliceConfigCtx, client.ObjectKey{Name: clusterName, Namespace: s.Namespace}, &cluster)
		if !exist {
			return field.Invalid(field.NewPath("Spec").Child("Clusters"), clusterName, "cluster is not registered")
		}
		if cluster.Spec.NetworkInterface == "" {
			return field.Required(field.NewPath("Spec").Child("Clusters").Child("NetworkInterface"), "for cluster "+clusterName)
		}
		if cluster.Spec.NodeIP == "" {
			return field.Required(field.NewPath("Spec").Child("Clusters").Child("NodeIP"), "for cluster "+clusterName)
		}
		if len(cluster.Status.CniSubnet) == 0 {
			return field.NotFound(field.NewPath("Status").Child("CniSubnet"), "in cluster "+clusterName+". Possible cause: Slice Operator installation is pending on the cluster.")
		}
		for _, cniSubnet := range cluster.Status.CniSubnet {
			if util.OverlapIP(cniSubnet, s.Spec.SliceSubnet) {
				return field.Invalid(field.NewPath("Spec").Child("SliceSubnet"), s.Spec.SliceSubnet, "must not overlap with CniSubnet "+cniSubnet+" of cluster "+clusterName)
			}
		}
	}
	return nil
}

// preventUpdate is a function to stop/avoid the update of config of slice
func preventUpdate() *field.Error {
	sliceConfig := controllerv1alpha1.SliceConfig{}
	_, _ = util.GetResourceIfExist(sliceConfigCtx, client.ObjectKey{Name: s.Name, Namespace: s.Namespace}, &sliceConfig)
	if sliceConfig.Spec.SliceSubnet != s.Spec.SliceSubnet {
		return field.Invalid(field.NewPath("Spec").Child("SliceSubnet"), s.Spec.SliceSubnet, "cannot be updated")
	}
	if sliceConfig.Spec.SliceType != s.Spec.SliceType {
		return field.Invalid(field.NewPath("Spec").Child("SliceType"), s.Spec.SliceType, "cannot be updated")
	}
	if sliceConfig.Spec.SliceGatewayProvider.SliceGatewayType != s.Spec.SliceGatewayProvider.SliceGatewayType {
		return field.Invalid(field.NewPath("Spec").Child("SliceGatewayProvider").Child("SliceGatewayType"), s.Spec.SliceGatewayProvider.SliceGatewayType, "cannot be updated")
	}
	if sliceConfig.Spec.SliceGatewayProvider.SliceCaType != s.Spec.SliceGatewayProvider.SliceCaType {
		return field.Invalid(field.NewPath("Spec").Child("SliceGatewayProvider").Child("SliceCaType"), s.Spec.SliceGatewayProvider.SliceCaType, "cannot be updated")
	}
	if sliceConfig.Spec.SliceIpamType != s.Spec.SliceIpamType {
		return field.Invalid(field.NewPath("Spec").Child("SliceIpamType"), s.Spec.SliceIpamType, "cannot be updated")
	}

	return nil
}

// validateQosProfile is a function to validate the Qos(quality of service)profile of slice
func validateQosProfile() *field.Error {
	if s.Spec.QosProfileDetails.BandwidthCeilingKbps < s.Spec.QosProfileDetails.BandwidthGuaranteedKbps {
		return field.Invalid(field.NewPath("Spec").Child("QosProfileDetails").Child("BandwidthGuaranteedKbps"), s.Spec.QosProfileDetails.BandwidthGuaranteedKbps, "BandwidthGuaranteedKbps cannot be greater than BandwidthCeilingKbps")
	}

	return nil
}

// validateExternalGatewayConfig is a function to validate the external gateway
func validateExternalGatewayConfig() *field.Error {
	count := 0
	var allClusters []string
	for _, config := range s.Spec.ExternalGatewayConfig {
		if util.ContainsString(config.Clusters, "*") {
			count++
			if len(config.Clusters) > 1 {
				return field.Invalid(field.NewPath("Spec").Child("ExternalGatewayConfig").Child("Clusters"), strings.Join(config.Clusters, ", "), "other clusters are not allowed when * is present")
			}
		}
		for _, cluster := range config.Clusters {
			allClusters = append(allClusters, cluster)
			if cluster != "*" && !util.ContainsString(s.Spec.Clusters, cluster) {
				return field.Invalid(field.NewPath("Spec").Child("ExternalGatewayConfig").Child("Clusters"), cluster, "cluster is not participating in slice config")
			}
		}
	}
	if count > 1 {
		return field.Invalid(field.NewPath("Spec").Child("ExternalGatewayConfig").Child("Clusters"), "*", "* is not allowed in more than one external gateways")
	}
	if dup, cl := util.CheckDuplicateInArray(allClusters); dup {
		return field.Invalid(field.NewPath("Spec").Child("ExternalGatewayConfig").Child("Clusters"), cl, "duplicate cluster")
	}
	return nil
}

// validateNamespaces is function to validate the application namespaces
func validateNamespaces() *field.Error {
	cluster := controllerv1alpha1.Cluster{}
	for _, applicationNamespaces := range s.Spec.NamespaceIsolationProfile.ApplicationNamespaces {
		if applicationNamespaces.Clusters[0] == "*" {
			for _, clusterSpec := range s.Spec.Clusters {
				err := validateAllowedClusterNamespaces(clusterSpec, cluster, applicationNamespaces, s.GetName())
				if err != nil {
					return err
				}
			}
		} else {
			for _, clusterName := range applicationNamespaces.Clusters {
				cluster := controllerv1alpha1.Cluster{}
				err := validateAllowedClusterNamespaces(clusterName, cluster, applicationNamespaces, s.GetName())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// validateClusterNamespaces is a function to validate the namespaces present is cluster
func validateAllowedClusterNamespaces(clusterName string, cluster controllerv1alpha1.Cluster, applicationNamespaces controllerv1alpha1.SliceNamespaceSelection, sliceName string) *field.Error {
	exist, _ := util.GetResourceIfExist(sliceConfigCtx, client.ObjectKey{Name: clusterName, Namespace: s.Namespace}, &cluster)
	if !exist {
		return field.Invalid(field.NewPath("Spec").Child("Clusters"), clusterName, "cluster is not registered")
	}
	for _, cluster_namespaces := range cluster.Status.Namespaces {
		if applicationNamespaces.Namespace == cluster_namespaces.Name && len(cluster_namespaces.SliceName) > 0 && cluster_namespaces.SliceName != sliceName {
			return field.Invalid(field.NewPath("Spec").Child("SliceConfig.NamespaceIsolationProfile.ApplicationNamespaces"), s.Name, "The given namespace: "+applicationNamespaces.Namespace+" is already acquired by other slice: "+cluster_namespaces.SliceName) // add slice name
		}
	}

	return nil
}
