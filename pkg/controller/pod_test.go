/*
Copyright 2021 The Kubernetes Authors.

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
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"
	fakev1 "k8s.io/client-go/kubernetes/fake"
	"k8s.io/klog/v2/klogr"
	azdiskv1beta2 "sigs.k8s.io/azuredisk-csi-driver/pkg/apis/azuredisk/v1beta2"
	azdiskfakes "sigs.k8s.io/azuredisk-csi-driver/pkg/apis/client/clientset/versioned/fake"
	consts "sigs.k8s.io/azuredisk-csi-driver/pkg/azureconstants"
	"sigs.k8s.io/azuredisk-csi-driver/pkg/azureutils"
	"sigs.k8s.io/azuredisk-csi-driver/pkg/azureutils/mockclient"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func NewTestPodController(controller *gomock.Controller, namespace string, objects ...runtime.Object) *ReconcilePod {
	azDiskObjs, kubeObjs := splitObjects(objects...)
	controllerSharedState := initState(mockclient.NewMockClient(controller), azdiskfakes.NewSimpleClientset(azDiskObjs...), fakev1.NewSimpleClientset(kubeObjs...), objects...)

	return &ReconcilePod{
		namespace:   namespace,
		SharedState: controllerSharedState,
		logger:      klogr.New(),
	}
}

func TestPodReconcile(t *testing.T) {
	tests := []struct {
		description string
		request     reconcile.Request
		setupFunc   func(*testing.T, *gomock.Controller) *ReconcilePod
		verifyFunc  func(*testing.T, *ReconcilePod, reconcile.Result, error)
	}{
		{
			description: "[Success] Should create AzVolume object for pod inline volume.",
			request:     testPod0Request,
			setupFunc: func(t *testing.T, mockCtl *gomock.Controller) *ReconcilePod {
				newPod := testPod0.DeepCopy()
				newPod.Spec.Volumes = append(newPod.Spec.Volumes, v1.Volume{
					Name: testManagedDiskURI0,
					VolumeSource: v1.VolumeSource{
						AzureDisk: &v1.AzureDiskVolumeSource{
							Kind:        (*v1.AzureDataDiskKind)(to.StringPtr(string(v1.AzureManagedDisk))),
							DataDiskURI: testManagedDiskURI0,
						},
					},
				})

				controller := NewTestPodController(
					mockCtl,
					testNamespace,
					newPod,
					&testNode0,
				)

				addTestNodeInAvailableAttachmentsMap(controller.SharedState, testNode0.Name, testNodeAvailableAttachmentCount)

				mockClients(controller.cachedClient.(*mockclient.MockClient), controller.azClient, controller.kubeClient)
				return controller
			},
			verifyFunc: func(t *testing.T, controller *ReconcilePod, result reconcile.Result, err error) {
				require.NoError(t, err)
				require.False(t, result.Requeue)

				azVolume, err := controller.azClient.DiskV1beta2().AzVolumes(testNamespace).Get(context.TODO(), testAzVolume0.Spec.VolumeName, metav1.GetOptions{})
				require.NoError(t, err)
				require.NotNil(t, azVolume)

				// check the azVolume's pv and pvc labels are not added
				require.NotContains(t, azVolume.Labels, consts.PvNameLabel)
				require.NotContains(t, azVolume.Labels, consts.PvcNameLabel)

				// check the azVolume is annotated with inlineVolumeAnnotation
				require.Contains(t, azVolume.Status.Annotations, consts.InlineVolumeAnnotation)
			},
		},
		{
			description: "[Success] Should attach replica attachment upon pod start.",
			request:     testPod0Request,
			setupFunc: func(t *testing.T, mockCtl *gomock.Controller) *ReconcilePod {
				newAttachment := testPrimaryAzVolumeAttachment0.DeepCopy()
				newAttachment.Status.State = azdiskv1beta2.Attached

				newVolume := testAzVolume0.DeepCopy()
				newVolume.Status.Detail = &azdiskv1beta2.AzVolumeStatusDetail{
					VolumeID: testManagedDiskURI0,
				}

				newPod := testPod0.DeepCopy()
				newPod.Status.Phase = v1.PodRunning

				controller := NewTestPodController(
					mockCtl,
					testNamespace,
					newVolume,
					newAttachment,
					&testNode0,
					&testNode1,
					&testPersistentVolume0,
					newPod)

				addTestNodeInAvailableAttachmentsMap(controller.SharedState, testNode0.Name, testNodeAvailableAttachmentCount)
				addTestNodeInAvailableAttachmentsMap(controller.SharedState, testNode1.Name, testNodeAvailableAttachmentCount)

				mockClients(controller.cachedClient.(*mockclient.MockClient), controller.azClient, controller.kubeClient)
				return controller
			},
			verifyFunc: func(t *testing.T, controller *ReconcilePod, result reconcile.Result, err error) {
				require.NoError(t, err)
				require.False(t, result.Requeue)

				roleReq, _ := azureutils.CreateLabelRequirements(consts.RoleLabel, selection.Equals, string(azdiskv1beta2.ReplicaRole))
				labelSelector := labels.NewSelector().Add(*roleReq)
				conditionFunc := func() (bool, error) {
					replicas, localError := controller.azClient.DiskV1beta2().AzVolumeAttachments(testPrimaryAzVolumeAttachment0.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector.String()})
					require.NoError(t, localError)
					require.NotNil(t, replicas)
					return len(replicas.Items) == 1, nil
				}
				err = wait.PollImmediate(verifyCRIInterval, verifyCRITimeout, conditionFunc)
				require.NoError(t, err)
			},
		},
		{
			description: "[Success] Should attach replica attachments of the volumes in same pod to same node.",
			request:     testPod1Request,
			setupFunc: func(t *testing.T, mockCtl *gomock.Controller) *ReconcilePod {
				newAttachment0 := testPrimaryAzVolumeAttachment0.DeepCopy()
				newAttachment0.Status.State = azdiskv1beta2.Attached
				newAttachment1 := testPrimaryAzVolumeAttachment1.DeepCopy()
				newAttachment1.Status.State = azdiskv1beta2.Attached

				newVolume0 := testAzVolume0.DeepCopy()
				newVolume0.Status.Detail = &azdiskv1beta2.AzVolumeStatusDetail{
					VolumeID: testManagedDiskURI0,
				}

				newVolume1 := testAzVolume1.DeepCopy()
				newVolume1.Status.Detail = &azdiskv1beta2.AzVolumeStatusDetail{
					VolumeID: testManagedDiskURI1,
				}

				newPod := testPod1.DeepCopy()
				newPod.Status.Phase = v1.PodRunning

				controller := NewTestPodController(
					mockCtl,
					testNamespace,
					newVolume0,
					newVolume1,
					newAttachment0,
					newAttachment1,
					&testPersistentVolume0,
					&testPersistentVolume1,
					&testNode0,
					&testNode1,
					&testNode2,
					newPod)

				addTestNodeInAvailableAttachmentsMap(controller.SharedState, testNode0.Name, testNodeAvailableAttachmentCount)
				addTestNodeInAvailableAttachmentsMap(controller.SharedState, testNode1.Name, testNodeAvailableAttachmentCount)
				addTestNodeInAvailableAttachmentsMap(controller.SharedState, testNode2.Name, testNodeAvailableAttachmentCount)

				mockClients(controller.cachedClient.(*mockclient.MockClient), controller.azClient, controller.kubeClient)
				return controller
			},
			verifyFunc: func(t *testing.T, controller *ReconcilePod, result reconcile.Result, err error) {
				require.NoError(t, err)
				require.False(t, result.Requeue)

				roleReq, _ := azureutils.CreateLabelRequirements(consts.RoleLabel, selection.Equals, string(azdiskv1beta2.ReplicaRole))
				labelSelector := labels.NewSelector().Add(*roleReq)
				conditionFunc := func() (bool, error) {
					replicas, localError := controller.azClient.DiskV1beta2().AzVolumeAttachments(testPrimaryAzVolumeAttachment0.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector.String()})
					require.NoError(t, localError)
					require.NotNil(t, replicas)
					if len(replicas.Items) == 2 {
						return replicas.Items[0].Spec.NodeName == replicas.Items[1].Spec.NodeName, nil
					}
					return false, nil
				}
				err = wait.PollImmediate(verifyCRIInterval, verifyCRITimeout, conditionFunc)
				require.NoError(t, err)
			},
		},
		{
			description: "[Success] Should attach replica attachments of the volumes in multiple pods to same nodes if any volume is shared.",
			request:     testPod1Request,
			setupFunc: func(t *testing.T, mockCtl *gomock.Controller) *ReconcilePod {
				newAttachment0 := testPrimaryAzVolumeAttachment0.DeepCopy()
				newAttachment0.Status.State = azdiskv1beta2.Attached
				newAttachment1 := testPrimaryAzVolumeAttachment1.DeepCopy()
				newAttachment1.Status.State = azdiskv1beta2.Attached

				newVolume0 := testAzVolume0.DeepCopy()
				newVolume0.Status.Detail = &azdiskv1beta2.AzVolumeStatusDetail{
					VolumeID: testManagedDiskURI0,
				}

				newVolume1 := testAzVolume1.DeepCopy()
				newVolume1.Status.Detail = &azdiskv1beta2.AzVolumeStatusDetail{
					VolumeID: testManagedDiskURI1,
				}

				newPod0 := testPod0.DeepCopy()
				newPod0.Status.Phase = v1.PodRunning

				newPod1 := testPod1.DeepCopy()
				newPod1.Status.Phase = v1.PodRunning

				controller := NewTestPodController(
					mockCtl,
					testNamespace,
					newVolume0,
					newVolume1,
					newAttachment0,
					newAttachment1,
					&testPersistentVolume0,
					&testPersistentVolume1,
					&testNode0,
					&testNode1,
					&testNode2,
					newPod0,
					newPod1)

				addTestNodeInAvailableAttachmentsMap(controller.SharedState, testNode0.Name, testNodeAvailableAttachmentCount)
				addTestNodeInAvailableAttachmentsMap(controller.SharedState, testNode1.Name, testNodeAvailableAttachmentCount)
				addTestNodeInAvailableAttachmentsMap(controller.SharedState, testNode2.Name, testNodeAvailableAttachmentCount)

				mockClients(controller.cachedClient.(*mockclient.MockClient), controller.azClient, controller.kubeClient)
				result, err := controller.Reconcile(context.TODO(), testPod0Request)
				require.False(t, result.Requeue)
				require.NoError(t, err)
				return controller
			},
			verifyFunc: func(t *testing.T, controller *ReconcilePod, result reconcile.Result, err error) {
				require.NoError(t, err)
				require.False(t, result.Requeue)

				roleReq, _ := azureutils.CreateLabelRequirements(consts.RoleLabel, selection.Equals, string(azdiskv1beta2.ReplicaRole))
				labelSelector := labels.NewSelector().Add(*roleReq)
				conditionFunc := func() (bool, error) {
					replicas, localError := controller.azClient.DiskV1beta2().AzVolumeAttachments(testPrimaryAzVolumeAttachment0.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector.String()})
					require.NoError(t, localError)
					require.NotNil(t, replicas)
					if len(replicas.Items) == 2 {
						return replicas.Items[0].Spec.NodeName == replicas.Items[1].Spec.NodeName, nil
					}
					return false, nil
				}
				err = wait.PollImmediate(verifyCRIInterval, verifyCRITimeout, conditionFunc)
				require.NoError(t, err)
			},
		},
		{
			description: "[Success] Should remove entry from podToClaimsMap and claimToPodMap if pod has been deleted.",
			request:     testPod0Request,
			setupFunc: func(t *testing.T, mockCtl *gomock.Controller) *ReconcilePod {
				newPod0 := testPod0.DeepCopy()

				newAttachment := testPrimaryAzVolumeAttachment0.DeepCopy()
				newAttachment.Status.State = azdiskv1beta2.Attached

				newVolume := testAzVolume0.DeepCopy()
				newVolume.Status.Detail = &azdiskv1beta2.AzVolumeStatusDetail{
					VolumeID: testManagedDiskURI0,
				}

				controller := NewTestPodController(
					mockCtl,
					testNamespace,
					newVolume,
					newAttachment,
					&testNode0,
					&testNode1,
					&testPersistentVolume0,
					newPod0)

				mockClients(controller.cachedClient.(*mockclient.MockClient), controller.azClient, controller.kubeClient)

				result, err := controller.Reconcile(context.TODO(), testPod0Request)
				require.NoError(t, err)
				require.False(t, result.Requeue)

				podKey := getQualifiedName(testNamespace, testPod0Name)
				_, ok := controller.podToClaimsMap.Load(podKey)
				require.True(t, ok)

				namespacedClaimName := getQualifiedName(testNamespace, testPersistentVolumeClaim0Name)
				_, ok = controller.claimToPodsMap.Load(namespacedClaimName)
				require.True(t, ok)

				err = controller.kubeClient.CoreV1().Pods(testNamespace).Delete(context.TODO(), testPod0Name, metav1.DeleteOptions{})
				require.NoError(t, err)

				return controller
			},
			verifyFunc: func(t *testing.T, controller *ReconcilePod, result reconcile.Result, err error) {
				require.NoError(t, err)
				require.False(t, result.Requeue)

				podKey := getQualifiedName(testNamespace, testPod0Name)
				_, ok := controller.podToClaimsMap.Load(podKey)
				require.False(t, ok)

				namespacedClaimName := getQualifiedName(testNamespace, testPersistentVolumeClaim0Name)
				_, ok = controller.claimToPodsMap.Load(namespacedClaimName)
				require.False(t, ok)
			},
		},
	}

	for _, test := range tests {
		tt := test
		t.Run(tt.description, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			controller := tt.setupFunc(t, mockCtl)
			result, err := controller.Reconcile(context.TODO(), tt.request)
			tt.verifyFunc(t, controller, result, err)
		})
	}
}

func TestPodRecover(t *testing.T) {
	tests := []struct {
		description string
		setupFunc   func(*testing.T, *gomock.Controller) *ReconcilePod
		verifyFunc  func(*testing.T, *ReconcilePod, error)
	}{
		{
			description: "[Success] Should create replica AzVolumeAttachment instances for volumes for all pods in proper nodes.",
			setupFunc: func(t *testing.T, mockCtl *gomock.Controller) *ReconcilePod {
				newPod := testPod1.DeepCopy()
				newPod.Status.Phase = v1.PodRunning

				newAttachment0 := testPrimaryAzVolumeAttachment0.DeepCopy()
				newAttachment0.Status.State = azdiskv1beta2.Attached
				newAttachment1 := testPrimaryAzVolumeAttachment1.DeepCopy()
				newAttachment1.Status.State = azdiskv1beta2.Attached

				newVolume0 := testAzVolume0.DeepCopy()
				newVolume0.Status.Detail = &azdiskv1beta2.AzVolumeStatusDetail{
					VolumeID: testManagedDiskURI0,
				}

				newVolume1 := testAzVolume1.DeepCopy()
				newVolume1.Status.Detail = &azdiskv1beta2.AzVolumeStatusDetail{
					VolumeID: testManagedDiskURI1,
				}

				controller := NewTestPodController(
					mockCtl,
					testNamespace,
					newVolume0,
					newVolume1,
					&testPersistentVolume0,
					&testPersistentVolume1,
					&testNode0,
					&testNode1,
					newAttachment0,
					newAttachment1,
					newPod)

				addTestNodeInAvailableAttachmentsMap(controller.SharedState, testNode0.Name, testNodeAvailableAttachmentCount)
				addTestNodeInAvailableAttachmentsMap(controller.SharedState, testNode1.Name, testNodeAvailableAttachmentCount)

				mockClients(controller.cachedClient.(*mockclient.MockClient), controller.azClient, controller.kubeClient)
				return controller
			},
			verifyFunc: func(t *testing.T, controller *ReconcilePod, err error) {
				require.NoError(t, err)

				roleReq, _ := azureutils.CreateLabelRequirements(consts.RoleLabel, selection.Equals, string(azdiskv1beta2.ReplicaRole))
				labelSelector := labels.NewSelector().Add(*roleReq)
				conditionFunc := func() (bool, error) {
					replicas, localError := controller.azClient.DiskV1beta2().AzVolumeAttachments(testPrimaryAzVolumeAttachment0.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector.String()})
					require.NoError(t, localError)
					require.NotNil(t, replicas)
					if len(replicas.Items) == 2 {
						return replicas.Items[0].Spec.NodeName == replicas.Items[1].Spec.NodeName, nil
					}
					return false, nil
				}
				err = wait.PollImmediate(verifyCRIInterval, verifyCRITimeout, conditionFunc)
				require.NoError(t, err)
			},
		},
	}

	for _, test := range tests {
		tt := test
		t.Run(tt.description, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			controller := tt.setupFunc(t, mockCtl)
			err := controller.Recover(context.TODO())
			tt.verifyFunc(t, controller, err)
		})
	}
}
