package main

import (
	"flag"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"os"
	"time"
)

const (
	logFormatLogfmt                     = "logfmt"
	logFormatJson                       = "json"

	RULER_HTTP_PORT = "10902"
)

var (
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
)

func main() {

	/*var logFormat string = "logfmt"
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	if logFormat == logFormatJson {
		logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	}*/
	logger = log.With(logger, "ts", log.DefaultTimestamp)
	logger = log.With(logger, "caller", log.DefaultCaller)

	var (
		kubeConfigFile *string
		err            error
		config         *rest.Config
	)

	kubeConfigFile = flag.String("kubeConfigFile", "/root/.kube/config", "kubernetes config file path")
	flag.Parse()
	config, err = clientcmd.BuildConfigFromFlags("", *kubeConfigFile)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			level.Error(logger).Log("msg", "cannot get a kubeconfig.")
			return
		}
	}
	client := kubernetes.NewForConfigOrDie(config)
	factory := informers.NewSharedInformerFactoryWithOptions(client, 45 * time.Second, informers.WithNamespace("monitoring"))
	cmController := factory.Core().V1().ConfigMaps()
	cmInformer := cmController.Informer()

	cmInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) {
			cm := obj.(*v1.ConfigMap)
			if cm.Labels == nil  {
				return
			}
			if cm.Labels["name"] == "thanos-rules" {
				level.Info(logger).Log("msg", "thanos ruler configmap object is created.")
				triggerReload()
				level.Info(logger).Log("msg", "thanos ruler has reloaded config files.")
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			var ok bool
			oldCm, ok := oldObj.(*v1.ConfigMap)
			if !ok {
				return
			}
			newCm := newObj.(*v1.ConfigMap)
			if !ok {
				return
			}
			if oldCm.ResourceVersion == newCm.ResourceVersion {
				return
			}
			if newCm.Labels == nil  {
				return
			}

			if newCm.Labels["name"] == "thanos-rules" {
				level.Info(logger).Log("msg", "thanos ruler configmap object has changed.")
				triggerReload()
				level.Info(logger).Log("msg", "thanos ruler has reloaded config files.")
			}
		},
		DeleteFunc: func(obj interface{}){
		},
	})

	stopCh := make(chan struct{})
	factory.Start(stopCh)
	factory.WaitForCacheSync(stopCh)
	<- stopCh
}

func triggerReload() {
	count := 10
	for i:= 0; i < count; i++ {
		time.Sleep(time.Second * 10)
		sendReloadReq()
	}
}

func sendReloadReq() {
	var (
		req  *http.Request
		resp *http.Response
		err error
	)
	//url := "http://" + "192.168.30.67:30539" + "/-/reload"
	url := "http://" + "localhost:" + RULER_HTTP_PORT + "/-/reload"
	req, _ = http.NewRequest("POST", url, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		level.Error(logger).Log("msg", fmt.Sprintf("http post req %s", req), "err", err)
		return
	}
	defer resp.Body.Close()
}







