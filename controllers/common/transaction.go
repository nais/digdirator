package common

import (
	"context"
	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/digdir"
	log "github.com/sirupsen/logrus"
)

type Transaction struct {
	Ctx          context.Context
	Instance     clients.Instance
	Logger       *log.Entry
	DigdirClient *digdir.Client
}

func NewTransaction(
	ctx context.Context,
	instance clients.Instance,
	logger *log.Entry,
	digdirClient *digdir.Client,
) Transaction {
	return Transaction{
		Ctx:          ctx,
		Instance:     instance,
		Logger:       logger,
		DigdirClient: digdirClient,
	}
}
