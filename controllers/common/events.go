package common

const (
	// Machine readable event "Reason" fields, used for determining synchronization state.
	EventSynchronized               = "Synchronized"
	EventFailedSynchronization      = "FailedSynchronization"
	EventFailedStatusUpdate         = "FailedStatusUpdate"
	EventCreatedInDigDir            = "CreatedInDigDir"
	EventUpdatedInDigDir            = "UpdatedInDigDir"
	EventRotatedInDigDir            = "RotatedInDigDir"
	EventSkipped                    = "Skipped"
	EventRetrying                   = "Retrying"
	EventActivatedScopeInDigDir     = "ActivatedScopeInDigDir"
	EventDeactivatedScopeInDigDir   = "DeactivatedScopeInDigDir"
	EventCreatedScopeInDigDir       = "CreatedScopeInDigDir"
	EventUpdatedScopeInDigDir       = "UpdatedScopeInDigDir"
	EventUpdatedACLForScopeInDigDir = "UpdatedACLForScopeInDigDir"
)
