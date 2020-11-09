package transaction

import (
	"context"
	"github.com/nais/digdirator/controllers/common"
	"github.com/nais/digdirator/pkg/digdir"
	"github.com/nais/digdirator/pkg/secrets"
	log "github.com/sirupsen/logrus"
)

type Transaction struct {
	Ctx            context.Context
	Instance       common.Instance
	Logger         *log.Entry
	ManagedSecrets *secrets.Lists
	DigdirClient   *digdir.Client
	SecretsClient  secrets.Client
}

func NewTransaction(
	ctx context.Context,
	instance common.Instance,
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
