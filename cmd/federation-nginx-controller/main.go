package main
import(
  "os"
  "os/signal"
  "k8s.io/federation/pkg/kubefed/util"
	"k8s.io/client-go/tools/clientcmd"
//  "github.com/golang/glog"
  "k8s.io/federation/cmd/federation-nginx-controller/pkg"
  "fmt"
  "time"
)


func main(){

//  loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
//	configOverrides := &clientcmd.ConfigOverrides{}
//  kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
  config := util.NewAdminConfig(clientcmd.NewDefaultPathOptions())

  fedClientSet, err := config.FederationClientset("federation-controller-manager@kfed", os.Getenv("KUBECONFIG"))

  controller, err := controller.NewNGINXFedIngressController(fedClientSet, 10*time.Second)

  // glog.Infof("kubeconfig: %v+",kubeConfig)
  if err == nil{
    fmt.Printf("Controller: %v", controller)
  }else{
    fmt.Printf("Error: %v", err)
    return
  }
  stopCh := make(chan struct{})
  fmt.Printf("Starting controller...\n")
  go controller.Run(stopCh)
  fmt.Printf("Controller started.\n")
  c := make(chan os.Signal, 1)
  signal.Notify(c, os.Interrupt)
  <-c
  close(stopCh)
}
