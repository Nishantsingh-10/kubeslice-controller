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
	"errors"
	"fmt"
	"testing"

	"github.com/dailymotion/allure-go"
	controllerv1alpha1 "github.com/kubeslice/kubeslice-controller/apis/controller/v1alpha1"
	"github.com/kubeslice/kubeslice-controller/service/mocks"
	"github.com/kubeslice/kubeslice-controller/util"
	utilMock "github.com/kubeslice/kubeslice-controller/util/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8sError "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSliceConfigSuite(t *testing.T) {
	for k, v := range SliceConfigTestBed {
		t.Run(k, func(t *testing.T) {
			allure.Test(t, allure.Name(k),
				allure.Action(func() {
					v(t)
				}))
		})
	}
}

var SliceConfigTestBed = map[string]func(*testing.T){
	"SliceConfig_ReconciliationCompleteHappyCase":                            SliceConfigReconciliationCompleteHappyCase,
	"SliceConfig_GetObjectErrorOtherThanNotFound":                            SliceConfigGetObjectErrorOtherThanNotFound,
	"SliceConfig_GetObjectErrorNotFound":                                     SliceConfigGetObjectErrorNotFound,
	"SliceConfig_DeleteTheObjectHappyCase":                                   SliceConfigDeleteTheObjectHappyCase,
	"SliceConfig_ObjectNamespaceNotFound":                                    SliceConfigObjectNamespaceNotFound,
	"SliceConfig_ObjectNotInProjectNamespace":                                SliceConfigObjectNotInProjectNamespace,
	"SliceConfig_ObjectWithDuplicateClustersInSpec":                          SliceConfigObjectWithDuplicateClustersInSpec,
	"SliceConfig_ErrorOnCreateWorkerSliceConfig":                             SliceConfigErrorOnCreateWorkerSliceConfig,
	"SliceConfig_ErrorOnCreateWorkerSliceGateway":                            SliceConfigErrorOnCreateWorkerSliceGateway,
	"SliceConfig_ErrorOnDeleteWorkerSliceGatewaysByLabel":                    SliceConfigErrorOnDeleteWorkerSliceGatewaysByLabel,
	"SliceConfig_ErrorOnDeleteWorkerSliceConfigByLabel":                      SliceConfigErrorOnDeleteWorkerSliceConfigByLabel,
	"SliceConfig_ErrorOnUpdatingTheFinalizer":                                SliceConfigErrorOnUpdatingTheFinalizer,
	"SliceConfig_RemoveFinalizerErrorOnUpdate":                               SliceConfigRemoveFinalizerErrorOnUpdate,
	"SliceConfig_RemoveFinalizerErrorOnGetAfterUpdate":                       SliceConfigRemoveFinalizerErrorOnGetAfterUpdate,
	"SliceConfig_DeleteHappyCase":                                            SliceConfigDeleteHappyCase,
	"SliceConfig_DeleteErrorOnList":                                          SliceConfigDeleteErrorOnList,
	"SliceConfig_DeleteErrorOnDelete":                                        SliceConfigDeleteErrorOnDelete,
	"SliceConfig_ErrorOnListingServiceExport":                                SliceConfigErrorOnListingServiceExport,
	"SliceConfig_ErrorOnCreateOrUpdateServiceImport":                         SliceConfigErrorOnCreateOrUpdateServiceImport,
}

