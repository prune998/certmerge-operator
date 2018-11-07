package certmerge

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/google/go-cmp/cmp"
	certmergev1alpha1 "github.com/prune998/certmerge-operator/pkg/apis/certmerge/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new CertMerge Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	r := newReconciler(mgr)
	return add(mgr, r, r.SecretTriggerCertMerge)
}

// newReconciler returns a new reconcile.Reconciler
// orifginaly return reconcile.Reconciler
func newReconciler(mgr manager.Manager) *ReconcileCertMerge {
	return &ReconcileCertMerge{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, mapFn handler.ToRequestsFunc) error {
	// Create a new controller
	c, err := controller.New("certmerge-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CertMerge
	err = c.Watch(&source.Kind{Type: &certmergev1alpha1.CertMerge{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// This will trigger the Reconcile if the Merged Secret is modified
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &certmergev1alpha1.CertMerge{},
	})
	if err != nil {
		return err
	}

	// This predicate deduplicate the watch trigger if no data is modified inside the secret
	// if the Secret is Deleted, don't send the delete event as we can trigger it from the update
	var p predicate.Predicate
	p = predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			log.WithFields(log.Fields{
				"event": e,
			}).Debugf("Update Predicate event")
			// This update is in fact a Delete event, process it
			if e.MetaNew.GetDeletionGracePeriodSeconds() != nil {
				return true
			}

			// if old and new data is the same, don't reconcile
			newObj := e.ObjectNew.DeepCopyObject().(*corev1.Secret)
			oldObj := e.ObjectOld.DeepCopyObject().(*corev1.Secret)
			if cmp.Equal(newObj.Data, oldObj.Data) {
				return false
			}

			return true
		},
		// don't process any Delete event as we catch them in Update
		DeleteFunc: func(e event.DeleteEvent) bool {
			log.WithFields(log.Fields{
				"event": e,
			}).Debugf("Delete Predicate event")
			return false
		},
	}

	// Watch for Secret change and process them through the SecretTriggerCertMerge function
	// This watch enables us to reconcile a CertMerge when a concerned Secret is changed (create/update/delete)
	err = c.Watch(
		&source.Kind{Type: &corev1.Secret{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn,
		},
		p,
	)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCertMerge{}

// ReconcileCertMerge reconciles a CertMerge object
type ReconcileCertMerge struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// SecretTriggerCertMerge check if a Secret is concerned by a CertMerge and enque the CertMerge for Reconcile
func (r *ReconcileCertMerge) SecretTriggerCertMerge(o handler.MapObject) []reconcile.Request {
	result := []reconcile.Request{}

	// drop secrets if the Operator is the Onwer
	// we don't support merging an already merged secret
	for _, owner := range o.Meta.GetOwnerReferences() {
		if owner.Kind == "CertMerge" {
			log.WithFields(log.Fields{
				"secret":    o.Meta.GetName(),
				"namespace": o.Meta.GetNamespace(),
			}).Infof("Secret is managed by CertMerge, dropping event")
			return nil
		}
	}

	log.WithFields(log.Fields{
		"secret":    o.Meta.GetName(),
		"namespace": o.Meta.GetNamespace(),
	}).Infof("Secret update triggered, reconciling CertMerge CR")

	// Fetch the triggered Secret Data
	instance := &corev1.Secret{}
	key := client.ObjectKey{Namespace: o.Meta.GetNamespace(), Name: o.Meta.GetName()}
	err := r.client.Get(context.TODO(), key, instance)
	if errors.IsNotFound(err) {
		// in this case it's a deleted secret, keep going
		// as we don't have the secret anymore, we need to trigger all the CertMerge CR
		// for the moment we do nothing and keep outdated secrets merged
		log.WithFields(log.Fields{
			"secret":    o.Meta.GetName(),
			"namespace": o.Meta.GetNamespace(),
		}).Infof("Secret not found, not reconciling any CertMerge CR")
		return nil
	}
	if err != nil {
		log.Errorf("Error retrieving Secret %v from store: %v", key, err)
		return nil
	}

	// sec will hold the CertMerge CR List we find
	cml := &certmergev1alpha1.CertMergeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CertMertgeList",
			APIVersion: "certmerge.lecentre.net/v1alpha1",
		},
	}

	// Get all CertMerges
	err = r.client.List(context.TODO(), nil, cml)
	if err != nil {
		return result
	}

	// parse each CertMerge CR and reconcile them if needed
	for _, cm := range cml.Items {
		if secretInCertMergeList(&cm, instance) || secretInCertMergeLabels(&cm, instance) {
			// trigger the CertMerge Reconcile
			result = append(result, reconcile.Request{
				NamespacedName: client.ObjectKey{Namespace: cm.Namespace, Name: cm.Name}})

			log.Infof("CertMerge %s/%s added to Reconcile List", cm.Namespace, cm.Name)
		}
	}
	return result
}

// secretInCertMergeList returns true if the provided Secret Name is included in the CertMerge CR SecretList
func secretInCertMergeList(certmerge *certmergev1alpha1.CertMerge, secret *corev1.Secret) bool {
	// check if secret name is explicitely listed
	for _, sd := range certmerge.Spec.SecretList {
		if sd.Name == secret.Name && sd.Namespace == secret.Namespace {
			return true
		}
	}
	return false
}

// secretInCertMergeLabels returns true if the provided Secret Labels match the Selector of the CertMerge CR
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

// Reconcile reads that state of the cluster for a CertMerge object and makes changes based on the state read
// and what is in the CertMerge.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCertMerge) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Infof("Reconciling CertMerge %s/%s", request.Namespace, request.Name)

	// Fetch the CertMerge instance
	instance := &certmergev1alpha1.CertMerge{}
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

	// Define a new Secret object
	secret := newSecretForCR(instance)

	// Set CertMerge instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, secret, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// create the DATA for the new secret based on the CertMerge request
	certData := make(map[string][]byte)

	// build the Cert Data from the provided secret List
	if len(instance.Spec.SecretList) > 0 {
		for _, sec := range instance.Spec.SecretList {
			secContent, err := r.searchSecretByName(sec.Name, sec.Namespace)
			if err != nil {
				log.WithFields(log.Fields{
					"certmerge": instance.Name,
					"namespace": instance.Namespace,
				}).Errorf("requested certificate target not found, skipping (%s/%s) - %v", sec.Namespace, sec.Name, err)
				continue
			}

			// search for key
			if secContent.Type != corev1.SecretTypeTLS {
				log.WithFields(log.Fields{
					"certmerge": instance.Name,
					"namespace": instance.Namespace,
				}).Infof("certificate %s/%s is not of TLS type, skipping", secContent.Namespace, secContent.Name)
				continue
			}

			log.WithFields(log.Fields{
				"certmerge": instance.Name,
				"namespace": instance.Namespace,
			}).Infof("adding certificate %s/%s to merge list", secContent.Namespace, secContent.Name)

			certData[sec.Name+".crt"] = secContent.Data["tls.crt"]
			certData[sec.Name+".key"] = secContent.Data["tls.key"]
		}
	}

	// building the Cert Data from the provided Labels
	if len(instance.Spec.Selector) > 0 {
		for _, sec := range instance.Spec.Selector {
			// do nothing if labels search is empty to avoid getting all the secrets of the cluster
			if len(sec.LabelSelector.MatchLabels) == 0 {
				log.WithFields(log.Fields{
					"certmerge": instance.Name,
					"namespace": instance.Namespace,
					"labels":    sec.LabelSelector.MatchLabels,
				}).Errorf("no labels defined in Selector, nothing to merge")
				continue
			}

			// search for Secrets using the API
			secContent, err := r.searchSecretByLabel(sec.LabelSelector.MatchLabels, sec.Namespace)
			if err != nil {
				log.WithFields(log.Fields{
					"certmerge": instance.Name,
					"namespace": instance.Namespace,
					"labels":    sec.LabelSelector.MatchLabels,
				}).Errorf("no certificates matching labels %v in %s, skipping - %v", sec.LabelSelector.MatchLabels, sec.Namespace, err)
				continue
			}

			// add valid secret's data to the Merged Secret
			log.WithFields(log.Fields{
				"certmerge": instance.Name,
				"namespace": instance.Namespace,
				"labels":    sec.LabelSelector.MatchLabels,
			}).Infof("found %d certificates to merge in %s/%s", len(secContent.Items), instance.Spec.SecretNamespace, instance.Spec.SecretName)

			// add valid secret's data to the Merged Secret
			for _, secCert := range secContent.Items {
				log.WithFields(log.Fields{
					"certmerge": instance.Name,
					"namespace": instance.Namespace,
				}).Infof("adding cert %s/%s to %s/%s", secCert.Namespace, secCert.Name, instance.Spec.SecretNamespace, instance.Spec.SecretName)
				certData[secCert.Name+".crt"] = secCert.Data["tls.crt"]
				certData[secCert.Name+".key"] = secCert.Data["tls.key"]
			}
		}
	}

	// add the Data to the secret
	secret.Data = certData

	// Check if this Secret already exists
	found := &corev1.Secret{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.WithFields(log.Fields{
			"certmerge": instance.Name,
			"namespace": instance.Namespace,
		}).Infof("Creating a new Secret %s/%s\n", secret.Namespace, secret.Name)

		err = r.client.Create(context.TODO(), secret)
		if err != nil {
			log.WithFields(log.Fields{
				"certmerge": instance.Name,
				"namespace": instance.Namespace,
			}).Errorf("Error creating new Secret %s/%s - %v\n", secret.Namespace, secret.Name, err)
			return reconcile.Result{}, err
		}

		// Secret created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		// unknown error
		return reconcile.Result{}, err
	}

	// if the Secret already exist, Update it
	log.WithFields(log.Fields{
		"certmerge": instance.Name,
		"namespace": instance.Namespace,
	}).Infof("Updating Secret %s/%s\n", secret.Namespace, secret.Name)

	err = r.client.Update(context.TODO(), secret)
	if err != nil {
		log.WithFields(log.Fields{
			"certmerge": instance.Name,
			"namespace": instance.Namespace,
		}).Errorf("Error updating Secret %s/%s - %v\n", secret.Namespace, secret.Name, err)
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// newSecretForCR returns an empty secret for holding the secrets merge
func newSecretForCR(cr *certmergev1alpha1.CertMerge) *corev1.Secret {
	labels := map[string]string{
		"certmerge": cr.Name,
		"creator":   "certmerge-operator",
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Spec.SecretName,
			Namespace: cr.Spec.SecretNamespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
	}
}

// search on secret by it's namespace:name
func (r *ReconcileCertMerge) searchSecretByName(name, namespace string) (*corev1.Secret, error) {
	sec := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, sec)
	if err != nil {
		return nil, err
	}
	return sec, nil
}

// search all secrets with the supplied labels
func (r *ReconcileCertMerge) searchSecretByLabel(ls map[string]string, namespace string) (*corev1.SecretList, error) {
	// create the Search options
	labelSelector := labels.SelectorFromSet(ls)
	listOps := &client.ListOptions{LabelSelector: labelSelector}

	// sec will hold the Secret List we find
	sec := &corev1.SecretList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
	}

	// search
	err := r.client.List(context.TODO(), listOps, sec)
	if err != nil {
		return nil, err
	}

	return sec, nil
}
