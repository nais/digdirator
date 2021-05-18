package common

const (
	// Machine readable event "Reason" fields, used for determining synchronization state.
	EventSynchronized               = "Synchronized"
	EventFailedSynchronization      = "FailedSynchronization"
	EventFailedStatusUpdate         = "FailedStatusUpdate"
	EventAddedFinalizer             = "AddedFinalizer"
	EventDeletedFinalizer           = "DeletedFinalizer"
	EventCreatedInDigDir            = "CreatedInDigDir"
	EventUpdatedInDigDir            = "UpdatedInDigDir"
	EventRotatedInDigDir            = "RotatedInDigDir"
	EventDeletedInDigDir            = "DeletedInDigDir"
	EventSkipped                    = "Skipped"
	EventRetrying                   = "Retrying"
	EventDeletedScopeInDigDir       = "DeletedScopeInDigDir"
	EventActivatedScopeInDigDir     = "ActivatedScopeInDigDir"
	EventCreatedScopeInDigDir       = "CreatedScopeInDigDir"
	EventUpdatedScopeInDigDir       = "UpdatedScopeInDigDir"
	EventUpdatedACLForScopeInDigDir = "UpdatedACLForScopeInDigDir"
)
