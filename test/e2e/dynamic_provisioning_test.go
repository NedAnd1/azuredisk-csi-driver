/*
Copyright 2019 The Kubernetes Authors.

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

package e2e

import (
	"context"
	"fmt"

	testconsts "sigs.k8s.io/azuredisk-csi-driver/test/const"
	"sigs.k8s.io/azuredisk-csi-driver/test/e2e/driver"
	"sigs.k8s.io/azuredisk-csi-driver/test/e2e/testsuites"

	"github.com/onsi/ginkgo/v2"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	restclientset "k8s.io/client-go/rest"
	"k8s.io/kubernetes/test/e2e/framework"
	admissionapi "k8s.io/pod-security-admission/api"
	azdisk "sigs.k8s.io/azuredisk-csi-driver/pkg/apis/client/clientset/versioned"
	consts "sigs.k8s.io/azuredisk-csi-driver/pkg/azureconstants"
	"sigs.k8s.io/azuredisk-csi-driver/test/resources"
	"sigs.k8s.io/azuredisk-csi-driver/test/utils/testutil"
)

var _ = ginkgo.Describe("Dynamic Provisioning", func() {
	t := dynamicProvisioningTestSuite{}
	scheduler := testutil.GetSchedulerForE2E()

	ginkgo.Context("[single-az]", func() {
		t.defineTests(false, scheduler)
	})

	ginkgo.Context("[multi-az]", func() {
		t.defineTests(true, scheduler)
	})
})

type dynamicProvisioningTestSuite struct {
	allowedTopologyValues []string
}

func (t *dynamicProvisioningTestSuite) defineTests(isMultiZone bool, schedulerName string) {
	f := framework.NewDefaultFramework("azuredisk")
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelPrivileged

	// Apply the minmally restrictive baseline Pod Security Standard profile to namespaces
	// created by the Kubernetes end-to-end test framework to enable testing with a nil
	// Pod security context.
	f.NamespacePodSecurityEnforceLevel = admissionapi.LevelBaseline

	var (
		cs          clientset.Interface
		ns          *v1.Namespace
		snapshotrcs restclientset.Interface
		testDriver  driver.PVTestDriver
	)

	ginkgo.BeforeEach(func() {
		checkPodsRestart := testutil.TestCmd{
			Command:  "bash",
			Args:     []string{"test/utils/check_driver_pods_restart.sh"},
			StartLog: "Check driver pods if restarts ...",
			EndLog:   "Check successfully",
		}
		testutil.ExecTestCmd([]testutil.TestCmd{checkPodsRestart})

		cs = f.ClientSet
		ns = f.Namespace

		var err error
		snapshotrcs, err = testutil.RestClient(testconsts.SnapshotAPIGroup, testconsts.APIVersionv1)
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("could not get rest clientset: %v", err))
		}

		// Populate allowedTopologyValues from node labels fior the first time
		if isMultiZone && len(t.allowedTopologyValues) == 0 {
			nodes, err := cs.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
			framework.ExpectNoError(err)
			allowedTopologyValuesMap := make(map[string]bool)
			for _, node := range nodes.Items {
				if zone, ok := node.Labels[testconsts.TopologyKey]; ok {
					allowedTopologyValuesMap[zone] = true
				}
			}
			for k := range allowedTopologyValuesMap {
				t.allowedTopologyValues = append(t.allowedTopologyValues, k)
			}
		}
	})

	testDriver = driver.InitAzureDiskDriver()
	ginkgo.It(fmt.Sprintf("should create a volume on demand with mount options [kubernetes.io/azure-disk] [disk.csi.azure.com] [Windows] [%s]", schedulerName), func() {
		pvcSize := "10Gi"
		if isMultiZone {
			pvcSize = "512Gi"
		}
		pods := []resources.PodDetails{
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						ClaimSize: pvcSize,
						MountOptions: []string{
							"barrier=1",
							"acl",
						},
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: testDriver,
			Pods:      pods,
			StorageClassParameters: map[string]string{
				consts.SkuNameField: "Standard_LRS",
			},
		}

		if testconsts.IsUsingInTreeVolumePlugin {
			// cover case: https://github.com/kubernetes/kubernetes/issues/103433
			test.StorageClassParameters = map[string]string{"Kind": "managed"}
		} else if isMultiZone {
			if testutil.IsZRSSupported(location) {
				test.StorageClassParameters[consts.SkuNameField] = "StandardSSD_ZRS"
				test.StorageClassParameters["networkAccessPolicy"] = "AllowAll"
			} else {
				test.StorageClassParameters[consts.SkuNameField] = "UltraSSD_LRS"
				test.StorageClassParameters["diskIopsReadWrite"] = "2000"
				test.StorageClassParameters["diskMbpsReadWrite"] = "320"
				test.StorageClassParameters["logicalSectorSize"] = "512"
			}

			test.StorageClassParameters["cachingmode"] = "None"
			test.StorageClassParameters["zoned"] = "true"
			test.StorageClassParameters["fsType"] = "btrfs"

			test.Pods[0].Volumes[0].MountOptions = []string{"barrier", "acl"}
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("should create a pod with volume mount subpath [disk.csi.azure.com] [Windows]", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()

		pods := []resources.PodDetails{
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data"),
				Volumes: []resources.VolumeDetails{
					{
						ClaimSize: "10Gi",
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				},
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
		}

		scParameters := map[string]string{
			/*hard code skuName key to verify e2e test utils handle storage class params case insensitive*/
			"skuName":             "Standard_LRS",
			"networkAccessPolicy": "DenyAll",
			"userAgent":           "azuredisk-e2e-test",
			"enableAsyncAttach":   "false",
		}
		test := testsuites.DynamicallyProvisionedVolumeSubpathTester{
			CSIDriver:              testDriver,
			Pods:                   pods,
			StorageClassParameters: scParameters,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("Should create and attach a volume with basic perfProfile [enableBursting][disk.csi.azure.com] [Windows]", func() {
		testutil.SkipIfOnAzureStackCloud()
		testutil.SkipIfUsingInTreeVolumePlugin()
		pods := []resources.PodDetails{
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						FSType:    "ext4",
						ClaimSize: "1Ti",
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: testDriver,
			Pods:      pods,
			StorageClassParameters: map[string]string{
				consts.SkuNameField: "Premium_LRS",
				"perfProfile":       "Basic",
				// enableBursting can only be applied to Premium disk, disk size > 512GB, Ultra & shared disk is not supported.
				"enableBursting":    "true",
				"userAgent":         "azuredisk-e2e-test",
				"enableAsyncAttach": "false",
			},
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("Should create and attach a volume with advanced perfProfile [enableBursting][disk.csi.azure.com] [Windows]", func() {
		testutil.SkipIfOnAzureStackCloud()
		testutil.SkipIfUsingInTreeVolumePlugin()
		pods := []resources.PodDetails{
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						FSType:    "ext4",
						ClaimSize: "1Ti",
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: testDriver,
			Pods:      pods,
			StorageClassParameters: map[string]string{
				consts.SkuNameField:                  "Premium_LRS",
				"perfProfile":                        "Advanced",
				"device-setting/queue/read_ahead_kb": "8",
				"device-setting/queue/nomerges":      "0",
				// enableBursting can only be applied to Premium disk, disk size > 512GB, Ultra & shared disk is not supported.
				"enableBursting":    "true",
				"userAgent":         "azuredisk-e2e-test",
				"enableAsyncAttach": "false",
			},
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should receive FailedMount event with invalid mount options [kubernetes.io/azure-disk] [disk.csi.azure.com] [%s]", schedulerName), func() {
		testutil.SkipIfTestingInWindowsCluster()

		pods := []resources.PodDetails{
			{
				Cmd: "echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						ClaimSize: "10Gi",
						MountOptions: []string{
							"invalid",
							"mount",
							"options",
						},
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
			},
		}
		test := testsuites.DynamicallyProvisionedInvalidMountOptions{
			CSIDriver:              testDriver,
			Pods:                   pods,
			StorageClassParameters: map[string]string{consts.SkuNameField: "StandardSSD_LRS"},
		}
		if !testconsts.IsUsingInTreeVolumePlugin && (location == "westus2" || location == "westeurope") {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "StandardSSD_ZRS"}
		}
		if testconsts.IsAzureStackCloud {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "Standard_LRS"}
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should create a raw block volume on demand [kubernetes.io/azure-disk] [disk.csi.azure.com] [%s]", schedulerName), func() {
		testutil.SkipIfTestingInWindowsCluster()

		pods := []resources.PodDetails{
			{
				Cmd: "ls /dev | grep e2e-test",
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						ClaimSize:  "10Gi",
						VolumeMode: resources.Block,
						VolumeDevice: resources.VolumeDeviceDetails{
							NameGenerate: "test-volume-",
							DevicePath:   "/dev/e2e-test",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver:              testDriver,
			Pods:                   pods,
			StorageClassParameters: map[string]string{consts.SkuNameField: "Premium_LRS"},
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "StandardSSD_ZRS"}
		}

		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("should create a volume in separate resource group and bind it to a pod [kubernetes.io/azure-disk] [disk.csi.azure.com]", func() {
		testutil.SkipIfTestingInWindowsCluster()
		testutil.SkipIfUsingInTreeVolumePlugin()

		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					FSType:    "ext4",
					ClaimSize: "10Gi",
					VolumeMount: resources.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
					VolumeAccessMode: v1.ReadWriteOnce,
				},
			}, t.allowedTopologyValues, isMultiZone),
		}

		test := testsuites.DynamicallyProvisionedExternalRgVolumeTest{
			CSIDriver:              testDriver,
			Pod:                    pod,
			StorageClassParameters: map[string]string{"skuName": "Premium_LRS"},
			SeparateResourceGroups: false,
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{"skuName": "StandardSSD_ZRS"}
		}

		test.Run(cs, ns)
	})

	ginkgo.It("should create multiple volumes, each in separate resource groups and attach them to a single pod [kubernetes.io/azure-disk] [disk.csi.azure.com]", func() {
		testutil.SkipIfTestingInWindowsCluster()
		testutil.SkipIfUsingInTreeVolumePlugin()

		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' > /mnt/test-1/data && echo 'hello world' > /mnt/test-2/data && grep 'hello world' /mnt/test-1/data && grep 'hello world' /mnt/test-2/data"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					FSType:    "ext4",
					ClaimSize: "10Gi",
					VolumeMount: resources.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
					VolumeAccessMode: v1.ReadWriteOnce,
				},
				{
					FSType:    "ext4",
					ClaimSize: "10Gi",
					VolumeMount: resources.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
					VolumeAccessMode: v1.ReadWriteOnce,
				},
			}, t.allowedTopologyValues, isMultiZone),
		}

		test := testsuites.DynamicallyProvisionedExternalRgVolumeTest{
			CSIDriver:              testDriver,
			Pod:                    pod,
			StorageClassParameters: map[string]string{"skuName": "Premium_LRS"},
			SeparateResourceGroups: true,
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{"skuName": "StandardSSD_ZRS"}
		}

		test.Run(cs, ns)
	})

	// Track issue https://github.com/kubernetes/kubernetes/issues/70505
	ginkgo.It(fmt.Sprintf("should create a volume on demand and mount it as readOnly in a pod [kubernetes.io/azure-disk] [disk.csi.azure.com] [Windows] [%s]", schedulerName), func() {
		pods := []resources.PodDetails{
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("touch /mnt/test-1/data"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						FSType:    "ext4",
						ClaimSize: "10Gi",
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
							ReadOnly:          true,
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
		}
		test := testsuites.DynamicallyProvisionedReadOnlyVolumeTest{
			CSIDriver:              testDriver,
			Pods:                   pods,
			StorageClassParameters: map[string]string{consts.SkuNameField: "StandardSSD_LRS"},
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "Premium_ZRS"}
		}
		if testconsts.IsAzureStackCloud {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "Standard_LRS"}
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should create multiple PV objects, bind to PVCs and attach all to different pods on the same node [kubernetes.io/azure-disk] [disk.csi.azure.com] [Windows] [%s]", schedulerName), func() {
		pods := []resources.PodDetails{
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("while true; do echo $(date -u) >> /mnt/test-1/data; sleep 3600; done"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						FSType:    "ext3",
						ClaimSize: "10Gi",
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("while true; do echo $(date -u) >> /mnt/test-1/data; sleep 3600; done"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						FSType:    "ext4",
						ClaimSize: "10Gi",
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("while true; do echo $(date -u) >> /mnt/test-1/data; sleep 3600; done"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						FSType:    "xfs",
						ClaimSize: "10Gi",
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
		}
		test := testsuites.DynamicallyProvisionedCollocatedPodTest{
			CSIDriver:              testDriver,
			Pods:                   pods,
			ColocatePods:           true,
			StorageClassParameters: map[string]string{consts.SkuNameField: "Premium_LRS"},
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "StandardSSD_ZRS"}
		}

		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should create a deployment object, write and read to it, delete the pod and write and read to it again [kubernetes.io/azure-disk] [disk.csi.azure.com] [Windows] [%s]", schedulerName), func() {
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					FSType:    "ext3",
					ClaimSize: "10Gi",
					VolumeMount: resources.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
					VolumeAccessMode: v1.ReadWriteOnce,
				},
			}, t.allowedTopologyValues, isMultiZone),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}

		podCheckCmd := []string{"cat", "/mnt/test-1/data"}
		expectedString := "hello world\n"
		if testconsts.IsWindowsCluster {
			podCheckCmd = []string{"cmd", "/c", "type C:\\mnt\\test-1\\data.txt"}
			expectedString = "hello world\r\n"
		}
		test := testsuites.DynamicallyProvisionedDeletePodTest{
			CSIDriver: testDriver,
			Pod:       pod,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:            podCheckCmd,
				ExpectedString: expectedString, // pod will be restarted so expect to see 2 instances of string
			},
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should delete PV with reclaimPolicy %q [kubernetes.io/azure-disk] [disk.csi.azure.com] [Windows]", v1.PersistentVolumeReclaimDelete), func() {
		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		volumes := resources.NormalizeVolumes([]resources.VolumeDetails{
			{
				FSType:           "ext4",
				ClaimSize:        "10Gi",
				ReclaimPolicy:    &reclaimPolicy,
				VolumeAccessMode: v1.ReadWriteOnce,
			},
		}, t.allowedTopologyValues, isMultiZone)
		test := testsuites.DynamicallyProvisionedReclaimPolicyTest{
			CSIDriver: testDriver,
			Volumes:   volumes,
		}
		test.Run(cs, ns)
	})

	ginkgo.It(fmt.Sprintf("should retain PV with reclaimPolicy %q [disk.csi.azure.com]", v1.PersistentVolumeReclaimRetain), func() {
		// This tests uses the CSI driver to delete the PV.
		// TODO: Go via the k8s interfaces and also make it more reliable for in-tree and then
		//       test can be enabled.
		testutil.SkipIfUsingInTreeVolumePlugin()

		reclaimPolicy := v1.PersistentVolumeReclaimRetain
		volumes := resources.NormalizeVolumes([]resources.VolumeDetails{
			{
				FSType:           "ext4",
				ClaimSize:        "10Gi",
				ReclaimPolicy:    &reclaimPolicy,
				VolumeAccessMode: v1.ReadWriteOnce,
			},
		}, t.allowedTopologyValues, isMultiZone)
		test := testsuites.DynamicallyProvisionedReclaimPolicyTest{
			CSIDriver:  testDriver,
			Volumes:    volumes,
			AzureCloud: azureCloud,
		}
		test.Run(cs, ns)
	})

	ginkgo.It(fmt.Sprintf("should clone a volume from an existing volume and read from it [disk.csi.azure.com] [%s]", schedulerName), func() {
		testutil.SkipIfTestingInWindowsCluster()
		testutil.SkipIfUsingInTreeVolumePlugin()

		pod := resources.PodDetails{
			Cmd: "echo 'hello world' > /mnt/test-1/data && fsync /mnt/test-1/data",
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					FSType:    "ext4",
					ClaimSize: "10Gi",
					VolumeMount: resources.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
					VolumeAccessMode: v1.ReadWriteOnce,
				},
			}, t.allowedTopologyValues, isMultiZone),
		}
		podWithClonedVolume := resources.PodDetails{
			Cmd: "grep 'hello world' /mnt/test-1/data",
		}
		test := testsuites.DynamicallyProvisionedVolumeCloningTest{
			CSIDriver:           testDriver,
			Pod:                 pod,
			PodWithClonedVolume: podWithClonedVolume,
			StorageClassParameters: map[string]string{
				consts.SkuNameField: "Standard_LRS",
				"fsType":            "xfs",
			},
		}

		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should clone a volume of larger size than the source volume and make sure the filesystem is appropriately adjusted [disk.csi.azure.com] [%s]", schedulerName), func() {
		testutil.SkipIfTestingInWindowsCluster()
		testutil.SkipIfUsingInTreeVolumePlugin()

		pod := resources.PodDetails{
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					FSType:    "ext4",
					ClaimSize: "10Gi",
					VolumeMount: resources.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
					VolumeAccessMode: v1.ReadWriteOnce,
				},
			}, t.allowedTopologyValues, isMultiZone),
		}
		clonedVolumeSize := "20Gi"

		podWithClonedVolume := resources.PodDetails{
			Cmd: "df -h | grep /mnt/test- | awk '{print $2}' | grep 20.0G",
		}

		test := testsuites.DynamicallyProvisionedVolumeCloningTest{
			CSIDriver:           testDriver,
			Pod:                 pod,
			PodWithClonedVolume: podWithClonedVolume,
			ClonedVolumeSize:    clonedVolumeSize,
			StorageClassParameters: map[string]string{
				consts.SkuNameField: "Standard_LRS",
				"fsType":            "xfs",
			},
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{
				consts.SkuNameField:   "StandardSSD_ZRS",
				"fsType":              "xfs",
				"networkAccessPolicy": "DenyAll",
			}
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should create multiple PV objects, bind to PVCs and attach all to a single pod [kubernetes.io/azure-disk] [disk.csi.azure.com] [Windows] [%s]", schedulerName), func() {
		pods := []resources.PodDetails{
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' > /mnt/test-1/data && echo 'hello world' > /mnt/test-2/data && echo 'hello world' > /mnt/test-3/data && grep 'hello world' /mnt/test-1/data && grep 'hello world' /mnt/test-2/data && grep 'hello world' /mnt/test-3/data"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						FSType:    "ext3",
						ClaimSize: "10Gi",
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
					{
						FSType:    "ext4",
						ClaimSize: "10Gi",
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
					{
						FSType:    "xfs",
						ClaimSize: "10Gi",
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver:              testDriver,
			Pods:                   pods,
			StorageClassParameters: map[string]string{consts.SkuNameField: "StandardSSD_LRS"},
		}
		if testconsts.IsAzureStackCloud {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "Standard_LRS"}
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "StandardSSD_ZRS"}
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should create a raw block volume and a filesystem volume on demand and bind to the same pod [kubernetes.io/azure-disk] [disk.csi.azure.com] [%s]", schedulerName), func() {
		testutil.SkipIfTestingInWindowsCluster()

		pods := []resources.PodDetails{
			{
				Cmd: "dd if=/dev/zero of=/dev/xvda bs=1024k count=100 && echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data",
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						FSType:    "ext4",
						ClaimSize: "10Gi",
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
					{
						FSType:       "ext4",
						MountOptions: []string{"rw"},
						ClaimSize:    "10Gi",
						VolumeMode:   resources.Block,
						VolumeDevice: resources.VolumeDeviceDetails{
							NameGenerate: "test-block-volume-",
							DevicePath:   "/dev/xvda",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver:              testDriver,
			Pods:                   pods,
			StorageClassParameters: map[string]string{consts.SkuNameField: "Premium_LRS"},
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "StandardSSD_ZRS"}
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should create a pod, write and read to it, take a volume snapshot, and create another pod from the snapshot [disk.csi.azure.com] [%s]", schedulerName), func() {
		testutil.SkipIfUsingInTreeVolumePlugin()

		pod := resources.PodDetails{
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			Cmd:          testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' > /mnt/test-1/data"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					FSType:    getFSType(testconsts.IsWindowsCluster),
					ClaimSize: "10Gi",
					VolumeMount: resources.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
					VolumeAccessMode: v1.ReadWriteOnce,
				},
			}, t.allowedTopologyValues, isMultiZone),
		}
		podWithSnapshot := resources.PodDetails{
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			Cmd:          testutil.ConvertToPowershellorCmdCommandIfNecessary("grep 'hello world' /mnt/test-1/data"),
		}
		test := testsuites.DynamicallyProvisionedVolumeSnapshotTest{
			CSIDriver:              testDriver,
			Pod:                    pod,
			ShouldOverwrite:        false,
			PodWithSnapshot:        podWithSnapshot,
			StorageClassParameters: map[string]string{consts.SkuNameField: "StandardSSD_LRS"},
			SnapshotStorageClassParameters: map[string]string{
				"incremental": "false", "dataAccessAuthMode": "AzureActiveDirectory",
			},
		}
		if testconsts.IsAzureStackCloud {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "Standard_LRS"}
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "StandardSSD_ZRS"}
		}
		test.Run(cs, snapshotrcs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should create a pod, write to its pv, take a volume snapshot, overwrite data in original pv, create another pod from the snapshot, and read unaltered original data from original pv[disk.csi.azure.com] [%s]", schedulerName), func() {
		testutil.SkipIfUsingInTreeVolumePlugin()

		pod := resources.PodDetails{
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			Cmd:          testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' > /mnt/test-1/data"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					FSType:    testutil.GetFSType(testconsts.IsWindowsCluster),
					ClaimSize: "10Gi",
					VolumeMount: resources.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
					VolumeAccessMode: v1.ReadWriteOnce,
				},
			}, t.allowedTopologyValues, isMultiZone),
		}

		podOverwrite := resources.PodDetails{
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			Cmd:          testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'overwrite' > /mnt/test-1/data; sleep 3600"),
		}

		podWithSnapshot := resources.PodDetails{
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			Cmd:          testutil.ConvertToPowershellorCmdCommandIfNecessary("grep 'hello world' /mnt/test-1/data"),
		}

		test := testsuites.DynamicallyProvisionedVolumeSnapshotTest{
			CSIDriver:              testDriver,
			Pod:                    pod,
			ShouldOverwrite:        true,
			PodOverwrite:           podOverwrite,
			PodWithSnapshot:        podWithSnapshot,
			StorageClassParameters: map[string]string{consts.SkuNameField: "StandardSSD_LRS"},
			SnapshotStorageClassParameters: map[string]string{
				"incremental": "true", "dataAccessAuthMode": "None",
			},
		}
		if testconsts.IsAzureStackCloud {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "Standard_LRS"}
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "StandardSSD_ZRS"}
		}
		test.Run(cs, snapshotrcs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should create a pod with multiple volumes [kubernetes.io/azure-disk] [disk.csi.azure.com] [Windows] [%s]", schedulerName), func() {
		volumes := []resources.VolumeDetails{}
		for i := 1; i <= 3; i++ {
			volume := resources.VolumeDetails{
				ClaimSize: "10Gi",
				VolumeMount: resources.VolumeMountDetails{
					NameGenerate:      "test-volume-",
					MountPathGenerate: "/mnt/test-",
				},
				VolumeAccessMode: v1.ReadWriteOnce,
			}
			volumes = append(volumes, volume)
		}

		pods := []resources.PodDetails{
			{
				Cmd:          testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data"),
				Volumes:      resources.NormalizeVolumes(volumes, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
		}
		test := testsuites.DynamicallyProvisionedPodWithMultiplePVsTest{
			CSIDriver: testDriver,
			Pods:      pods,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should create a volume on demand and resize it [disk.csi.azure.com] [Windows] [%s]", schedulerName), func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
			VolumeAccessMode: v1.ReadWriteOnce,
		}
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("while true; do echo $(date -u) >> /mnt/test-1/data; sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					ClaimSize: volume.ClaimSize,
					MountOptions: []string{
						"barrier=1",
						"acl",
					},
					VolumeMount:      volume.VolumeMount,
					VolumeAccessMode: v1.ReadWriteOnce,
				},
			}, t.allowedTopologyValues, isMultiZone),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}

		test := testsuites.DynamicallyProvisionedResizeVolumeTest{
			CSIDriver:              testDriver,
			Volume:                 volume,
			Pod:                    pod,
			ResizeOffline:          true,
			StorageClassParameters: map[string]string{consts.SkuNameField: "Standard_LRS"},
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "StandardSSD_ZRS", "fsType": "btrfs"}
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("should create a volume on demand and dynamically resize it without detaching [disk.csi.azure.com] ", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		testutil.SkipIfNotDynamicallyResizeSupported(location)
		//Subscription must be registered for LiveResize
		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
			VolumeAccessMode: v1.ReadWriteOnce,
		}
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("while true; do echo $(date -u) >> /mnt/test-1/data; sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					ClaimSize: volume.ClaimSize,
					MountOptions: []string{
						"barrier=1",
						"acl",
					},
					VolumeMount:      volume.VolumeMount,
					VolumeAccessMode: v1.ReadWriteOnce,
				},
			}, t.allowedTopologyValues, isMultiZone),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}

		test := testsuites.DynamicallyProvisionedResizeVolumeTest{
			CSIDriver:              testDriver,
			Volume:                 volume,
			Pod:                    pod,
			ResizeOffline:          false,
			StorageClassParameters: map[string]string{consts.SkuNameField: "StandardSSD_LRS"},
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("should create a block volume on demand and dynamically resize it without detaching [disk.csi.azure.com] ", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		testutil.SkipIfNotDynamicallyResizeSupported(location)
		//Subscription must be registered for LiveResize
		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeDevice: resources.VolumeDeviceDetails{
				NameGenerate: "test-volume-",
				DevicePath:   "/dev/e2e-test",
			},
			VolumeAccessMode: v1.ReadWriteOnce,
			VolumeMode:       resources.Block,
		}
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("while true; do echo $(date -u) >> /mnt/test-1/data; sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					ClaimSize: volume.ClaimSize,
					MountOptions: []string{
						"barrier=1",
						"acl",
					},
					VolumeMount:      volume.VolumeMount,
					VolumeAccessMode: v1.ReadWriteOnce,
				},
			}, t.allowedTopologyValues, isMultiZone),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}

		test := testsuites.DynamicallyProvisionedResizeVolumeTest{
			CSIDriver:              testDriver,
			Volume:                 volume,
			Pod:                    pod,
			ResizeOffline:          false,
			StorageClassParameters: map[string]string{consts.SkuNameField: "StandardSSD_LRS"},
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("should create a volume azuredisk with tag [disk.csi.azure.com] [Windows]", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		pods := []resources.PodDetails{
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("while true; do echo $(date -u) >> /mnt/test-1/data; sleep 3600; done"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						ClaimSize: "10Gi",
						MountOptions: []string{
							"barrier=1",
							"acl",
						},
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
		}
		tags := "disk=test"
		test := testsuites.DynamicallyProvisionedAzureDiskWithTag{
			CSIDriver:              testDriver,
			Pods:                   pods,
			StorageClassParameters: map[string]string{consts.SkuNameField: "Standard_LRS", "tags": tags},
			Tags:                   tags,
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "StandardSSD_ZRS", "tags": tags}
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should detach disk after pod deleted when maxMountReplicaCount = 0 [disk.csi.azure.com] [Windows] [%s]", schedulerName), func() {
		pods := []resources.PodDetails{
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("while true; do echo $(date -u) >> /mnt/test-1/data; sleep 3600; done"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						ClaimSize: "10Gi",
						MountOptions: []string{
							"barrier=1",
							"acl",
						},
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
		}
		test := testsuites.DynamicallyProvisionedAzureDiskDetach{
			CSIDriver:              testDriver,
			Pods:                   pods,
			StorageClassParameters: map[string]string{consts.SkuNameField: "Standard_LRS"},
		}
		if !testconsts.IsUsingInTreeVolumePlugin && testutil.IsZRSSupported(location) {
			test.StorageClassParameters = map[string]string{consts.SkuNameField: "StandardSSD_ZRS"}
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should delete AzVolumeAttachment after pod deleted when maxMountReplicaCount == 0 [disk.csi.azure.com] [%s]", schedulerName), func() {
		testutil.SkipIfNotUsingCSIDriverV2()

		azDiskClient, err := azdisk.NewForConfig(f.ClientConfig())
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Failed to create disk client. Error: %v", err))
		}

		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
		}

		pod := resources.PodDetails{
			Cmd:          testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes:      resources.NormalizeVolumes([]resources.VolumeDetails{volume}, t.allowedTopologyValues, isMultiZone),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
		}

		storageClassParameters := map[string]string{
			consts.SkuNameField:     "Premium_LRS",
			consts.MaxSharesField:   "1",
			consts.CachingModeField: "None",
		}

		test := testsuites.DynamicallyProvisionedPodDelete{
			CSIDriver:              testDriver,
			Pod:                    pod,
			AzDiskClient:           azDiskClient,
			StorageClassParameters: storageClassParameters,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It(fmt.Sprintf("should demote AzVolumeAttachment after pod deleted when maxMountReplicaCount > 0 [disk.csi.azure.com] [%s]", schedulerName), func() {
		testutil.SkipIfNotUsingCSIDriverV2()
		testutil.SkipIfUsingInTreeVolumePlugin()

		azDiskClient, err := azdisk.NewForConfig(f.ClientConfig())
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Failed to create disk client. Error: %v", err))
		}

		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
		}

		pod := resources.PodDetails{
			Cmd:          testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes:      resources.NormalizeVolumes([]resources.VolumeDetails{volume}, t.allowedTopologyValues, isMultiZone),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
		}

		storageClassParameters := map[string]string{
			consts.SkuNameField:     "Premium_LRS",
			consts.MaxSharesField:   "2",
			consts.CachingModeField: "None",
		}

		test := testsuites.DynamicallyProvisionedPodDelete{
			CSIDriver:              testDriver,
			Pod:                    pod,
			AzDiskClient:           azDiskClient,
			StorageClassParameters: storageClassParameters,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("should create a statefulset object, write and read to it, delete the pod and write and read to it again [kubernetes.io/azure-disk] [disk.csi.azure.com] [Windows]", func() {
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					FSType:    "ext3",
					ClaimSize: "10Gi",
					VolumeMount: resources.VolumeMountDetails{
						NameGenerate:      "pvc",
						MountPathGenerate: "/mnt/test-",
					},
					VolumeAccessMode: v1.ReadWriteOnce,
				},
			}, t.allowedTopologyValues, isMultiZone),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}

		podCheckCmd := []string{"cat", "/mnt/test-1/data"}
		expectedString := "hello world\n"
		if testconsts.IsWindowsCluster {
			podCheckCmd = []string{"cmd", "/c", "type C:\\mnt\\test-1\\data.txt"}
			expectedString = "hello world\r\n"
		}
		test := testsuites.DynamicallyProvisionedStatefulSetTest{
			CSIDriver: testDriver,
			Pod:       pod,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:            podCheckCmd,
				ExpectedString: expectedString, // pod will be restarted so expect to see 2 instances of string
			},
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("Should not create replicas on node with non-tolerable taint", func() {
		ginkgo.Skip("This test is failing randomly. Skipping the test case while the issue is being debugged.")
		testutil.SkipIfUsingInTreeVolumePlugin()
		if isMultiZone {
			ginkgo.Skip("test case does not apply to multi az case")
		}
		testutil.SkipIfNotUsingCSIDriverV2()

		azDiskClient, err := azdisk.NewForConfig(f.ClientConfig())
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Failed to create disk client. Error: %v", err))
		}

		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
			VolumeAccessMode: v1.ReadWriteOnce,
		}
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					ClaimSize: volume.ClaimSize,
					MountOptions: []string{
						"barrier=1",
						"acl",
					},
					VolumeMount:      volume.VolumeMount,
					VolumeAccessMode: volume.VolumeAccessMode,
				},
			}, t.allowedTopologyValues, false),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}

		storageClassParameters := map[string]string{
			consts.SkuNameField:     "Premium_LRS",
			consts.MaxSharesField:   "2",
			consts.CachingModeField: "None",
		}

		test := testsuites.PodToleration{
			CSIDriver:              testDriver,
			Pod:                    pod,
			AzDiskClient:           azDiskClient,
			IsMultiZone:            isMultiZone,
			Volume:                 volume,
			StorageClassParameters: storageClassParameters,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("Should create replicas on node with matching pod node selector", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		if isMultiZone {
			ginkgo.Skip("test case does not apply to multi az case")
		}
		testutil.SkipIfNotUsingCSIDriverV2()

		azDiskClient, err := azdisk.NewForConfig(f.ClientConfig())
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Failed to create disk client. Error: %v", err))
		}

		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
			VolumeAccessMode: v1.ReadWriteOnce,
		}
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					ClaimSize: volume.ClaimSize,
					MountOptions: []string{
						"barrier=1",
						"acl",
					},
					VolumeMount:      volume.VolumeMount,
					VolumeAccessMode: volume.VolumeAccessMode,
				},
			}, t.allowedTopologyValues, false),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}

		storageClassParameters := map[string]string{
			consts.SkuNameField:     "Premium_LRS",
			consts.MaxSharesField:   "2",
			consts.CachingModeField: "None",
		}

		test := testsuites.PodNodeSelector{
			CSIDriver:              testDriver,
			Pod:                    pod,
			AzDiskClient:           azDiskClient,
			IsMultiZone:            isMultiZone,
			Volume:                 volume,
			StorageClassParameters: storageClassParameters,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("Should create replicas on node with matching pod node affinity", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		if isMultiZone {
			ginkgo.Skip("test case does not apply to multi az case")
		}
		testutil.SkipIfNotUsingCSIDriverV2()

		azDiskClient, err := azdisk.NewForConfig(f.ClientConfig())
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Failed to create disk client. Error: %v", err))
		}

		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
			VolumeAccessMode: v1.ReadWriteOnce,
		}
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					ClaimSize: volume.ClaimSize,
					MountOptions: []string{
						"barrier=1",
						"acl",
					},
					VolumeMount:      volume.VolumeMount,
					VolumeAccessMode: volume.VolumeAccessMode,
				},
			}, t.allowedTopologyValues, false),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}

		storageClassParameters := map[string]string{
			consts.SkuNameField:     "Premium_LRS",
			consts.MaxSharesField:   "2",
			consts.CachingModeField: "None",
		}

		test := testsuites.PodNodeAffinity{
			CSIDriver:              testDriver,
			Pod:                    pod,
			IsMultiZone:            isMultiZone,
			AzDiskClient:           azDiskClient,
			Volume:                 volume,
			StorageClassParameters: storageClassParameters,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("Should create replicas on node with matching pod affinity", func() {

		testutil.SkipIfUsingInTreeVolumePlugin()
		if isMultiZone {
			ginkgo.Skip("test case does not apply to multi az case")
		}
		testutil.SkipIfNotUsingCSIDriverV2()

		azDiskClient, err := azdisk.NewForConfig(f.ClientConfig())
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Failed to create disk client. Error: %v", err))
		}

		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
			VolumeAccessMode: v1.ReadWriteOnce,
		}
		pods := []resources.PodDetails{
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						ClaimSize: volume.ClaimSize,
						MountOptions: []string{
							"barrier=1",
							"acl",
						},
						VolumeMount:      volume.VolumeMount,
						VolumeAccessMode: volume.VolumeAccessMode,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
				UseCMD:       false,
			},
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						ClaimSize: volume.ClaimSize,
						MountOptions: []string{
							"barrier=1",
							"acl",
						},
						VolumeMount:      volume.VolumeMount,
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
				UseCMD:       false,
			},
		}

		storageClassParameters := map[string]string{
			consts.SkuNameField:     "Premium_LRS",
			consts.MaxSharesField:   "2",
			consts.CachingModeField: "None",
		}

		test := testsuites.PodAffinity{
			CSIDriver:              testDriver,
			Pods:                   pods,
			IsMultiZone:            isMultiZone,
			AzDiskClient:           azDiskClient,
			Volume:                 volume,
			StorageClassParameters: storageClassParameters,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("Should create replicas spread across zones for zrs", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		if !isMultiZone {
			ginkgo.Skip("test case only applies to the multi az case")
		}

		testutil.SkipIfNotUsingCSIDriverV2()
		testutil.SkipIfNotZRSSupported(location)

		azDiskClient, err := azdisk.NewForConfig(f.ClientConfig())
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Failed to create disk client. Error: %v", err))
		}

		reclaimPolicy := v1.PersistentVolumeReclaimDelete
		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
			ReclaimPolicy:    &reclaimPolicy,
			VolumeAccessMode: v1.ReadWriteOnce,
		}
		volumeBindingMode := storagev1.VolumeBindingImmediate
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes: []resources.VolumeDetails{
				{
					ClaimSize: volume.ClaimSize,
					MountOptions: []string{
						"barrier=1",
						"acl",
					},
					VolumeMount:           volume.VolumeMount,
					VolumeAccessMode:      v1.ReadWriteOnce,
					AllowedTopologyValues: t.allowedTopologyValues,
					VolumeBindingMode:     &volumeBindingMode,
				},
			},
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}

		storageClassParameters := map[string]string{
			consts.SkuNameField:     "StandardSSD_ZRS",
			consts.MaxSharesField:   "2",
			consts.CachingModeField: "None",
		}

		test := testsuites.DynamicallyProvisionedVolumeReplicasAcrossZones{
			CSIDriver:              testDriver,
			Pod:                    pod,
			AzDiskClient:           azDiskClient,
			StorageClassParameters: storageClassParameters,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("Should not create replicas on node with matching pod anti-affinity", func() {

		testutil.SkipIfUsingInTreeVolumePlugin()
		if isMultiZone {
			ginkgo.Skip("test case does not apply to multi az case")
		}
		testutil.SkipIfNotUsingCSIDriverV2()

		azDiskClient, err := azdisk.NewForConfig(f.ClientConfig())
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Failed to create disk client. Error: %v", err))
		}

		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
			VolumeAccessMode: v1.ReadWriteOnce,
		}
		pods := []resources.PodDetails{
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						ClaimSize: volume.ClaimSize,
						MountOptions: []string{
							"barrier=1",
							"acl",
						},
						VolumeMount:      volume.VolumeMount,
						VolumeAccessMode: volume.VolumeAccessMode,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
				UseCMD:       false,
			},
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						ClaimSize: volume.ClaimSize,
						MountOptions: []string{
							"barrier=1",
							"acl",
						},
						VolumeMount:      volume.VolumeMount,
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
				UseCMD:       false,
			},
		}

		storageClassParameters := map[string]string{
			consts.SkuNameField:     "Premium_LRS",
			consts.MaxSharesField:   "2",
			consts.CachingModeField: "None",
		}

		test := testsuites.PodAffinity{
			CSIDriver:              testDriver,
			Pods:                   pods,
			IsMultiZone:            isMultiZone,
			AzDiskClient:           azDiskClient,
			Volume:                 volume,
			StorageClassParameters: storageClassParameters,
			IsAntiAffinityTest:     true,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("Should test pod failover with cordoning a node", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		if isMultiZone {
			ginkgo.Skip("test case does not apply to multi az case")
		}

		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
			VolumeAccessMode: v1.ReadWriteOnce,
		}
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					ClaimSize: volume.ClaimSize,
					MountOptions: []string{
						"barrier=1",
						"acl",
					},
					VolumeMount:      volume.VolumeMount,
					VolumeAccessMode: volume.VolumeAccessMode,
				},
			}, t.allowedTopologyValues, false),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}
		podCheckCmd := []string{"cat", "/mnt/test-1/data"}
		expectedString := "hello world\n"
		if testconsts.IsWindowsCluster {
			podCheckCmd = []string{"cmd", "/c", "type C:\\mnt\\test-1\\data.txt"}
			expectedString = "hello world\r\n"
		}

		storageClassParameters := map[string]string{consts.SkuNameField: "StandardSSD_LRS"}

		test := testsuites.PodFailover{
			CSIDriver: testDriver,
			Pod:       pod,
			Volume:    volume,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:            podCheckCmd,
				ExpectedString: expectedString, // pod will be restarted so expect to see 2 instances of string
			},
			StorageClassParameters: storageClassParameters,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("Should test pod failover with cordoning a node using ZRS", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		testutil.SkipIfNotZRSSupported(location)

		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
			VolumeAccessMode: v1.ReadWriteOnce,
		}
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					ClaimSize: volume.ClaimSize,
					MountOptions: []string{
						"barrier=1",
						"acl",
					},
					VolumeMount:      volume.VolumeMount,
					VolumeAccessMode: volume.VolumeAccessMode,
				},
			}, t.allowedTopologyValues, false),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}
		podCheckCmd := []string{"cat", "/mnt/test-1/data"}
		expectedString := "hello world\n"
		if testconsts.IsWindowsCluster {
			podCheckCmd = []string{"cmd", "/c", "type C:\\mnt\\test-1\\data.txt"}
			expectedString = "hello world\r\n"
		}

		storageClassParameters := map[string]string{consts.SkuNameField: "StandardSSD_ZRS"}

		test := testsuites.PodFailover{
			CSIDriver: testDriver,
			Pod:       pod,
			Volume:    volume,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:            podCheckCmd,
				ExpectedString: expectedString, // pod will be restarted so expect to see 2 instances of string
			},
			StorageClassParameters: storageClassParameters,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("Should test pod failover and check for correct number of replicas", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		skuName := "StandardSSD_LRS"
		if isMultiZone {
			testutil.SkipIfNotZRSSupported(location)
			skuName = "StandardSSD_ZRS"
		}

		// BUG: Issue #1349 Test case currently fails on Windows
		testutil.SkipIfTestingInWindowsCluster()

		azDiskClient, err := azdisk.NewForConfig(f.ClientConfig())
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Failed to create disk client. Error: %v", err))
		}
		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
			VolumeAccessMode: v1.ReadWriteOnce,
		}
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					ClaimSize: volume.ClaimSize,
					MountOptions: []string{
						"barrier=1",
						"acl",
					},
					VolumeMount:      volume.VolumeMount,
					VolumeAccessMode: volume.VolumeAccessMode,
				},
			}, t.allowedTopologyValues, isMultiZone),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}
		podCheckCmd := []string{"cat", "/mnt/test-1/data"}
		expectedString := "hello world\n"
		if testconsts.IsWindowsCluster {
			podCheckCmd = []string{"cmd", "/c", "type C:\\mnt\\test-1\\data.txt"}
			expectedString = "hello world\r\n"
		}

		storageClassParameters := map[string]string{consts.SkuNameField: skuName, "maxShares": "2"}

		test := testsuites.PodFailoverWithReplicas{
			CSIDriver: testDriver,
			Pod:       pod,
			Volume:    volume,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:            podCheckCmd,
				ExpectedString: expectedString, // pod will be restarted so expect to see 2 instances of string
			},
			StorageClassParameters: storageClassParameters,
			AzDiskClient:           azDiskClient,
			IsMultiZone:            isMultiZone,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("Should test an increase in replicas when scaling up", func() {
		ginkgo.Skip("Skip failing test while investigating root cause.")

		testutil.SkipIfUsingInTreeVolumePlugin()
		skuName := "StandardSSD_LRS"
		if isMultiZone {
			testutil.SkipIfNotZRSSupported(location)
			skuName = "StandardSSD_ZRS"
		}
		azDiskClient, err := azdisk.NewForConfig(f.ClientConfig())
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Failed to create disk client. Error: %v", err))
		}
		volume := resources.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: resources.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
		}
		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					ClaimSize: volume.ClaimSize,
					MountOptions: []string{
						"barrier=1",
						"acl",
					},
					VolumeMount: volume.VolumeMount,
				},
			}, t.allowedTopologyValues, isMultiZone),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}
		podCheckCmd := []string{"cat", "/mnt/test-1/data"}
		expectedString := "hello world\n"
		if testconsts.IsWindowsCluster {
			podCheckCmd = []string{"cmd", "/c", "type C:\\mnt\\test-1\\data.txt"}
			expectedString = "hello world\r\n"
		}

		storageClassParameters := map[string]string{consts.SkuNameField: skuName, "maxShares": "3", "cachingMode": "None"}

		test := testsuites.PodNodeScaleUp{
			CSIDriver: testDriver,
			Pod:       pod,
			Volume:    volume,
			PodCheck: &testsuites.PodExecCheck{
				Cmd:            podCheckCmd,
				ExpectedString: expectedString,
			},
			StorageClassParameters: storageClassParameters,
			AzDiskClient:           azDiskClient,
			IsMultiZone:            isMultiZone,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("should succeed when attaching a shared block volume to multiple pods [disk.csi.azure.com][shared disk]", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		testutil.SkipIfOnAzureStackCloud()
		testutil.SkipIfTestingInWindowsCluster()
		if isMultiZone {
			testutil.SkipIfNotZRSSupported(location)
		}

		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("while true; do sleep 5; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					ClaimSize: "10Gi",
					VolumeDevice: resources.VolumeDeviceDetails{
						NameGenerate: "test-volume-",
						DevicePath:   "/dev/e2e-test",
					},
					VolumeMode:       resources.Block,
					VolumeAccessMode: v1.ReadWriteMany,
				},
			}, t.allowedTopologyValues, isMultiZone),
			UseCMD:          false,
			IsWindows:       testconsts.IsWindowsCluster,
			WinServerVer:    testconsts.WinServerVer,
			UseAntiAffinity: isMultiZone,
			ReplicaCount:    2,
		}

		storageClassParameters := map[string]string{
			consts.SkuNameField: "StandardSSD_LRS",
			"maxshares":         "2",
			"cachingmode":       "None",
		}
		if testutil.IsZRSSupported(location) {
			storageClassParameters[consts.SkuNameField] = "StandardSSD_ZRS"
		}

		podCheck := &testsuites.PodExecCheck{
			ExpectedString: "VOLUME ATTACHED",
		}
		if !testconsts.IsWindowsCluster {
			podCheck.Cmd = []string{
				"sh",
				"-c",
				"(stat /dev/e2e-test > /dev/null) && echo \"VOLUME ATTACHED\"",
			}
		} else {
			podCheck.Cmd = []string{
				"powershell",
				"-NoLogo",
				"-Command",
				"if (Test-Path c:\\dev\\e2e-test) { \"VOLUME ATTACHED\" | Out-Host }",
			}
		}

		test := testsuites.DynamicallyProvisionedSharedDiskTester{
			CSIDriver:              testDriver,
			Pod:                    pod,
			PodCheck:               podCheck,
			StorageClassParameters: storageClassParameters,
		}
		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("should succeed with advanced perfProfile [disk.csi.azure.com] [Windows]", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		testutil.SkipIfOnAzureStackCloud()
		testutil.SkipIfTestingInWindowsCluster()
		pods := []resources.PodDetails{
			{
				Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' > /mnt/test-1/data && grep 'hello world' /mnt/test-1/data"),
				Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
					{
						ClaimSize: "10Gi",
						MountOptions: []string{
							"barrier=1",
							"acl",
						},
						VolumeMount: resources.VolumeMountDetails{
							NameGenerate:      "test-volume-",
							MountPathGenerate: "/mnt/test-",
						},
						VolumeAccessMode: v1.ReadWriteOnce,
					},
				}, t.allowedTopologyValues, isMultiZone),
				IsWindows:    testconsts.IsWindowsCluster,
				WinServerVer: testconsts.WinServerVer,
			},
		}
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: testDriver,
			Pods:      pods,
			StorageClassParameters: map[string]string{
				"skuname":                             "StandardSSD_LRS",
				"perfProfile":                         "advanced",
				"device-setting/queue/max_sectors_kb": "211",
				"device-setting/queue/scheduler":      "none",
				"device-setting/device/queue_depth":   "17",
				"device-setting/queue/nr_requests":    "44",
				"device-setting/queue/read_ahead_kb":  "256",
				"device-setting/queue/wbt_lat_usec":   "0",
				"device-setting/queue/rotational":     "0",
			},
		}

		test.Run(cs, ns, schedulerName)
	})

	ginkgo.It("should check failed replica attachments are recreated after space is made from a volume detaching.", func() {
		testutil.SkipIfUsingInTreeVolumePlugin()
		skuName := "StandardSSD_LRS"
		if isMultiZone {
			testutil.SkipIfNotZRSSupported(location)
			skuName = "StandardSSD_ZRS"
		}

		// TODO: Disable flakey test until #1378 is fixed.
		ginkgo.Skip("Skip flakey test until #1378 is fixed.")

		azDiskClient, err := azdisk.NewForConfig(f.ClientConfig())
		framework.ExpectNoError(err, "Failed to create disk client.")

		pod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("while true; do echo $(date -u) >> /mnt/test-1/data; sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					FSType:    "ext3",
					ClaimSize: "5Gi",
					VolumeMount: resources.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			}, []string{}, isMultiZone),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
		}

		newPod := resources.PodDetails{
			Cmd: testutil.ConvertToPowershellorCmdCommandIfNecessary("echo 'hello world' >> /mnt/test-1/data && while true; do sleep 3600; done"),
			Volumes: resources.NormalizeVolumes([]resources.VolumeDetails{
				{
					ClaimSize: "5Gi",
					MountOptions: []string{
						"barrier=1",
						"acl",
					},
					VolumeMount: resources.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			}, t.allowedTopologyValues, isMultiZone),
			IsWindows:    testconsts.IsWindowsCluster,
			WinServerVer: testconsts.WinServerVer,
			UseCMD:       false,
		}

		test := testsuites.DynamicallyProvisionedScaleReplicasOnDetach{
			CSIDriver:              testDriver,
			StatefulSetPod:         pod,
			NewPod:                 newPod,
			StorageClassParameters: map[string]string{consts.SkuNameField: skuName, "maxShares": "2", "cachingMode": "None"},
			AzDiskClient:           azDiskClient,
		}
		test.Run(cs, ns, schedulerName)
	})
}
