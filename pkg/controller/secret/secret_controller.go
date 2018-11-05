package secret

import (
	"context"

	log "github.com/sirupsen/logrus"

	certmergev1alpha1 "github.com/prune998/certmerge-operator/pkg/apis/certmerge/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Secret Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSecret{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("secret-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Secret
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileSecret{}

// ReconcileSecret reconciles a Secret object
type ReconcileSecret struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Secret object and makes changes based on the state read
// and what is in the Secret.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSecret) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.WithFields(log.Fields{
		"name":      request.Name,
		"namespace": request.Namespace,
	}).Infof("Reconciling Secret")

	// Fetch the Secret instance
	instance := &corev1.Secret{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// we need to search if this secret is a match for a CertMerge
	// get the list of all CertMerge and parse them...

	// sec will hold the Secret List we find
	cml := &certmergev1alpha1.CertMergeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CertMertgeList",
			APIVersion: "certmerge.lecentre.net/v1alpha1",
		},
	}
	listOps := &client.ListOptions{}
	// search
	err = r.client.List(context.TODO(), listOps, cml)
	if err != nil {
		return reconcile.Result{}, err
	}
	for _, cm := range cml.Items {
		if secretInCertMergeList(&cm, instance) || secretInCertMergeLabels(&cm, instance) {
			// trigger the CertMerge Reconcile
			// for the moment, update the status of the CertMerge so it triggers the reconcile
			cm.Status.UpToDate = false

			log.WithFields(log.Fields{
				"name":      request.Name,
				"namespace": request.Namespace,
			}).Infof("need to reconcile CertMerge %s/%s", cm.Namespace, cm.Name)

			// err := r.client.Update(context.TODO(), cm)
			// if err != nil {
			// 	log.WithFields(log.Fields{
			// 		"name":      request.Name,
			// 		"namespace": request.Namespace,
			// 	}).Errorf("Error updating CertMerge %s/%s - %v\n", cm.Namespace, cm.Name, err)
			// 	return reconcile.Result{}, err
			// }
		}
	}
	return reconcile.Result{}, nil
}

func secretInCertMergeList(certmerge *certmergev1alpha1.CertMerge, secret *corev1.Secret) bool {
	// check if secret name is explicitely listed
	for _, sd := range certmerge.Spec.SecretList {
		if sd.Name == secret.Name && sd.Namespace == secret.Namespace {
			return true
		}
	}
	return false
}

func secretInCertMergeLabels(certmerge *certmergev1alpha1.CertMerge, secret *corev1.Secret) bool {
	// check if secret labels match a CertMerge Selector
	for _, se := range certmerge.Spec.Selector {
		isOk := false
		if se.Namespace == secret.Namespace {
			for key, val := range se.LabelSelector.MatchLabels {
				if value, ok := secret.Labels[key]; ok {
					if value == val {
						isOk = true
					}
				}
			}
		}
		if isOk {
			return true
		}
	}
	return false

}
