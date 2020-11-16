package v1

const (
	// Keys for outputting data to secrets
	IDPortenJwkKey           = "IDPORTEN_CLIENT_JWK"
	IDPortenClientID         = "IDPORTEN_CLIENT_ID"
	IDPortenWellKnownURL     = "IDPORTEN_WELL_KNOWN_URL"
	IDPortenRedirectURI      = "IDPORTEN_REDIRECT_URI"
	MaskinportenJwkKey       = "MASKINPORTEN_CLIENT_JWK"
	MaskinportenClientID     = "MASKINPORTEN_CLIENT_ID"
	MaskinportenWellKnownURL = "MASKINPORTEN_WELL_KNOWN_URL"
	MaskinportenScopes       = "MASKINPORTEN_SCOPES"

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

	AccessTokenLifetimeSeconds = 3600
)
