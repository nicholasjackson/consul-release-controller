/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"os"
	"path"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-hclog"
	admissionController "github.com/nicholasjackson/consul-release-controller/pkg/controllers"
	consulreleasecontrollerv1 "github.com/nicholasjackson/consul-release-controller/pkg/controllers/kubernetes/api/v1"
	"github.com/nicholasjackson/consul-release-controller/pkg/controllers/kubernetes/controllers"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(consulreleasecontrollerv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

type Kubernetes struct {
	mngr        manager.Manager
	ctx         context.Context
	cancel      context.CancelFunc
	log         hclog.Logger
	provider    interfaces.Provider
	tlsCert     string
	tlsKey      string
	webhookPort int
}

func New(p interfaces.Provider, tlsCert, tlsKey string, webhookPort int) *Kubernetes {
	ctx, cancelFunc := context.WithCancel(context.Background())

	return &Kubernetes{
		ctx:         ctx,
		cancel:      cancelFunc,
		log:         p.GetLogger().Named("kubernetes-controller"),
		provider:    p,
		tlsCert:     tlsCert,
		tlsKey:      tlsKey,
		webhookPort: webhookPort,
	}
}

func (k *Kubernetes) Start() error {
	logSink := newSinkLogger(k.log)
	ctrl.SetLogger(logr.New(logSink))

	webhookServer := webhook.Server{}
	webhookServer.CertDir = path.Dir(k.tlsCert)
	webhookServer.CertName = path.Base(k.tlsCert)
	webhookServer.KeyName = path.Base(k.tlsKey)
	webhookServer.Port = k.webhookPort

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     "0",
		HealthProbeBindAddress: "0",
		LeaderElection:         false,
		LeaderElectionID:       "4224bb32.nicholasjackson.io",
		WebhookServer:          &webhookServer,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ReleaseReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Provider: k.provider,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Release")
		return err
	}
	//+kubebuilder:scaffold:builder

	setupLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()

	admissionPlugin := admissionController.NewAdmission(k.provider, k.log.ResetNamed("kubernetes-webhook"))

	setupLog.Info("registering webhooks to the webhook server")
	hookServer.Register(
		"/validate-v1-deployment",
		&webhook.Admission{
			Handler: NewDeploymentAdmission(mgr.GetClient(), admissionPlugin),
		},
	)

	setupLog.Info("Starting Kubernetes controller")
	if err := mgr.Start(k.ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}

	return nil
}

func (k *Kubernetes) Stop() error {
	k.log.Info("Stopping Kubernetes controller")
	k.cancel()
	return nil
}
