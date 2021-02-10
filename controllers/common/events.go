package common

const (
	// Machine readable event "Reason" fields, used for determining synchronization state.
	EventSynchronized          = "Synchronized"
	EventFailedSynchronization = "FailedSynchronization"
	EventFailedStatusUpdate    = "FailedStatusUpdate"
	EventAddedFinalizer        = "AddedFinalizer"
	EventDeletedFinalizer      = "DeletedFinalizer"
	EventCreatedInDigDir       = "CreatedInDigDir"
	EventUpdatedInDigDir       = "UpdatedInDigDir"
	EventRotatedInDigDir       = "RotatedInDigDir"
	EventDeletedInDigDir       = "DeletedInDigDir"
	EventRetrying              = "Retrying"
)
