package kubernetes

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/nicholasjackson/consul-canary-controller/clients"
	"github.com/nicholasjackson/consul-canary-controller/metrics"
	appmetrics "github.com/nicholasjackson/consul-canary-controller/metrics"
	"github.com/nicholasjackson/consul-canary-controller/plugins"
	"github.com/nicholasjackson/consul-canary-controller/plugins/interfaces"
	"github.com/nicholasjackson/consul-canary-controller/state"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type K8sWebhook struct {
	logger          hclog.Logger
	store           state.Store
	metrics         appmetrics.Metrics
	pluginProviders interfaces.Provider
}

func NewK8sWebhook(l hclog.Logger, m metrics.Metrics, s state.Store, p interfaces.Provider) (*K8sWebhook, error) {
	return &K8sWebhook{logger: l, metrics: m, store: s, pluginProviders: p}, nil
}

func (k *K8sWebhook) Mutating() func(w http.ResponseWriter, r *http.Request) {

	mt := kwhmutating.MutatorFunc(func(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
		deployment, ok := obj.(*appsv1.Deployment)
		if !ok {
			k.logger.Error("Kubernetes webhook called with type that is not a deployment")

			return &kwhmutating.MutatorResult{}, nil
		}

		// check if we have a release that matches the name of this pod
		rel, err := k.store.ListReleases(&state.ListOptions{Runtime: plugins.PluginRuntimeTypeKubernetes})
		if err != nil {
			k.logger.Error("Unable to list releases", "error", err)

			return &kwhmutating.MutatorResult{}, nil
		}

		for _, r := range rel {
			err := r.Build(k.pluginProviders)
			if err != nil {
				k.logger.Error("Unable to build plugin", "error", err)

				return &kwhmutating.MutatorResult{}, nil
			}

			rp := r.RuntimePlugin()
			conf := rp.BaseConfig()

			if conf.Deployment == deployment.Name && conf.Namespace == deployment.ObjectMeta.Namespace {
				// Mutate our object with the required annotations.
				if deployment.Annotations == nil {
					deployment.Annotations = make(map[string]string)
				}

				deployment.Annotations["consul-releaser"] = "true"

				// check if this is the first deploy of this application
				kc, err := clients.NewKubernetes(os.Getenv("KUBECONFIG"))
				if err != nil {
					k.logger.Error("Unable to create a Kubernetes client", "error", err)
					return &kwhmutating.MutatorResult{}, fmt.Errorf("unable to create client: %s", err)
				}

				var status interfaces.RuntimeDeploymentStatus = interfaces.RuntimeDeploymentNotFound

				_, err = kc.GetDeployment(deployment.Name, deployment.Namespace)
				if err == nil {
					status = interfaces.RuntimeDeploymentUpdate
				}

				// trigger the deployment actions for the plugins, this is an async call
				k.logger.Info("Calling plugin deploy for kubernetes deployment", "deployment", deployment.Name, "namespace", deployment.Namespace)
				r.Deploy(status)

				return &kwhmutating.MutatorResult{MutatedObject: deployment}, nil
			}
		}

		return &kwhmutating.MutatorResult{}, nil
	})

	// Create webhook.
	wh, _ := kwhmutating.NewWebhook(kwhmutating.WebhookConfig{
		ID:      "deployment-annotate",
		Mutator: mt,
		Logger:  &wrappedLogger{log: k.logger},
	})

	// Get HTTP handler from webhook.
	whHandler, _ := kwhhttp.HandlerFor(
		kwhhttp.HandlerConfig{
			Webhook: wh,
			Logger:  &wrappedLogger{log: k.logger},
		})

	return whHandler.ServeHTTP
}

type wrappedLogger struct {
	log hclog.Logger
}

func (w *wrappedLogger) Infof(format string, args ...interface{}) {
	w.log.StandardLogger(&hclog.StandardLoggerOptions{ForceLevel: hclog.Info}).Printf(format, args...)
}

func (w *wrappedLogger) Warningf(format string, args ...interface{}) {
	w.log.StandardLogger(&hclog.StandardLoggerOptions{ForceLevel: hclog.Warn}).Printf(format, args...)
}

func (w *wrappedLogger) Errorf(format string, args ...interface{}) {
	w.log.StandardLogger(&hclog.StandardLoggerOptions{ForceLevel: hclog.Error}).Printf(format, args...)
}

func (w *wrappedLogger) Debugf(format string, args ...interface{}) {
	w.log.StandardLogger(&hclog.StandardLoggerOptions{ForceLevel: hclog.Debug}).Printf(format, args...)
}

func (w *wrappedLogger) WithValues(map[string]interface{}) kwhlog.Logger { return w }
func (w *wrappedLogger) WithCtxValues(context.Context) kwhlog.Logger     { return w }
func (w *wrappedLogger) SetValuesOnCtx(parent context.Context, values kwhlog.Kv) context.Context {
	return parent
}
