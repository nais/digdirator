package common

import (
	"context"

	"github.com/nais/digdirator/pkg/clients"
)

type Transaction struct {
	Ctx      context.Context
	Instance clients.Instance
}

func NewTransaction(ctx context.Context, instance clients.Instance) *Transaction {
	return &Transaction{
		Ctx:      ctx,
		Instance: instance,
	}
}
