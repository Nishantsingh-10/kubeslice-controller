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
	"testing"

	"github.com/dailymotion/allure-go"
	apiutil "github.com/kubeslice/apis/pkg/util"
	workerv1alpha1 "github.com/kubeslice/apis/pkg/worker/v1alpha1"
	utilMock "github.com/kubeslice/kubeslice-controller/util/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWorkerSliceGatewayWebhookValidationSuite(t *testing.T) {
	for k, v := range WorkerSliceGatewayWebhookValidationTestBed {
		t.Run(k, func(t *testing.T) {
			allure.Test(t, allure.Name(k),
				allure.Action(func() {
					v(t)
				}))
		})
	}
}

var WorkerSliceGatewayWebhookValidationTestBed = map[string]func(*testing.T){
	"WorkerSliceGatewayWebhookValidation_UpdateValidateWorkerSliceGatewayUpdatingGatewayNumber": UpdateValidateWorkerSliceGatewayUpdatingGatewayNumber,
	"WorkerSliceGatewayWebhookValidation_UpdateValidateWorkerSliceGatewayWithoutErrors":         UpdateValidateWorkerSliceGatewayWithoutErrors,
}

func UpdateValidateWorkerSliceGatewayUpdatingGatewayNumber(t *testing.T) {
	name := "worker_slice_Gateway"
	namespace := "namespace"
	clientMock, newWorkerSliceGateway, ctx := setupWorkerSliceGatewayWebhookValidationTest(name, namespace)
	existingWorkerSliceGateway := workerv1alpha1.WorkerSliceGateway{}
	clientMock.On("Get", ctx, client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, &existingWorkerSliceGateway).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*workerv1alpha1.WorkerSliceGateway)
		arg.Spec.GatewayNumber = 1
	}).Once()
	newWorkerSliceGateway.Spec.GatewayNumber = 2
	err := ValidateWorkerSliceGatewayUpdate(ctx, newWorkerSliceGateway)
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "Spec.GatewayNumber: Invalid value:")
	clientMock.AssertExpectations(t)
}

func UpdateValidateWorkerSliceGatewayWithoutErrors(t *testing.T) {
	name := "worker_slice_Gateway"
	namespace := "namespace"
	clientMock, newWorkerSliceGateway, ctx := setupWorkerSliceGatewayWebhookValidationTest(name, namespace)
	existingWorkerSliceGateway := workerv1alpha1.WorkerSliceGateway{}
	clientMock.On("Get", ctx, client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, &existingWorkerSliceGateway).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*workerv1alpha1.WorkerSliceGateway)
		arg.Spec.GatewayNumber = 1
	}).Once()
	newWorkerSliceGateway.Spec.GatewayNumber = 1
	err := ValidateWorkerSliceGatewayUpdate(ctx, newWorkerSliceGateway)
	require.Nil(t, err)
	clientMock.AssertExpectations(t)
}

func setupWorkerSliceGatewayWebhookValidationTest(name string, namespace string) (*utilMock.Client, *workerv1alpha1.WorkerSliceGateway, context.Context) {
	clientMock := &utilMock.Client{}
	workerSliceGateway := &workerv1alpha1.WorkerSliceGateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	ctx := apiutil.PrepareKubeSliceControllersRequestContext(context.Background(), clientMock, nil, "WorkerSliceGatewayWebhookValidationServiceTest")
	return clientMock, workerSliceGateway, ctx
}
