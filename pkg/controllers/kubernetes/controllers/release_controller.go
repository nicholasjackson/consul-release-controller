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

package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	"github.com/nicholasjackson/consul-release-controller/pkg/plugins/interfaces"

	consulreleasecontrollerv1 "github.com/nicholasjackson/consul-release-controller/pkg/controllers/kubernetes/api/v1"
)

const (
	deploymentField = ".spec.runtime.config.deployment"
	finalizerName   = "release.finalizers.consul-release-controller"
)

// ReleaseReconciler reconciles a Release object
type ReleaseReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Provider interfaces.Provider
}

//+kubebuilder:rbac:groups=consul-release-controller.nicholasjackson.io,resources=releases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=consul-release-controller.nicholasjackson.io,resources=releases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=consul-release-controller.nicholasjackson.io,resources=releases/finalizers,verbs=update

// Add the RBAC for the linked deployment
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Release object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// TODO(user): your logic here
	logger.Info("Reconcile running", "name", req.NamespacedName)

	rc := &consulreleasecontrollerv1.Release{}
	if err := r.Get(ctx, req.NamespacedName, rc); err != nil {
		logger.Info("Unable to fetch Release, most likely deleted")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if rc.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !ContainsString(rc.ObjectMeta.Finalizers, finalizerName) {
			rc.ObjectMeta.Finalizers = append(rc.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(context.Background(), rc); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		}

	} else {
		// The object is being deleted
		if ContainsString(rc.ObjectMeta.Finalizers, finalizerName) {

			// Delete the release
			r.deleteRelease(rc, logger)

			// remove our finalizer from the list and update it.
			rc.ObjectMeta.Finalizers = RemoveString(rc.ObjectMeta.Finalizers, finalizerName)

			if err := r.Update(context.Background(), rc); err != nil {
				logger.Error(err, "Unable to delete resolver", "name", rc.Name)

				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// update the release and start any required events
	err := r.updateRelease(rc, logger)

	return ctrl.Result{}, err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&consulreleasecontrollerv1.Release{},
		deploymentField,
		func(rawObj client.Object) []string {
			// Extract the Deployment name from the ConfigDeployment Spec, if one is provided
			release := rawObj.(*consulreleasecontrollerv1.Release)
			if release.Spec.Runtime.Config.Deployment == "" {
				return nil
			}

			return []string{release.Spec.Runtime.Config.Deployment}
		},
	)

	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&consulreleasecontrollerv1.Release{}).
		Complete(r)
}

func (r *ReleaseReconciler) updateRelease(rc *consulreleasecontrollerv1.Release, log logr.Logger) error {
	// check to see if the release exists
	rel, err := r.Provider.GetDataStore().GetRelease(rc.Name)
	if err == nil {
		log.Info("Found existing release, updating", "name", rc.Name)

		// check the version if the controller restarts kubernetes will re-submit the release, in this
		// instance we do not want to reset the state
		if rel.Version == fmt.Sprintf("%d", rc.ObjectMeta.Generation) {
			log.Info("Ignoring release, not updated", "name", rc.Name, "version", rel.Version)
			return nil
		}

		// check the state of the release, only allow an update when idle or failed
		if rel.CurrentState() != interfaces.StateIdle && rel.CurrentState() != interfaces.StateFail {
			log.Info("Release active, not updating", "name", rc.Name, "version", rel.Version)
			return fmt.Errorf("unable to update release, release currently active")
		}
	}

	log.Info("Upsert release", "name", rc.Name)

	rm := rc.ConvertToModel()

	// Update the store
	err = r.Provider.GetDataStore().UpsertRelease(rm)
	if err != nil {
		log.Error(err, "Unable to create new release", "name", rc.Name)
		return err
	}

	// clear any existing state
	r.Provider.DeleteStateMachine(rm)

	// create a new statemachine
	sm, err := r.Provider.GetStateMachine(rm)
	if err != nil {
		log.Error(err, "Unable to get statemachine", "name", rc.Name)
		return err
	}

	// Configure the release
	err = sm.Configure()
	if err != nil {
		log.Error(err, "Unable to configure new release", "name", rc.Name)
		return err
	}

	if sm.CurrentState() == interfaces.StateIdle {
		log.Info("New deployment, trigger event", "name", rc.Name)
		sm.Deploy()
	}

	return nil
}

func (r *ReleaseReconciler) deleteRelease(rc *consulreleasecontrollerv1.Release, log logr.Logger) error {
	rm, err := r.Provider.GetDataStore().GetRelease(rc.Name)
	if err == interfaces.ReleaseNotFound {
		log.Info("Unable to delete release, not found", "name", rc.Name)

		// nothing to do
		return nil
	}

	if err != nil {
		log.Error(err, "Unable to delete release", "name", rc.Name)

		return err
	}

	// get the statemachine
	sm, err := r.Provider.GetStateMachine(rm)
	if err != nil {
		log.Error(err, "Unable to get statemachine", "name", rc.Name)
		return err
	}

	// cleanup
	err = sm.Destroy()
	if err != nil {
		log.Error(err, "Unable to delete release", "name", rc.Name)

		return err
	}

	// wait until finished
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()

		for {
			if ctx.Err() != nil {
				log.Error(fmt.Errorf("Timeout waiting to destroy release"), "name", rc.Name)
				return
			}

			// destroy is complete
			if sm.CurrentState() == interfaces.StateIdle {
				log.Info("Destroy complete, removing release", "name", rc.Name)
				r.Provider.DeleteStateMachine(rm)
				if err != nil {
					log.Error(err, "Unable to delete statemachine", "name", rc.Name)
				}

				err = r.Provider.GetDataStore().DeleteRelease(rm.Name)
				if err != nil {
					log.Error(err, "Unable to delete release", "name", rc.Name)
				}

				return
			}

			if sm.CurrentState() == interfaces.StateFail {
				log.Error(fmt.Errorf("Unable to destroy release"), "name", rc.Name)
				return
			}

			log.Info("Waiting for destroy to complete", "current_state", sm.CurrentState())
			time.Sleep(2 * time.Second)
		}

	}()

	return nil

}