func SliceConfigReconciliationCompleteHappyCase(t *testing.T) {
	workerSliceGatewayMock, workerSliceConfigMock, _, workerServiceImportMock, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Once()
	clientMock.On("Update", ctx, mock.Anything).Return(nil).Once()
	clientMock.On("Get", ctx, mock.Anything, mock.Anything).Return(nil).Once()
	namespace := corev1.Namespace{}
	clientMock.On("Get", ctx, mock.Anything, &namespace).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*corev1.Namespace)
		if arg.Labels == nil {
			arg.Labels = make(map[string]string)
		}
		arg.Name = requestObj.Namespace
		arg.Labels[util.LabelName] = fmt.Sprintf(util.LabelValue, "Project", requestObj.Namespace)
	}).Once()
	clusterMap := map[string]int{
		"cluster-1": 1,
		"cluster-2": 2,
	}
	workerSliceConfigMock.On("CreateMinimalWorkerSliceConfig", ctx, mock.Anything, requestObj.Namespace, mock.Anything, mock.Anything, mock.Anything).Return(clusterMap, nil).Once()
	workerSliceGatewayMock.On("CreateMinimumWorkerSliceGateways", ctx, mock.Anything, mock.Anything, requestObj.Namespace, mock.Anything, clusterMap, mock.Anything).Return(ctrl.Result{}, nil).Once()
	label := map[string]string{
		"original-slice-name": sliceConfig.Name,
	}
	serviceExportList := &controllerv1alpha1.ServiceExportConfigList{}
	clientMock.On("List", ctx, serviceExportList, client.InNamespace(requestObj.Namespace), client.MatchingLabels(label)).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(1).(*controllerv1alpha1.ServiceExportConfigList)
		arg.Items = []controllerv1alpha1.ServiceExportConfig{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service-export-config-1",
					Namespace: requestObj.Namespace,
				},
			},
		}
	}).Once()
	workerServiceImportMock.On("CreateMinimalWorkerServiceImport", ctx, sliceConfig.Spec.Clusters, requestObj.Namespace, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	result, err := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.NoError(t, nil)
	require.Equal(t, expectedResult, result)
	require.Nil(t, err)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
	workerSliceConfigMock.AssertExpectations(t)
	workerSliceGatewayMock.AssertExpectations(t)
	workerServiceImportMock.AssertExpectations(t)
}

func SliceConfigGetObjectErrorOtherThanNotFound(t *testing.T) {
	_, _, _, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	err1 := errors.New("internal_error")
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(err1).Once()
	result, err2 := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.Error(t, err2)
	require.Equal(t, expectedResult, result)
	require.Equal(t, err1, err2)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
}

func SliceConfigGetObjectErrorNotFound(t *testing.T) {
	_, _, _, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	notFoundError := k8sError.NewNotFound(util.Resource("SliceConfigTest"), "isNotFound")
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(notFoundError).Once()
	result, err2 := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.NoError(t, nil)
	require.Equal(t, expectedResult, result)
	require.Nil(t, err2)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
}

func SliceConfigDeleteTheObjectHappyCase(t *testing.T) {
	workerSliceGatewayMock, workerSliceConfigMock, serviceExportConfigMock, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	time := metav1.Now()
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*controllerv1alpha1.SliceConfig)
		arg.ObjectMeta.DeletionTimestamp = &time
	}).Once()
	workerSliceGatewayMock.On("DeleteWorkerSliceGatewaysByLabel", ctx, mock.Anything, requestObj.Namespace).Return(nil).Once()
	workerSliceConfigMock.On("DeleteWorkerSliceConfigByLabel", ctx, mock.Anything, requestObj.Namespace).Return(nil).Once()
	//remove finalizer
	clientMock.On("Update", ctx, mock.Anything).Return(nil).Once()
	result, err := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.NoError(t, nil)
	require.Equal(t, expectedResult, result)
	require.Nil(t, err)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
	workerSliceGatewayMock.AssertExpectations(t)
	workerSliceConfigMock.AssertExpectations(t)
	serviceExportConfigMock.AssertExpectations(t)
}

func SliceConfigObjectNamespaceNotFound(t *testing.T) {
	_, _, _, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Once()
	clientMock.On("Update", ctx, mock.Anything).Return(nil).Once()
	clientMock.On("Get", ctx, mock.Anything, mock.Anything).Return(nil).Once()
	notFoundError := k8sError.NewNotFound(util.Resource("SliceConfigTest"), "isNotFound")
	namespace := corev1.Namespace{}
	clientMock.On("Get", ctx, mock.Anything, &namespace).Return(notFoundError).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*corev1.Namespace)
		if arg.Labels == nil {
			arg.Labels = make(map[string]string)
		}
		arg.Name = requestObj.Namespace
		arg.Labels[util.LabelName] = fmt.Sprintf(util.LabelValue, "Project", requestObj.Namespace)
	}).Once()
	result, err := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.NoError(t, nil)
	require.Equal(t, expectedResult, result)
	require.Nil(t, err)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
}

