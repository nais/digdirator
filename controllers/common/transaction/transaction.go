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
