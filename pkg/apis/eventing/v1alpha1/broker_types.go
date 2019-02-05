/*
 * Copyright 2019 The Knative Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import (
	"github.com/knative/pkg/apis"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/webhook"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Broker struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state of the Broker.
	Spec BrokerSpec `json:"spec,omitempty"`

	// Status represents the current state of the Broker. This data may be out of
	// date.
	// +optional
	Status BrokerStatus `json:"status,omitempty"`
}

// Check that Broker can be validated, can be defaulted, and has immutable fields.
var _ apis.Validatable = (*Broker)(nil)
var _ apis.Defaultable = (*Broker)(nil)
var _ apis.Immutable = (*Broker)(nil)
var _ runtime.Object = (*Broker)(nil)
var _ webhook.GenericCRD = (*Broker)(nil)

type BrokerSpec struct {
	// TODO By enabling the status subresource metadata.generation should increment
	// thus making this property obsolete.
	//
	// We should be able to drop this property with a CRD conversion webhook
	// in the future
	//
	// +optional
	DeprecatedGeneration int64 `json:"generation,omitempty"`

	ChannelTemplate       *ChannelSpec      `json:"channelTemplate,omitempty"`
}

var brokerCondSet = duckv1alpha1.NewLivingConditionSet(BrokerConditionIngress, BrokerConditionChannel, BrokerConditionFilter, BrokerConditionAddressable)

// BrokerStatus represents the current state of a Broker.
type BrokerStatus struct {
	// ObservedGeneration is the most recent generation observed for this Broker.
	// It corresponds to the Broker's generation, which is updated on mutation by
	// the API Server.
	// TODO: The above comment is only true once
	// https://github.com/kubernetes/kubernetes/issues/58778 is fixed.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Represents the latest available observations of a broker's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions duckv1alpha1.Conditions `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// Broker is Addressable. It currently exposes the endpoint as a
	// fully-qualified DNS name which will distribute traffic over the
	// provided targets from inside the cluster.
	//
	// It generally has the form {broker}-router.{namespace}.svc.{cluster domain name}
	Address duckv1alpha1.Addressable `json:"address,omitempty"`
}

const (
	BrokerConditionReady = duckv1alpha1.ConditionReady

	BrokerConditionIngress duckv1alpha1.ConditionType = "Ingress"

	BrokerConditionChannel duckv1alpha1.ConditionType = "Channel"

	BrokerConditionFilter duckv1alpha1.ConditionType = "Filter"

	BrokerConditionAddressable duckv1alpha1.ConditionType = "Addressable"
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (bs *BrokerStatus) GetCondition(t duckv1alpha1.ConditionType) *duckv1alpha1.Condition {
	return brokerCondSet.Manage(bs).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (bs *BrokerStatus) IsReady() bool {
	return brokerCondSet.Manage(bs).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (bs *BrokerStatus) InitializeConditions() {
	brokerCondSet.Manage(bs).InitializeConditions()
}

func (bs *BrokerStatus) MarkIngressReady() {
	brokerCondSet.Manage(bs).MarkTrue(BrokerConditionIngress)
}

func (bs *BrokerStatus) MarkIngressFailed(err error) {
	brokerCondSet.Manage(bs).MarkFalse(BrokerConditionIngress, "failed", "%v", err)
}

func (bs *BrokerStatus) MarkChannelReady() {
	brokerCondSet.Manage(bs).MarkTrue(BrokerConditionChannel)
}

func (bs *BrokerStatus) MarkChannelFailed(err error) {
	brokerCondSet.Manage(bs).MarkFalse(BrokerConditionChannel, "failed", "%v", err)
}

func (bs *BrokerStatus) MarkFilterReady() {
	brokerCondSet.Manage(bs).MarkTrue(BrokerConditionFilter)
}

// SetAddress makes this Channel addressable by setting the hostname. It also
// sets the ChannelConditionAddressable to true.
func (bs *BrokerStatus) SetAddress(hostname string) {
	bs.Address.Hostname = hostname
	if hostname != "" {
		chanCondSet.Manage(bs).MarkTrue(BrokerConditionAddressable)
	} else {
		chanCondSet.Manage(bs).MarkFalse(BrokerConditionAddressable, "emptyHostname", "hostname is the empty string")
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// BrokerList is a collection of Brokers.
type BrokerList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Broker `json:"items"`
}