func SliceConfigObjectNotInProjectNamespace(t *testing.T) {
	_, _, _, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Once()
	clientMock.On("Update", ctx, mock.Anything).Return(nil).Once()
	clientMock.On("Get", ctx, mock.Anything, mock.Anything).Return(nil).Once()
	namespace := corev1.Namespace{}
	clientMock.On("Get", ctx, mock.Anything, &namespace).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*corev1.Namespace)
		if arg.Labels == nil {
			arg.Labels = make(map[string]string)
		}
		arg.Name = requestObj.Namespace
		arg.Labels[util.LabelName] = fmt.Sprintf(util.LabelValue, "SomeOther", requestObj.Namespace)
	}).Once()
	result, err := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.NoError(t, nil)
	require.Equal(t, expectedResult, result)
	require.Nil(t, err)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
}

func SliceConfigObjectWithDuplicateClustersInSpec(t *testing.T) {
	_, _, _, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*controllerv1alpha1.SliceConfig)
		if arg.Spec.Clusters == nil {
			arg.Spec.Clusters = []string{}
		}
		arg.Spec.Clusters = []string{"cluster-1", "cluster-1"}
	}).Once()
	result, err := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.NoError(t, nil)
	require.Equal(t, expectedResult, result)
	require.Nil(t, err)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
}

func SliceConfigErrorOnCreateWorkerSliceConfig(t *testing.T) {
	_, workerSliceConfigMock, _, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Once()
	clientMock.On("Update", ctx, mock.Anything).Return(nil).Once()
	clientMock.On("Get", ctx, mock.Anything, mock.Anything).Return(nil).Once()
	namespace := corev1.Namespace{}
	clientMock.On("Get", ctx, mock.Anything, &namespace).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*corev1.Namespace)
		if arg.Labels == nil {
			arg.Labels = make(map[string]string)
		}
		arg.Name = requestObj.Namespace
		arg.Labels[util.LabelName] = fmt.Sprintf(util.LabelValue, "Project", requestObj.Namespace)
	}).Once()
	clusterMap := map[string]int{
		"cluster-1": 1,
		"cluster-2": 2,
	}
	err1 := errors.New("internal_error")
	workerSliceConfigMock.On("CreateMinimalWorkerSliceConfig", ctx, mock.Anything, requestObj.Namespace, mock.Anything, mock.Anything, mock.Anything).Return(clusterMap, err1).Once()
	result, err2 := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.Error(t, err2)
	require.Equal(t, expectedResult, result)
	require.Equal(t, err1, err2)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
	workerSliceConfigMock.AssertExpectations(t)
}

func SliceConfigErrorOnCreateWorkerSliceGateway(t *testing.T) {
	workerSliceGatewayMock, workerSliceConfigMock, _, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Once()
	clientMock.On("Update", ctx, mock.Anything).Return(nil).Once()
	clientMock.On("Get", ctx, mock.Anything, mock.Anything).Return(nil).Once()
	namespace := corev1.Namespace{}
	clientMock.On("Get", ctx, mock.Anything, &namespace).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*corev1.Namespace)
		if arg.Labels == nil {
			arg.Labels = make(map[string]string)
		}
		arg.Name = requestObj.Namespace
		arg.Labels[util.LabelName] = fmt.Sprintf(util.LabelValue, "Project", requestObj.Namespace)
	}).Once()
	clusterMap := map[string]int{
		"cluster-1": 1,
		"cluster-2": 2,
	}
	workerSliceConfigMock.On("CreateMinimalWorkerSliceConfig", ctx, mock.Anything, requestObj.Namespace, mock.Anything, mock.Anything, mock.Anything).Return(clusterMap, nil).Once()
	err1 := errors.New("internal_error")
	workerSliceGatewayMock.On("CreateMinimumWorkerSliceGateways", ctx, mock.Anything, mock.Anything, requestObj.Namespace, mock.Anything, clusterMap, mock.Anything).Return(ctrl.Result{}, err1).Once()
	result, err2 := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.Error(t, err2)
	require.Equal(t, expectedResult, result)
	require.Equal(t, err1, err2)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
	workerSliceConfigMock.AssertExpectations(t)
	workerSliceGatewayMock.AssertExpectations(t)
}

func SliceConfigErrorOnDeleteWorkerSliceGatewaysByLabel(t *testing.T) {
	workerSliceGatewayMock, _, serviceExportConfigMock, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	time := metav1.Now()
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*controllerv1alpha1.SliceConfig)
		arg.ObjectMeta.DeletionTimestamp = &time
	}).Once()
	err1 := errors.New("internal_error")
	workerSliceGatewayMock.On("DeleteWorkerSliceGatewaysByLabel", ctx, mock.Anything, requestObj.Namespace).Return(err1).Once()
	result, err2 := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.Error(t, err2)
	require.Equal(t, expectedResult, result)
	require.Equal(t, err1, err2)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
	serviceExportConfigMock.AssertExpectations(t)
	workerSliceGatewayMock.AssertExpectations(t)
}

