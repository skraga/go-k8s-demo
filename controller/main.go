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

	config, err := clientcmd.BuildConfigFromFlags("", getKubeConfigPath())
	if err != nil {
		klog.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	sharedInformer := newSharedInformer(clientset)
	workQueue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	stopCh := make(chan struct{})
	defer close(stopCh)

	go sharedInformer.Run(stopCh)
	go runController(workQueue, sharedInformer, clientset)

	<-stopCh
}

func newSharedInformer(clientset *kubernetes.Clientset) cache.SharedIndexInformer {
	listWatcher := cache.NewListWatchFromClient(clientset.AppsV1().RESTClient(), "deployments", corev1.NamespaceAll, fields.Everything())

	sharedInformer := cache.NewSharedIndexInformer(
		listWatcher,
		&appsv1.Deployment{},
		0,
		cache.Indexers{},
	)

	_, _ = sharedInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment := obj.(*appsv1.Deployment)
			handleDeploymentEvent(deployment, clientset)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newDeployment := newObj.(*appsv1.Deployment)
			handleDeploymentEvent(newDeployment, clientset)
		},
	})

	return sharedInformer
}

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

func runController(workQueue workqueue.RateLimitingInterface, sharedInformer cache.SharedIndexInformer, clientset *kubernetes.Clientset) {
	klog.Infoln("Watching deployments...")
	for {
		key, shutdown := workQueue.Get()
		if shutdown {
			return
		}

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

func processDeploymentEvent(key string, sharedInformer cache.SharedIndexInformer, clientset *kubernetes.Clientset) error {
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

func getKubeConfigPath() string {
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); err == nil {
		return ""
	}

	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	home := homedir.HomeDir()
	return filepath.Join(home, ".kube", "config")
}
