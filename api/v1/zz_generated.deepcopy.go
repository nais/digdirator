// +build !ignore_autogenerated

/*


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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IDPortenClient) DeepCopyInto(out *IDPortenClient) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IDPortenClient.
func (in *IDPortenClient) DeepCopy() *IDPortenClient {
	if in == nil {
		return nil
	}
	out := new(IDPortenClient)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IDPortenClient) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IDPortenClientList) DeepCopyInto(out *IDPortenClientList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]IDPortenClient, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IDPortenClientList.
func (in *IDPortenClientList) DeepCopy() *IDPortenClientList {
	if in == nil {
		return nil
	}
	out := new(IDPortenClientList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *IDPortenClientList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IDPortenClientSpec) DeepCopyInto(out *IDPortenClientSpec) {
	*out = *in
	if in.ReplyURLs != nil {
		in, out := &in.ReplyURLs, &out.ReplyURLs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.PostLogoutRedirectURIs != nil {
		in, out := &in.PostLogoutRedirectURIs, &out.PostLogoutRedirectURIs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Scopes != nil {
		in, out := &in.Scopes, &out.Scopes
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IDPortenClientSpec.
func (in *IDPortenClientSpec) DeepCopy() *IDPortenClientSpec {
	if in == nil {
		return nil
	}
	out := new(IDPortenClientSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *IDPortenClientStatus) DeepCopyInto(out *IDPortenClientStatus) {
	*out = *in
	in.Timestamp.DeepCopyInto(&out.Timestamp)
	if in.KeyIDs != nil {
		in, out := &in.KeyIDs, &out.KeyIDs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new IDPortenClientStatus.
func (in *IDPortenClientStatus) DeepCopy() *IDPortenClientStatus {
	if in == nil {
		return nil
	}
	out := new(IDPortenClientStatus)
	in.DeepCopyInto(out)
	return out
}