func SliceConfigErrorOnDeleteWorkerSliceConfigByLabel(t *testing.T) {
	workerSliceGatewayMock, workerSliceConfigMock, serviceExportConfigMock, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	time := metav1.Now()
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*controllerv1alpha1.SliceConfig)
		arg.ObjectMeta.DeletionTimestamp = &time
	}).Once()
	err1 := errors.New("internal_error")
	workerSliceGatewayMock.On("DeleteWorkerSliceGatewaysByLabel", ctx, mock.Anything, requestObj.Namespace).Return(nil).Once()
	workerSliceConfigMock.On("DeleteWorkerSliceConfigByLabel", ctx, mock.Anything, requestObj.Namespace).Return(err1).Once()
	result, err2 := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.Error(t, err2)
	require.Equal(t, expectedResult, result)
	require.Equal(t, err1, err2)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
	serviceExportConfigMock.AssertExpectations(t)
	workerSliceGatewayMock.AssertExpectations(t)
	workerSliceConfigMock.AssertExpectations(t)
}

func SliceConfigErrorOnUpdatingTheFinalizer(t *testing.T) {
	_, _, _, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Once()
	err1 := errors.New("internal_error")
	clientMock.On("Update", ctx, mock.Anything).Return(err1).Once()
	result, err2 := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.Error(t, err2)
	require.Equal(t, expectedResult, result)
	require.Equal(t, err1, err2)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
}

func SliceConfigRemoveFinalizerErrorOnUpdate(t *testing.T) {
	workerSliceGatewayMock, workerSliceConfigMock, serviceExportConfigMock, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	time := metav1.Now()
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*controllerv1alpha1.SliceConfig)
		arg.ObjectMeta.DeletionTimestamp = &time
	}).Once()
	workerSliceGatewayMock.On("DeleteWorkerSliceGatewaysByLabel", ctx, mock.Anything, requestObj.Namespace).Return(nil).Once()
	workerSliceConfigMock.On("DeleteWorkerSliceConfigByLabel", ctx, mock.Anything, requestObj.Namespace).Return(nil).Once()
	err1 := errors.New("internal_error")
	clientMock.On("Update", ctx, mock.Anything).Return(err1).Once()
	result, err2 := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.Error(t, err2)
	require.Equal(t, expectedResult, result)
	require.Equal(t, err1, err2)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
	workerSliceGatewayMock.AssertExpectations(t)
	workerSliceConfigMock.AssertExpectations(t)
	serviceExportConfigMock.AssertExpectations(t)
}

func SliceConfigRemoveFinalizerErrorOnGetAfterUpdate(t *testing.T) {
	workerSliceGatewayMock, workerSliceConfigMock, serviceExportConfigMock, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	time := metav1.Now()
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*controllerv1alpha1.SliceConfig)
		arg.ObjectMeta.DeletionTimestamp = &time
	}).Once()
	workerSliceGatewayMock.On("DeleteWorkerSliceGatewaysByLabel", ctx, mock.Anything, requestObj.Namespace).Return(nil).Once()
	workerSliceConfigMock.On("DeleteWorkerSliceConfigByLabel", ctx, mock.Anything, requestObj.Namespace).Return(nil).Once()
	//remove finalizer
	err1 := errors.New("internal_error")
	clientMock.On("Update", ctx, mock.Anything).Return(err1).Once()
	result, err2 := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.Error(t, err2)
	require.Equal(t, expectedResult, result)
	require.Equal(t, err1, err2)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
	workerSliceGatewayMock.AssertExpectations(t)
	workerSliceConfigMock.AssertExpectations(t)
	serviceExportConfigMock.AssertExpectations(t)
}

