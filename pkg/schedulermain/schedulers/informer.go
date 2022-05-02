package schedulers

import (
	"fmt"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/schedulermain/utils"
	"os"
	"path/filepath"
	"time"
)

// HomeDir 获取home目录
func HomeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// InitInformer 初始化informer
func (s *Schedulers) InitInformer() {
	kubeConfig := filepath.Join(HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		panic(err.Error())
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	// 初始化informer
	factory := informers.NewSharedInformerFactory(clientSet, time.Minute)
	stopper := make(chan struct{})
	nodeInformer := factory.Core().V1().Nodes()
	go factory.Start(stopper)
	informer := nodeInformer.Informer()
	// 添加更新node的handle
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    s.addNode,
		UpdateFunc: s.updateNode,
		DeleteFunc: s.deleteNode,
	})
	go informer.Run(stopper)
	if !cache.WaitForCacheSync(stopper, nodeInformer.Informer().HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}
	s.NodeLister = nodeInformer.Lister()
	nodes, err := s.NodeLister.List(labels.Everything())
	if err != nil {
		panic(err.Error())
	}
	s.SchedulerRWLock.Lock()
	defer s.SchedulerRWLock.Unlock()
	for _, node := range nodes {
		if nodeIsReady(node) {
			s.NodeMap[node.Name] = true
		}
	}
}

func getNodeListString(nodeMap map[string]bool) string {
	nodeString := "{"
	for node, _ := range nodeMap {
		nodeString += node + ","
	}
	nodeString = nodeString[:len(nodeString)-1]
	nodeString += "}"
	return nodeString
}

func (s *Schedulers) addNode(obj interface{}) { // 增加node
	node := obj.(*v1.Node)
	if nodeIsReady(node) {
		s.SchedulerRWLock.Lock()
		defer s.SchedulerRWLock.Unlock()
		s.NodeMap[node.Name] = true
		utils.LogV(1, fmt.Sprintf("[add node] add node %s, node list:%s",
			node.Name, getNodeListString(s.NodeMap)))
		s.UpdateSchedulerPartition()
	}
}

func (s *Schedulers) updateNode(oldObj interface{}, newObj interface{}) { // 更新node
	oldNode := oldObj.(*v1.Node)
	newNode := newObj.(*v1.Node)
	if oldNode.Name == newNode.Name && nodeIsReady(oldNode) == nodeIsReady(newNode) { // 名字和状态都没变，不更新
		return
	}
	// 否则进行更新
	s.SchedulerRWLock.Lock()
	defer s.SchedulerRWLock.Unlock()
	if oldNode.Status.Conditions[len(oldNode.Status.Conditions)-1].Type == v1.NodeReady {
		delete(s.NodeMap, oldNode.Name)
	}
	if newNode.Status.Conditions[len(newNode.Status.Conditions)-1].Type == v1.NodeReady {
		s.NodeMap[newNode.Name] = true
	}
	utils.LogV(1, fmt.Sprintf("[update node] old node:%s, new node:%s, node list:%s",
		oldNode.Name, newNode.Name, getNodeListString(s.NodeMap)))
	s.UpdateSchedulerPartition()
}

func (s *Schedulers) deleteNode(obj interface{}) { // 删除node
	node := obj.(*v1.Node)
	if !nodeIsReady(node) {
		s.SchedulerRWLock.Lock()
		defer s.SchedulerRWLock.Unlock()
		delete(s.NodeMap, node.Name)
		utils.LogV(1, fmt.Sprintf("[delete node] delete node %s, node list:%s",
			node.Name, getNodeListString(s.NodeMap)))
		s.UpdateSchedulerPartition()
	}
}

func nodeIsReady(node *v1.Node) bool {
	return node.Status.Conditions[len(node.Status.Conditions)-1].Type == v1.NodeReady &&
		node.Status.Conditions[len(node.Status.Conditions)-1].Status == v1.ConditionTrue
}
