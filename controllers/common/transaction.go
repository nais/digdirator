package common

import (
	"context"

	"github.com/nais/digdirator/pkg/clients"
	log "github.com/sirupsen/logrus"
)

type Transaction struct {
	Ctx      context.Context
	Instance clients.Instance
	Logger   *log.Entry
}

func NewTransaction(ctx context.Context, instance clients.Instance, logger *log.Entry) *Transaction {
	return &Transaction{
		Ctx:      ctx,
		Instance: instance,
		Logger:   logger,
	}
}
