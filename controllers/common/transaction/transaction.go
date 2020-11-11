package transaction

import (
	"context"
	"github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/digdir"
	"github.com/nais/digdirator/pkg/secrets"
	log "github.com/sirupsen/logrus"
)

type Transaction struct {
	Ctx            context.Context
	Instance       v1.Instance
	Logger         *log.Entry
	ManagedSecrets *secrets.Lists
	DigdirClient   *digdir.Client
	SecretsClient  secrets.Client
}

func NewTransaction(
	ctx context.Context,
	instance v1.Instance,
	logger *log.Entry,
	managedSecrets *secrets.Lists,
	digdirClient *digdir.Client,
	secretsClient secrets.Client,
) Transaction {
	return Transaction{
		Ctx:            ctx,
		Instance:       instance,
		Logger:         logger,
		ManagedSecrets: managedSecrets,
		DigdirClient:   digdirClient,
		SecretsClient:  secretsClient,
	}
}
