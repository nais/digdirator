package common

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/nais/digdirator/pkg/clients"
	"github.com/nais/digdirator/pkg/config"
	"github.com/nais/digdirator/pkg/digdir"
)

type Transaction struct {
	Ctx          context.Context
	Instance     clients.Instance
	Logger       *log.Entry
	DigdirClient *digdir.Client
	Config       *config.Config
}

func NewTransaction(
	ctx context.Context,
	instance clients.Instance,
	logger *log.Entry,
	digdirClient *digdir.Client,
	config *config.Config,
) Transaction {
	return Transaction{
		Ctx:          ctx,
		Instance:     instance,
		Logger:       logger,
		DigdirClient: digdirClient,
		Config:       config,
	}
}
