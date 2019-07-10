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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"knative.dev/pkg/apis"
)

var triggerCondSet = apis.NewLivingConditionSet(TriggerConditionBroker, TriggerConditionSubscribed)

const (
	// TriggerConditionReady has status True when all subconditions below have been set to True.
	TriggerConditionReady = apis.ConditionReady

	TriggerConditionBroker apis.ConditionType = "Broker"

	TriggerConditionSubscribed apis.ConditionType = "Subscribed"

	// TriggerAnyFilter Constant to represent that we should allow anything.
	TriggerAnyFilter = ""
)

// GetCondition returns the condition currently associated with the given type, or nil.
func (ts *TriggerStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return triggerCondSet.Manage(ts).GetCondition(t)
}

// IsReady returns true if the resource is ready overall.
func (ts *TriggerStatus) IsReady() bool {
	return triggerCondSet.Manage(ts).IsHappy()
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (ts *TriggerStatus) InitializeConditions() {
	triggerCondSet.Manage(ts).InitializeConditions()
}

func (ts *TriggerStatus) PropagateBrokerStatus(bs *BrokerStatus) {
	if bs.IsReady() {
		triggerCondSet.Manage(ts).MarkTrue(TriggerConditionBroker)
	} else {
		msg := "nil"
		if bc := brokerCondSet.Manage(bs).GetCondition(BrokerConditionReady); bc != nil {
			msg = bc.Message
		}
		ts.MarkBrokerFailed("BrokerNotReady", "Broker is not ready: %s", msg)
	}
}

func (ts *TriggerStatus) MarkBrokerFailed(reason, messageFormat string, messageA ...interface{}) {
	triggerCondSet.Manage(ts).MarkFalse(TriggerConditionBroker, reason, messageFormat, messageA...)
}

func (ts *TriggerStatus) PropagateSubscriptionStatus(ss *SubscriptionStatus) {
	if ss.IsReady() {
		triggerCondSet.Manage(ts).MarkTrue(TriggerConditionSubscribed)
	} else {
		msg := "nil"
		if sc := subCondSet.Manage(ss).GetCondition(SubscriptionConditionReady); sc != nil {
			msg = sc.Message
		}
		ts.MarkNotSubscribed("SubscriptionNotReady", "Subscription is not ready: %s", msg)
	}
}

func (ts *TriggerStatus) MarkNotSubscribed(reason, messageFormat string, messageA ...interface{}) {
	triggerCondSet.Manage(ts).MarkFalse(TriggerConditionSubscribed, reason, messageFormat, messageA...)
}

func (ts *TriggerStatus) SetAddress(url *apis.URL) {
	if url != nil {
		ts.Address.URL = url
		brokerCondSet.Manage(ts).MarkTrue(BrokerConditionAddressable)
	} else {
		ts.Address.URL = nil
		brokerCondSet.Manage(ts).MarkFalse(BrokerConditionAddressable, "emptyHostname", "hostname is the empty string")
	}
}

func (ts *TriggerStatus) extendImporters(index int) {
	if index >= len(ts.Importers) {
		ts.Importers = append(ts.Importers, make([]TriggerImporterStatus, index-len(ts.Importers)+1)...)
	}
}

func (ts *TriggerStatus) MarkImporter(index int, importer v1.ObjectReference) {
	ts.extendImporters(index)
	ts.Importers[index].Ref = &importer
}

func (ts *TriggerStatus) MarkImportersFailed(err error) {
	// something
}

func (ts *TriggerStatus) MarkImporterRef(index int, importer corev1.ObjectReference) {
	ts.extendImporters(index)
	ts.Importers[index].Ref = &importer
}

func (ts *TriggerStatus) PropagateImporterStatus(index int, status runtime.RawExtension) {
	ts.extendImporters(index)
	ts.Importers[index].Status = &status
}

func (ts *TriggerStatus) MarkImporterError(index int, err error) {
	ts.extendImporters(index)
	ts.Importers[index].Error = err.Error()
}

func (ts *TriggerStatus) MarkBroadcast() {
	ts.Type = TriggerTypeBroadcast
}

func (ts *TriggerStatus) MarkTargeted() {
	ts.Type = TriggerTypeTargeted
}
