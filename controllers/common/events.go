package common

const (
	// Machine readable event "Reason" fields, used for determining synchronization state.
	EventSynchronized               = "Synchronized"
	EventFailedSynchronization      = "FailedSynchronization"
	EventCreatedInDigDir            = "CreatedInDigDir"
	EventUpdatedInDigDir            = "UpdatedInDigDir"
	EventRotatedInDigDir            = "RotatedInDigDir"
	EventActivatedScopeInDigDir     = "ActivatedScopeInDigDir"
	EventDeactivatedScopeInDigDir   = "DeactivatedScopeInDigDir"
	EventCreatedScopeInDigDir       = "CreatedScopeInDigDir"
	EventUpdatedScopeInDigDir       = "UpdatedScopeInDigDir"
	EventUpdatedACLForScopeInDigDir = "UpdatedACLForScopeInDigDir"
)
