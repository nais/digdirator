package reconciler

import (
	"github.com/nais/digdirator/pkg/config"
	"gopkg.in/square/go-jose.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const RequeueInterval = 10 * time.Second

type Reconciler struct {
	Client client.Client
	Reader client.Reader
	Scheme *runtime.Scheme

	Recorder   record.EventRecorder
	Config     *config.Config
	Signer     jose.Signer
	HttpClient *http.Client
}