func SliceConfigDeleteHappyCase(t *testing.T) {
	name := "slice-1"
	namespace := "namespace"
	_, _, _, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest(name, namespace)
	clientMock.On("List", ctx, &controllerv1alpha1.SliceConfigList{}, client.InNamespace(requestObj.Namespace)).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(1).(*controllerv1alpha1.SliceConfigList)
		arg.Items = []controllerv1alpha1.SliceConfig{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: requestObj.Namespace,
				},
			},
		}
	}).Once()
	sliceConfig.Name = name
	sliceConfig.Namespace = namespace
	clientMock.On("Delete", ctx, sliceConfig).Return(nil).Once()
	result, err := sliceConfigService.DeleteSliceConfigs(ctx, requestObj.Namespace)
	expectedResult := ctrl.Result{}
	require.NoError(t, nil)
	require.Equal(t, expectedResult, result)
	require.Nil(t, err)
	clientMock.AssertExpectations(t)
}

func SliceConfigDeleteErrorOnList(t *testing.T) {
	name := "slice-1"
	namespace := "namespace"
	_, _, _, _, requestObj, clientMock, _, ctx, sliceConfigService := setupSliceConfigTest(name, namespace)
	err1 := errors.New("internal_error")
	clientMock.On("List", ctx, &controllerv1alpha1.SliceConfigList{}, client.InNamespace(requestObj.Namespace)).Return(err1).Run(func(args mock.Arguments) {
		arg := args.Get(1).(*controllerv1alpha1.SliceConfigList)
		arg.Items = []controllerv1alpha1.SliceConfig{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: requestObj.Namespace,
				},
			},
		}
	}).Once()
	result, err2 := sliceConfigService.DeleteSliceConfigs(ctx, requestObj.Namespace)
	expectedResult := ctrl.Result{}
	require.Error(t, err2)
	require.Equal(t, expectedResult, result)
	require.Equal(t, err1, err2)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
}

func SliceConfigDeleteErrorOnDelete(t *testing.T) {
	name := "slice-1"
	namespace := "namespace"
	_, _, _, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest(name, namespace)
	clientMock.On("List", ctx, &controllerv1alpha1.SliceConfigList{}, client.InNamespace(requestObj.Namespace)).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(1).(*controllerv1alpha1.SliceConfigList)
		arg.Items = []controllerv1alpha1.SliceConfig{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: requestObj.Namespace,
				},
			},
		}
	}).Once()
	sliceConfig.Name = name
	sliceConfig.Namespace = namespace
	err1 := errors.New("internal_error")
	clientMock.On("Delete", ctx, sliceConfig).Return(err1).Once()
	result, err2 := sliceConfigService.DeleteSliceConfigs(ctx, requestObj.Namespace)
	expectedResult := ctrl.Result{}
	require.Error(t, err2)
	require.Equal(t, expectedResult, result)
	require.Equal(t, err1, err2)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
}

func SliceConfigErrorOnListingServiceExport(t *testing.T) {
	workerSliceGatewayMock, workerSliceConfigMock, _, _, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Once()
	clientMock.On("Update", ctx, mock.Anything).Return(nil).Once()
	clientMock.On("Get", ctx, mock.Anything, mock.Anything).Return(nil).Once()
	namespace := corev1.Namespace{}
	clientMock.On("Get", ctx, mock.Anything, &namespace).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*corev1.Namespace)
		if arg.Labels == nil {
			arg.Labels = make(map[string]string)
		}
		arg.Name = requestObj.Namespace
		arg.Labels[util.LabelName] = fmt.Sprintf(util.LabelValue, "Project", requestObj.Namespace)
	}).Once()
	clusterMap := map[string]int{
		"cluster-1": 1,
		"cluster-2": 2,
	}
	workerSliceConfigMock.On("CreateMinimalWorkerSliceConfig", ctx, mock.Anything, requestObj.Namespace, mock.Anything, mock.Anything, mock.Anything).Return(clusterMap, nil).Once()
	workerSliceGatewayMock.On("CreateMinimumWorkerSliceGateways", ctx, mock.Anything, mock.Anything, requestObj.Namespace, mock.Anything, clusterMap, mock.Anything).Return(ctrl.Result{}, nil).Once()
	label := map[string]string{
		"original-slice-name": sliceConfig.Name,
	}
	serviceExportList := &controllerv1alpha1.ServiceExportConfigList{}
	err1 := errors.New("internal_error")
	clientMock.On("List", ctx, serviceExportList, client.InNamespace(requestObj.Namespace), client.MatchingLabels(label)).Return(err1).Once()
	result, err := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.Error(t, err)
	require.Equal(t, expectedResult, result)
	require.Equal(t, err, err1)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
	workerSliceConfigMock.AssertExpectations(t)
	workerSliceGatewayMock.AssertExpectations(t)
}

