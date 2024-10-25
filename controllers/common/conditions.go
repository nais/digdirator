package common

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionType string

const (
	ConditionTypeReady                         ConditionType = "Ready"
	ConditionTypeError                         ConditionType = "Error"
	ConditionTypeInvalidConsumedScopes         ConditionType = "InvalidConsumedScopes"
	ConditionTypeInvalidExposedScopesConsumers ConditionType = "InvalidExposedScopesConsumers"
)

type ConditionReason string

const (
	ConditionReasonFailed       ConditionReason = "Failed"
	ConditionReasonProcessing   ConditionReason = "Processing"
	ConditionReasonSynchronized ConditionReason = "Synchronized"
	ConditionReasonValidated    ConditionReason = "Validated"
)

func ReadyCondition(status metav1.ConditionStatus, reason ConditionReason, message string, generation int64) metav1.Condition {
	return metav1.Condition{
		Type:               string(ConditionTypeReady),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: generation,
	}
}

func ErrorCondition(status metav1.ConditionStatus, reason ConditionReason, message string, generation int64) metav1.Condition {
	return metav1.Condition{
		Type:               string(ConditionTypeError),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: generation,
	}
}

func InvalidConsumedScopesCondition(status metav1.ConditionStatus, reason ConditionReason, message string, generation int64) metav1.Condition {
	return metav1.Condition{
		Type:               string(ConditionTypeInvalidConsumedScopes),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: generation,
	}
}

func InvalidExposedScopesConsumersCondition(status metav1.ConditionStatus, reason ConditionReason, message string, generation int64) metav1.Condition {
	return metav1.Condition{
		Type:               string(ConditionTypeInvalidExposedScopesConsumers),
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: generation,
	}
}

func HasRetryableStatusCondition(conditions *[]metav1.Condition) bool {
	if conditions == nil {
		return false
	}

	isError := IsStatusConditionTrue(conditions, ConditionTypeError)
	isInvalidConsumedScopes := IsStatusConditionTrue(conditions, ConditionTypeInvalidConsumedScopes)

	return isError || isInvalidConsumedScopes
}

func IsStatusConditionTrue(conditions *[]metav1.Condition, conditionType ConditionType) bool {
	if conditions == nil {
		return false
	}

	return meta.IsStatusConditionTrue(*conditions, string(conditionType))
}
