//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright The Kubernetes Authors.

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

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1beta2

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzDiskDriverConfiguration) DeepCopyInto(out *AzDiskDriverConfiguration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ControllerConfig = in.ControllerConfig
	out.NodeConfig = in.NodeConfig
	out.CloudConfig = in.CloudConfig
	out.ClientConfig = in.ClientConfig
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzDiskDriverConfiguration.
func (in *AzDiskDriverConfiguration) DeepCopy() *AzDiskDriverConfiguration {
	if in == nil {
		return nil
	}
	out := new(AzDiskDriverConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzDriverCondition) DeepCopyInto(out *AzDriverCondition) {
	*out = *in
	in.LastHeartbeatTime.DeepCopyInto(&out.LastHeartbeatTime)
	in.LastTransitionTime.DeepCopyInto(&out.LastTransitionTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzDriverCondition.
func (in *AzDriverCondition) DeepCopy() *AzDriverCondition {
	if in == nil {
		return nil
	}
	out := new(AzDriverCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzDriverNode) DeepCopyInto(out *AzDriverNode) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(AzDriverNodeStatus)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzDriverNode.
func (in *AzDriverNode) DeepCopy() *AzDriverNode {
	if in == nil {
		return nil
	}
	out := new(AzDriverNode)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AzDriverNode) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzDriverNodeList) DeepCopyInto(out *AzDriverNodeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AzDriverNode, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzDriverNodeList.
func (in *AzDriverNodeList) DeepCopy() *AzDriverNodeList {
	if in == nil {
		return nil
	}
	out := new(AzDriverNodeList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AzDriverNodeList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzDriverNodeSpec) DeepCopyInto(out *AzDriverNodeSpec) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzDriverNodeSpec.
func (in *AzDriverNodeSpec) DeepCopy() *AzDriverNodeSpec {
	if in == nil {
		return nil
	}
	out := new(AzDriverNodeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzDriverNodeStatus) DeepCopyInto(out *AzDriverNodeStatus) {
	*out = *in
	if in.LastHeartbeatTime != nil {
		in, out := &in.LastHeartbeatTime, &out.LastHeartbeatTime
		*out = (*in).DeepCopy()
	}
	if in.ReadyForVolumeAllocation != nil {
		in, out := &in.ReadyForVolumeAllocation, &out.ReadyForVolumeAllocation
		*out = new(bool)
		**out = **in
	}
	if in.StatusMessage != nil {
		in, out := &in.StatusMessage, &out.StatusMessage
		*out = new(string)
		**out = **in
	}
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]AzDriverCondition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzDriverNodeStatus.
func (in *AzDriverNodeStatus) DeepCopy() *AzDriverNodeStatus {
	if in == nil {
		return nil
	}
	out := new(AzDriverNodeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzError) DeepCopyInto(out *AzError) {
	*out = *in
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzError.
func (in *AzError) DeepCopy() *AzError {
	if in == nil {
		return nil
	}
	out := new(AzError)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzVolume) DeepCopyInto(out *AzVolume) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzVolume.
func (in *AzVolume) DeepCopy() *AzVolume {
	if in == nil {
		return nil
	}
	out := new(AzVolume)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AzVolume) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzVolumeAttachment) DeepCopyInto(out *AzVolumeAttachment) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzVolumeAttachment.
func (in *AzVolumeAttachment) DeepCopy() *AzVolumeAttachment {
	if in == nil {
		return nil
	}
	out := new(AzVolumeAttachment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AzVolumeAttachment) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzVolumeAttachmentList) DeepCopyInto(out *AzVolumeAttachmentList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AzVolumeAttachment, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzVolumeAttachmentList.
func (in *AzVolumeAttachmentList) DeepCopy() *AzVolumeAttachmentList {
	if in == nil {
		return nil
	}
	out := new(AzVolumeAttachmentList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AzVolumeAttachmentList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzVolumeAttachmentSpec) DeepCopyInto(out *AzVolumeAttachmentSpec) {
	*out = *in
	if in.VolumeContext != nil {
		in, out := &in.VolumeContext, &out.VolumeContext
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzVolumeAttachmentSpec.
func (in *AzVolumeAttachmentSpec) DeepCopy() *AzVolumeAttachmentSpec {
	if in == nil {
		return nil
	}
	out := new(AzVolumeAttachmentSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzVolumeAttachmentStatus) DeepCopyInto(out *AzVolumeAttachmentStatus) {
	*out = *in
	if in.Detail != nil {
		in, out := &in.Detail, &out.Detail
		*out = new(AzVolumeAttachmentStatusDetail)
		(*in).DeepCopyInto(*out)
	}
	if in.Error != nil {
		in, out := &in.Error, &out.Error
		*out = new(AzError)
		(*in).DeepCopyInto(*out)
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzVolumeAttachmentStatus.
func (in *AzVolumeAttachmentStatus) DeepCopy() *AzVolumeAttachmentStatus {
	if in == nil {
		return nil
	}
	out := new(AzVolumeAttachmentStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzVolumeAttachmentStatusDetail) DeepCopyInto(out *AzVolumeAttachmentStatusDetail) {
	*out = *in
	if in.PublishContext != nil {
		in, out := &in.PublishContext, &out.PublishContext
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzVolumeAttachmentStatusDetail.
func (in *AzVolumeAttachmentStatusDetail) DeepCopy() *AzVolumeAttachmentStatusDetail {
	if in == nil {
		return nil
	}
	out := new(AzVolumeAttachmentStatusDetail)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzVolumeList) DeepCopyInto(out *AzVolumeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AzVolume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzVolumeList.
func (in *AzVolumeList) DeepCopy() *AzVolumeList {
	if in == nil {
		return nil
	}
	out := new(AzVolumeList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AzVolumeList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzVolumeSpec) DeepCopyInto(out *AzVolumeSpec) {
	*out = *in
	if in.VolumeCapability != nil {
		in, out := &in.VolumeCapability, &out.VolumeCapability
		*out = make([]VolumeCapability, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.CapacityRange != nil {
		in, out := &in.CapacityRange, &out.CapacityRange
		*out = new(CapacityRange)
		**out = **in
	}
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Secrets != nil {
		in, out := &in.Secrets, &out.Secrets
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ContentVolumeSource != nil {
		in, out := &in.ContentVolumeSource, &out.ContentVolumeSource
		*out = new(ContentVolumeSource)
		**out = **in
	}
	if in.AccessibilityRequirements != nil {
		in, out := &in.AccessibilityRequirements, &out.AccessibilityRequirements
		*out = new(TopologyRequirement)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzVolumeSpec.
func (in *AzVolumeSpec) DeepCopy() *AzVolumeSpec {
	if in == nil {
		return nil
	}
	out := new(AzVolumeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzVolumeStatus) DeepCopyInto(out *AzVolumeStatus) {
	*out = *in
	if in.Detail != nil {
		in, out := &in.Detail, &out.Detail
		*out = new(AzVolumeStatusDetail)
		(*in).DeepCopyInto(*out)
	}
	if in.Error != nil {
		in, out := &in.Error, &out.Error
		*out = new(AzError)
		(*in).DeepCopyInto(*out)
	}
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzVolumeStatus.
func (in *AzVolumeStatus) DeepCopy() *AzVolumeStatus {
	if in == nil {
		return nil
	}
	out := new(AzVolumeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzVolumeStatusDetail) DeepCopyInto(out *AzVolumeStatusDetail) {
	*out = *in
	if in.VolumeContext != nil {
		in, out := &in.VolumeContext, &out.VolumeContext
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ContentSource != nil {
		in, out := &in.ContentSource, &out.ContentSource
		*out = new(ContentVolumeSource)
		**out = **in
	}
	if in.AccessibleTopology != nil {
		in, out := &in.AccessibleTopology, &out.AccessibleTopology
		*out = make([]Topology, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzVolumeStatusDetail.
func (in *AzVolumeStatusDetail) DeepCopy() *AzVolumeStatusDetail {
	if in == nil {
		return nil
	}
	out := new(AzVolumeStatusDetail)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CapacityRange) DeepCopyInto(out *CapacityRange) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CapacityRange.
func (in *CapacityRange) DeepCopy() *CapacityRange {
	if in == nil {
		return nil
	}
	out := new(CapacityRange)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClientConfiguration) DeepCopyInto(out *ClientConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClientConfiguration.
func (in *ClientConfiguration) DeepCopy() *ClientConfiguration {
	if in == nil {
		return nil
	}
	out := new(ClientConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudConfiguration) DeepCopyInto(out *CloudConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudConfiguration.
func (in *CloudConfiguration) DeepCopy() *CloudConfiguration {
	if in == nil {
		return nil
	}
	out := new(CloudConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContentVolumeSource) DeepCopyInto(out *ContentVolumeSource) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContentVolumeSource.
func (in *ContentVolumeSource) DeepCopy() *ContentVolumeSource {
	if in == nil {
		return nil
	}
	out := new(ContentVolumeSource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerConfiguration) DeepCopyInto(out *ControllerConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerConfiguration.
func (in *ControllerConfiguration) DeepCopy() *ControllerConfiguration {
	if in == nil {
		return nil
	}
	out := new(ControllerConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ListSnapshotsResult) DeepCopyInto(out *ListSnapshotsResult) {
	*out = *in
	if in.Entries != nil {
		in, out := &in.Entries, &out.Entries
		*out = make([]Snapshot, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ListSnapshotsResult.
func (in *ListSnapshotsResult) DeepCopy() *ListSnapshotsResult {
	if in == nil {
		return nil
	}
	out := new(ListSnapshotsResult)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ListVolumesResult) DeepCopyInto(out *ListVolumesResult) {
	*out = *in
	if in.Entries != nil {
		in, out := &in.Entries, &out.Entries
		*out = make([]VolumeEntry, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ListVolumesResult.
func (in *ListVolumesResult) DeepCopy() *ListVolumesResult {
	if in == nil {
		return nil
	}
	out := new(ListVolumesResult)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeConfiguration) DeepCopyInto(out *NodeConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeConfiguration.
func (in *NodeConfiguration) DeepCopy() *NodeConfiguration {
	if in == nil {
		return nil
	}
	out := new(NodeConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Snapshot) DeepCopyInto(out *Snapshot) {
	*out = *in
	in.CreationTime.DeepCopyInto(&out.CreationTime)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Snapshot.
func (in *Snapshot) DeepCopy() *Snapshot {
	if in == nil {
		return nil
	}
	out := new(Snapshot)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Topology) DeepCopyInto(out *Topology) {
	*out = *in
	if in.Segments != nil {
		in, out := &in.Segments, &out.Segments
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Topology.
func (in *Topology) DeepCopy() *Topology {
	if in == nil {
		return nil
	}
	out := new(Topology)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TopologyRequirement) DeepCopyInto(out *TopologyRequirement) {
	*out = *in
	if in.Requisite != nil {
		in, out := &in.Requisite, &out.Requisite
		*out = make([]Topology, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Preferred != nil {
		in, out := &in.Preferred, &out.Preferred
		*out = make([]Topology, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TopologyRequirement.
func (in *TopologyRequirement) DeepCopy() *TopologyRequirement {
	if in == nil {
		return nil
	}
	out := new(TopologyRequirement)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeCapability) DeepCopyInto(out *VolumeCapability) {
	*out = *in
	if in.MountFlags != nil {
		in, out := &in.MountFlags, &out.MountFlags
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeCapability.
func (in *VolumeCapability) DeepCopy() *VolumeCapability {
	if in == nil {
		return nil
	}
	out := new(VolumeCapability)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeCondition) DeepCopyInto(out *VolumeCondition) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeCondition.
func (in *VolumeCondition) DeepCopy() *VolumeCondition {
	if in == nil {
		return nil
	}
	out := new(VolumeCondition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeDetails) DeepCopyInto(out *VolumeDetails) {
	*out = *in
	if in.VolumeContext != nil {
		in, out := &in.VolumeContext, &out.VolumeContext
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.ContentSource != nil {
		in, out := &in.ContentSource, &out.ContentSource
		*out = new(ContentVolumeSource)
		**out = **in
	}
	if in.AccessibleTopology != nil {
		in, out := &in.AccessibleTopology, &out.AccessibleTopology
		*out = make([]Topology, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeDetails.
func (in *VolumeDetails) DeepCopy() *VolumeDetails {
	if in == nil {
		return nil
	}
	out := new(VolumeDetails)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeEntry) DeepCopyInto(out *VolumeEntry) {
	*out = *in
	if in.Details != nil {
		in, out := &in.Details, &out.Details
		*out = new(VolumeDetails)
		(*in).DeepCopyInto(*out)
	}
	if in.Status != nil {
		in, out := &in.Status, &out.Status
		*out = new(VolumeStatus)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeEntry.
func (in *VolumeEntry) DeepCopy() *VolumeEntry {
	if in == nil {
		return nil
	}
	out := new(VolumeEntry)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VolumeStatus) DeepCopyInto(out *VolumeStatus) {
	*out = *in
	if in.PublishedNodeIds != nil {
		in, out := &in.PublishedNodeIds, &out.PublishedNodeIds
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Condition != nil {
		in, out := &in.Condition, &out.Condition
		*out = new(VolumeCondition)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VolumeStatus.
func (in *VolumeStatus) DeepCopy() *VolumeStatus {
	if in == nil {
		return nil
	}
	out := new(VolumeStatus)
	in.DeepCopyInto(out)
	return out
}
