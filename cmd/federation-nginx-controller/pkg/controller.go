package controller

import (
  "time"
  "github.com/golang/glog"
	"k8s.io/client-go/tools/cache"
	"k8s.io/federation/pkg/federation-controller/util"
  extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
  federationclientset "k8s.io/federation/client/clientset_generated/federation_clientset"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type NGINXFedIngressController struct{
  client federationclientset.Interface
  informerController    cache.Controller
  store         cache.Store
	ingressFederatedInformer util.FederatedInformer
}

func NewNGINXFedIngressController(client federationclientset.Interface, resyncPeriod time.Duration)(*NGINXFedIngressController, error){

    glog.Infof("Creating IngressController")

    nic := &NGINXFedIngressController{
      client: client,
    }

/*    handlers := framework.ResourceEventHandlerFuncs{
		    AddFunc: func(obj interface{}) {
          glog.Infof("Ingress added")
		    },
		    DeleteFunc: func(obj interface{}) {
          glog.Infof("Ingress removed")
		    },
	  }
*/
		handlers := util.NewTriggerOnAllChanges(
			func(obj pkgruntime.Object) {
				glog.Infof("Function handler")
			},
		)

    nic.store, nic.informerController = cache.NewInformer(
        &cache.ListWatch{
          ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error){
            return client.Extensions().Ingresses(metav1.NamespaceAll).Watch(options)
          },
          WatchFunc: func(options metav1.ListOptions) (watch.Interface, error){
            return client.Extensions().Ingresses(metav1.NamespaceAll).Watch(options) 
          },
        },
        &extensionsv1beta1.Ingress{},
        resyncPeriod,
        handlers)

    return &nic, nil
}

func (nic *NGINXFedIngressController) Run(stopCh <- chan struct{}){
  go nic.informerController.Run(stopCh)
}
