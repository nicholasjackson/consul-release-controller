package handlers

import (
	"context"
	"net/http"

	"github.com/hashicorp/go-hclog"
	kwhhttp "github.com/slok/kubewebhook/v2/pkg/http"
	kwhlog "github.com/slok/kubewebhook/v2/pkg/log"
	kwhmodel "github.com/slok/kubewebhook/v2/pkg/model"
	kwhmutating "github.com/slok/kubewebhook/v2/pkg/webhook/mutating"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type K8sWebhook struct {
	log hclog.Logger
}

func NewK8sWebhook(l hclog.Logger) (*K8sWebhook, error) {
	return &K8sWebhook{log: l}, nil
}

func (k *K8sWebhook) Mutating() func(w http.ResponseWriter, r *http.Request) {
	mt := kwhmutating.MutatorFunc(func(_ context.Context, _ *kwhmodel.AdmissionReview, obj metav1.Object) (*kwhmutating.MutatorResult, error) {
		deployment, ok := obj.(*appsv1.Deployment)
		if !ok {
			return &kwhmutating.MutatorResult{}, nil
		}

		// Mutate our object with the required annotations.
		if deployment.Annotations == nil {
			deployment.Annotations = make(map[string]string)
		}

		deployment.Annotations["mutated"] = "true"
		deployment.Annotations["mutator"] = "deployment-annotate"

		return &kwhmutating.MutatorResult{MutatedObject: deployment}, nil
	})

	// Create webhook.
	wh, _ := kwhmutating.NewWebhook(kwhmutating.WebhookConfig{
		ID:      "deployment-annotate",
		Mutator: mt,
		Logger:  &wrappedLogger{log: k.log},
	})

	// Get HTTP handler from webhook.
	whHandler, _ := kwhhttp.HandlerFor(
		kwhhttp.HandlerConfig{
			Webhook: wh,
			Logger:  &wrappedLogger{log: k.log},
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
