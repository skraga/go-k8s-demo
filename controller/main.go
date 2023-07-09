package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// Build the Kubernetes client configuration
	config, err := clientcmd.BuildConfigFromFlags("", getKubeConfigPath())
	if err != nil {
		klog.Fatal(err)
	}

	// Create a new clientset based on the configuration
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	// Initialize a shared informer to watch for deployment events
	sharedInformer := newSharedInformer(clientset)

	// Create a work queue with rate limiting
	workQueue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	stopCh := make(chan struct{})
	defer close(stopCh)

	// Start the shared informer and controller goroutines
	go sharedInformer.Run(stopCh)
	go runController(workQueue, sharedInformer, clientset)

	// Wait for a termination signal
	<-stopCh
}

// newSharedInformer creates a new shared informer for deployments.
func newSharedInformer(clientset *kubernetes.Clientset) cache.SharedIndexInformer {
	// Create a list watcher for deployments in all namespaces
	listWatcher := cache.NewListWatchFromClient(clientset.AppsV1().RESTClient(), "deployments", corev1.NamespaceAll, fields.Everything())

	// Create a shared informer for deployments
	sharedInformer := cache.NewSharedIndexInformer(
		listWatcher,
		&appsv1.Deployment{},
		0,
		cache.Indexers{},
	)

	// Add event handlers to the informer
	_, _ = sharedInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// Handle the addition of a new deployment
		AddFunc: func(obj interface{}) {
			deployment := obj.(*appsv1.Deployment)
			handleDeploymentEvent(deployment, clientset)
		},
		// Handle the update of an existing deployment
		UpdateFunc: func(oldObj, newObj interface{}) {
			newDeployment := newObj.(*appsv1.Deployment)
			handleDeploymentEvent(newDeployment, clientset)
		},
	})

	return sharedInformer
}

// handleDeploymentEvent handles a deployment event and scales the deployment if required.
func handleDeploymentEvent(deployment *appsv1.Deployment, clientset *kubernetes.Clientset) {
	annotations := deployment.GetAnnotations()
	if annotations == nil || annotations["demo-controller.local/ha"] != "true" {
		return
	}

	if *deployment.Spec.Replicas <= 1 {
		klog.Infof("Scaling Deployment %s/%s to 2 replicas\n", deployment.Namespace, deployment.Name)
		replicas := int32(2)
		deployment.Spec.Replicas = &replicas
		_, err := clientset.AppsV1().Deployments(deployment.Namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
		if err != nil {
			klog.Errorf("Failed to scale Deployment %s/%s: %v\n", deployment.Namespace, deployment.Name, err)
		}
	}
}

// runController is the main loop for processing deployment events.
func runController(workQueue workqueue.RateLimitingInterface, sharedInformer cache.SharedIndexInformer, clientset *kubernetes.Clientset) {
	klog.Infoln("Watching deployments...")
	for {
		// Get the next deployment key from the work queue
		key, shutdown := workQueue.Get()
		if shutdown {
			return
		}

		// Process the deployment event
		err := processDeploymentEvent(key.(string), sharedInformer, clientset)
		if err != nil {
			klog.Errorf("Error processing Deployment event: %v\n", err)
			workQueue.AddRateLimited(key)
		} else {
			workQueue.Forget(key)
		}

		workQueue.Done(key)
	}
}

// processDeploymentEvent processes a deployment event and delegates to handleDeploymentEvent.
func processDeploymentEvent(key string, sharedInformer cache.SharedIndexInformer, clientset *kubernetes.Clientset) error {
	// Retrieve the deployment object from the shared informer
	obj, exists, err := sharedInformer.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	deployment := obj.(*appsv1.Deployment)
	handleDeploymentEvent(deployment, clientset)

	return nil
}

// getKubeConfigPath returns the path to the kubeconfig file.
func getKubeConfigPath() string {
	// Check if running inside a Kubernetes cluster
	if _, ok := os.LookupEnv("KUBERNETES_SERVICE_HOST"); ok {
		return ""
	}

	// Check if KUBECONFIG environment variable is set
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	// Use the default kubeconfig file path in the user's home directory
	home := homedir.HomeDir()
	return filepath.Join(home, ".kube", "config")
}
