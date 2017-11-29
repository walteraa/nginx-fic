package controller

import (
  "time"
  "reflect"
  "github.com/golang/glog"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/controller"
	"k8s.io/federation/pkg/federation-controller/util"
  extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
  federationclientset "k8s.io/federation/client/clientset_generated/federation_clientset"
	pkgruntime "k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  federationapi "k8s.io/federation/apis/federation/v1beta1"
  kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/watch"
  "fmt"
  "k8s.io/client-go/rest"
  apiv1 "k8s.io/api/core/v1"
)

type NGINXFedIngressController struct{
  client federationclientset.Interface
  informerController    cache.Controller
  store         cache.Store
	ingressFederatedInformer util.FederatedInformer
}

type BackendServer struct{
	Servers []string
	Port string
}

type IngressPath struct{
	Path string
	Backend BackendServer
}


func NewNGINXFedIngressController(client federationclientset.Interface, resyncPeriod time.Duration)(*NGINXFedIngressController, error){

    glog.Infof("Creating IngressController")
    fmt.Printf("Creating Federated Ingress Controller\n")
    nic := &NGINXFedIngressController{
      client: client,
    }


    handlers := &cache.ResourceEventHandlerFuncs{
        DeleteFunc: func(old interface{}){
          ingress := old.(*extensionsv1beta1.Ingress)
          err := DeleteCfg(ingress.Name)
          if err == nil{
             fmt.Printf("[DELETE] Ingress{ Name:%s, Namespace: %s  } deleted successfully!",ingress.Name, ingress.Namespace)
           }else{
              fmt.Printf("[DELETE] Error when deleting Ingress{ Name:%s, Namespace%s}",ingress.Name,ingress.Namespace)
           }
        },
        AddFunc: func(cur interface{}){
          ingress := cur.(*extensionsv1beta1.Ingress)
          fmt.Printf("[CREATE] Ingress{ Name:%s, Namespace: %s  }",ingress.Name, ingress.Namespace)

					paths := ingress.Spec.Rules[0].IngressRuleValue.HTTP.Paths
					var ingressPaths []IngressPath
					for _, path := range paths{
              serviceName := path.Backend.ServiceName
              servicePort := path.Backend.ServicePort
              service,errsvc := client.CoreV1().Services(ingress.Namespace).Get(serviceName, metav1.GetOptions{})

              backendIngresses := service.Status.LoadBalancer.Ingress
              backend := BackendServer{Port: servicePort.String() }
              var servers []string

              for _,ingress := range backendIngresses{
                  servers = append(servers, ingress.IP) 
              }

              backend.Servers = servers

              if errsvc == nil{
                  fmt.Printf("The path %v has the service: %v",path.Path, backend)
              }
							ingressPaths = append(ingressPaths,IngressPath{ Path:path.Path, Backend: backend  })

					}
					configMapData, err := WriteCfg(ingress.Name, ingressPaths)
					if err != nil{
						fmt.Printf("Error creating configuration for %s. Error: %v",ingress.Name,err)
					}else{
            configMap,errcfg := SaveConfigMap(ingress.Namespace,ingress.Name, configMapData) 
            if errcfg != nil{
              fmt.Printf("NGINX configuration file created for %s Ingress(ConfigMap):\n\n %v",ingress.Name,configMap)
            }else{
              fmt.Printf("Error when creating NGINX configmap for %v Ingress",ingress.Name)
            }
          }
        },
        UpdateFunc: func(old,cur interface{}){
          newIngress := cur.(*extensionsv1beta1.Ingress)
          if old !=nil && cur != nil && !reflect.DeepEqual(old,cur){
            fmt.Printf("[UPDATE] Ingress{ Name:%s, Namespace: %s  }",newIngress.Name, newIngress.Namespace)
          }
        },
    }

    nic.store, nic.informerController = cache.NewInformer(
        &cache.ListWatch{
          ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error){
            return client.Extensions().Ingresses(metav1.NamespaceAll).List(options)
          },
          WatchFunc: func(options metav1.ListOptions) (watch.Interface, error){
            return client.Extensions().Ingresses(metav1.NamespaceAll).Watch(options)
          },
        },
        &extensionsv1beta1.Ingress{},
        controller.NoResyncPeriodFunc(),
        util.NewTriggerOnAllChanges(
                    func(obj pkgruntime.Object){
                      glog.Infof("Object changed: %v", obj)
                    },
        ))

    nic.ingressFederatedInformer = util.NewFederatedInformer(
        client,
        func(cluster *federationapi.Cluster, targetClient kubeclientset.Interface) (cache.Store, cache.Controller){
            return cache.NewInformer(
                &cache.ListWatch{
                    ListFunc: func(options metav1.ListOptions) (pkgruntime.Object, error){
                      return targetClient.Extensions().Ingresses(metav1.NamespaceAll).List(options)
                    },
                    WatchFunc: func(options metav1.ListOptions) (watch.Interface, error){
                      return targetClient.Extensions().Ingresses(metav1.NamespaceAll).Watch(options)
                    },
                },
                &extensionsv1beta1.Ingress{},
                controller.NoResyncPeriodFunc(),
                //Do something when some ingress changes(add/remove/update)
                handlers)
        },

        //Do procedures when a new cluster becomes available
        &util.ClusterLifecycleHandlerFuncs{
            ClusterAvailable: func(cluster *federationapi.Cluster){
              glog.Infof("The cluster %v became available", cluster.Name)
              fmt.Printf("\n--->The cluster %v became available.\n\n",cluster.Name)
            },
        },

    )

    return nic, nil
}

func SaveConfigMap(namespace,name, data string ) (*apiv1.ConfigMap, error){
  config, err := rest.InClusterConfig()
  if err != nil{
    return &apiv1.ConfigMap{},err
  }

  clientset,err := kubeclientset.NewForConfig(config)

  cmap := &apiv1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
			Name: name + "_configmap",
      Namespace: namespace,
		},
  }

  cmapdata := map[string]string{}
  cmapdata["nginx_conf"] = data

  cmap.Data = cmapdata

  return clientset.CoreV1().ConfigMaps(namespace).Create(cmap)
}


func (nic *NGINXFedIngressController) Run(stopCh <- chan struct{}){
  glog.Infof("Starting NGINX Federated Ingress Controller")
  fmt.Printf("Starting NGINX Federated Ingress Controller\n")

  go nic.informerController.Run(stopCh)
  fmt.Printf("Informer controller started!\n")
  glog.Infof("Starting Fedrated Ingress Informer")
  fmt.Printf("Starting Fedrated Ingress Informer\n")
  go nic.ingressFederatedInformer.Start()
  fmt.Printf("Federated Ingress Informer started!\n")

	<-stopCh
  glog.Infof("Stopping NGINX Federated Ingress Informer")
  fmt.Printf("Stopping NGINX Federated Ingress Informer\n")
  nic.ingressFederatedInformer.Stop()
  
}
