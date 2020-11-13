package common

import (
	"context"
	"github.com/nais/digdirator/api/v1"
	"github.com/nais/digdirator/pkg/digdir"
	log "github.com/sirupsen/logrus"
)

type Transaction struct {
	Ctx          context.Context
	Instance     v1.Instance
	Logger       *log.Entry
	DigdirClient *digdir.Client
}

func NewTransaction(
	ctx context.Context,
	instance v1.Instance,
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
