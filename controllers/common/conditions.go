package common

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	conditionTypeReady = "Ready"
	conditionTypeError = "Error"
)

type ConditionReason string

const (
	ConditionReasonFailed       ConditionReason = "Failed"
	ConditionReasonProcessing                   = "Processing"
	ConditionReasonSynchronized                 = "Synchronized"
)

func readyCondition(status metav1.ConditionStatus, reason ConditionReason, message string, generation int64) metav1.Condition {
	return metav1.Condition{
		Type:               conditionTypeReady,
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: generation,
	}
}

func errorCondition(status metav1.ConditionStatus, reason ConditionReason, message string, generation int64) metav1.Condition {
	return metav1.Condition{
		Type:               conditionTypeError,
		Status:             status,
		Reason:             string(reason),
		Message:            message,
		ObservedGeneration: generation,
	}
}