func SliceConfigErrorOnCreateOrUpdateServiceImport(t *testing.T) {
	workerSliceGatewayMock, workerSliceConfigMock, _, workerServiceImportMock, requestObj, clientMock, sliceConfig, ctx, sliceConfigService := setupSliceConfigTest("slice_config", "namespace")
	clientMock.On("Get", ctx, requestObj.NamespacedName, sliceConfig).Return(nil).Once()
	clientMock.On("Update", ctx, mock.Anything).Return(nil).Once()
	clientMock.On("Get", ctx, mock.Anything, mock.Anything).Return(nil).Once()
	namespace := corev1.Namespace{}
	clientMock.On("Get", ctx, mock.Anything, &namespace).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*corev1.Namespace)
		if arg.Labels == nil {
			arg.Labels = make(map[string]string)
		}
		arg.Name = requestObj.Namespace
		arg.Labels[util.LabelName] = fmt.Sprintf(util.LabelValue, "Project", requestObj.Namespace)
	}).Once()
	clusterMap := map[string]int{
		"cluster-1": 1,
		"cluster-2": 2,
	}
	workerSliceConfigMock.On("CreateMinimalWorkerSliceConfig", ctx, mock.Anything, requestObj.Namespace, mock.Anything, mock.Anything, mock.Anything).Return(clusterMap, nil).Once()
	workerSliceGatewayMock.On("CreateMinimumWorkerSliceGateways", ctx, mock.Anything, mock.Anything, requestObj.Namespace, mock.Anything, clusterMap, mock.Anything).Return(ctrl.Result{}, nil).Once()
	label := map[string]string{
		"original-slice-name": sliceConfig.Name,
	}
	serviceExportList := &controllerv1alpha1.ServiceExportConfigList{}
	err1 := errors.New("internal_error")
	clientMock.On("List", ctx, serviceExportList, client.InNamespace(requestObj.Namespace), client.MatchingLabels(label)).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(1).(*controllerv1alpha1.ServiceExportConfigList)
		arg.Items = []controllerv1alpha1.ServiceExportConfig{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service-export-config-1",
					Namespace: requestObj.Namespace,
				},
			},
		}
	}).Once()
	workerServiceImportMock.On("CreateMinimalWorkerServiceImport", ctx, sliceConfig.Spec.Clusters, requestObj.Namespace, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(err1).Once()
	result, err := sliceConfigService.ReconcileSliceConfig(ctx, requestObj)
	expectedResult := ctrl.Result{}
	require.Error(t, err)
	require.Equal(t, expectedResult, result)
	require.Equal(t, err, err1)
	require.False(t, result.Requeue)
	clientMock.AssertExpectations(t)
	workerSliceConfigMock.AssertExpectations(t)
	workerSliceGatewayMock.AssertExpectations(t)
	workerServiceImportMock.AssertExpectations(t)
}

func setupSliceConfigTest(name string, namespace string) (*mocks.IWorkerSliceGatewayService, *mocks.IWorkerSliceConfigService, *mocks.IServiceExportConfigService, *mocks.IWorkerServiceImportService, ctrl.Request, *utilMock.Client, *controllerv1alpha1.SliceConfig, context.Context, SliceConfigService) {
	workerSliceGatewayMock := &mocks.IWorkerSliceGatewayService{}
	workerSliceConfigMock := &mocks.IWorkerSliceConfigService{}
	serviceExportConfigMock := &mocks.IServiceExportConfigService{}
	workerServiceImportMock := &mocks.IWorkerServiceImportService{}
	sliceConfigService := SliceConfigService{
		sgs: workerSliceGatewayMock,
		ms:  workerSliceConfigMock,
		se:  serviceExportConfigMock,
		si:  workerServiceImportMock,
	}
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	requestObj := ctrl.Request{
		NamespacedName: namespacedName,
	}
	clientMock := &utilMock.Client{}
	sliceConfig := &controllerv1alpha1.SliceConfig{}
	ctx := util.PrepareKubeSliceControllersRequestContext(context.Background(), clientMock, nil, "SliceConfigServiceTest")
	return workerSliceGatewayMock, workerSliceConfigMock, serviceExportConfigMock, workerServiceImportMock, requestObj, clientMock, sliceConfig, ctx, sliceConfigService
}
