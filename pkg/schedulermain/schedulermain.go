package schedulermain

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubernetes/pkg/schedulermain/apis"
	"k8s.io/kubernetes/pkg/schedulermain/podschedule"
	"k8s.io/kubernetes/pkg/schedulermain/schedulers"
	"k8s.io/kubernetes/pkg/schedulermain/utils"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
	"log"
	"net"
	"net/rpc"
	"path/filepath"
	"sync"
	"time"
)

type SchedulerMain struct {
	NodeMetricsClientSet *metricsv.Clientset
	schedulers.Schedulers
	podschedule.PodSchedule
}

// server 启动rpc服务
func (s *SchedulerMain) server(port int) {
	var wg sync.WaitGroup
	err := apis.RegisterService(s)
	if err != nil {
		log.Fatal("register error:", err)
		return
	}
	listener, err := net.Listen("tcp", ":"+fmt.Sprintf("%d", port))
	if err != nil {
		log.Fatal("listen error:", err)
		return
	}
	wg.Add(1)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatal("listen accept error:", err)
				continue
			}
			go rpc.ServeConn(conn)
		}
	}()
	// 启动后台清理pod协程
	go s.DeletePod()
	// 启动后台更新scheduler worker权重协程
	go func() {
		for {
			s.updateSchedulerWeight()
			time.Sleep(3 * time.Second)
		}
	}()
	// 无限等待
	wg.Wait()
}

// NewSchedulerMain 根据配置信息创建scheduler main实例
func NewSchedulerMain(conf *apis.SchedulerMainConf) SchedulerMain {
	utils.V = conf.V
	podschedule.ScheduleTimeoutInterval = time.Duration(conf.ScheduleTimeoutInterval) * time.Second
	podschedule.PodExpire = time.Duration(conf.PodExpire) * time.Second
	schedulers.HeartBeatTimeout = time.Duration(conf.HeartBeatTimeout) * time.Second
	return SchedulerMain{
		NodeMetricsClientSet: NewNodeMetricsClientSet(),
		Schedulers:           schedulers.NewSchedulers(conf.Partition),
		PodSchedule:          podschedule.NewPodSchedule(),
	}
}

// NewNodeMetricsClientSet 获得Metrics Client
func NewNodeMetricsClientSet() *metricsv.Clientset {
	kubeconfig := filepath.Join(schedulers.HomeDir(), ".kube", "config")
	// uses the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	metricsClientSet, err := metricsv.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return metricsClientSet
}

// nodeResource 定义node资源结构体
type nodeResource struct {
	cpu, memory float64
}

// updateSchedulerWeight 更新各个调度器权重
func (s *SchedulerMain) updateSchedulerWeight() {
	nodesResource := make(map[string]nodeResource)
	cpuSum := resource.Quantity{}
	memorySum := resource.Quantity{}
	// 默认2核4G
	defaultCpu, _ := resource.ParseQuantity("2")
	defaultMemory, _ := resource.ParseQuantity("4000000000")
	metricsNodes, err := s.NodeMetricsClientSet.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, node := range metricsNodes.Items {
			node2, err := s.NodeLister.Get(node.Name)
			if err != nil {
				panic(err.Error())
			}
			//fmt.Printf("%s [usage]: cpu:%d, memory:%d, storge:%s pod:%s \n",
			//	node.Name,
			//	node.Usage.Cpu().MilliValue(),
			//	node.Usage.Memory().Value(),
			//	node.Usage.Storage(),
			//	node.Usage.Pods())
			//fmt.Printf("  [allocatable]: cpu:%d, memory:%d, storge:%s pod:%s \n",
			//	node2.Status.Allocatable.Cpu().MilliValue(),
			//	node2.Status.Allocatable.Memory().Value(),
			//	node2.Status.Allocatable.Storage(),
			//	node2.Status.Allocatable.Pods())
			cpu := resource.Quantity{}
			cpu.Add(*(node2.Status.Allocatable.Cpu()))
			cpu.Sub(*(node.Usage.Cpu()))
			cpuSum.Add(cpu)
			memory := resource.Quantity{}
			memory.Add(*(node2.Status.Allocatable.Memory()))
			memory.Sub(*(node.Usage.Memory()))
			memorySum.Add(memory)
			nodesResource[node.Name] = nodeResource{
				cpu:    float64(cpu.MilliValue()),
				memory: float64(memory.Value()),
			}
		}
	}
	s.PodRWLock.Lock()
	s.SchedulerRWLock.RLock()
	for n, _ := range s.NodeMap { // 如果节点metrics api无法监测，就设为默认值
		if _, ok := nodesResource[n]; !ok {
			nodesResource[n] = nodeResource{ // 默认值为2核4G
				cpu:    float64(defaultCpu.MilliValue()),
				memory: float64(defaultMemory.Value()),
			}
			cpuSum.Add(defaultCpu)
			memorySum.Add(defaultMemory)
		}
	}
	s.SchedulerWeight = make(map[int]float64)
	for id, nodes := range s.SchedulerNodeList {
		s.SchedulerWeight[id] = 0
		for _, n := range nodes {
			s.SchedulerWeight[id] += (nodesResource[n].cpu/float64(cpuSum.MilliValue()) +
				nodesResource[n].memory/float64(memorySum.Value())) / 2
		}
	}
	if utils.V >= 4 {
		nodeInfoString := ""
		for node, info := range nodesResource {
			nodeInfoString += fmt.Sprintf("%s:{cpu:%.2f, memory:%.2fMB},",
				node,
				info.cpu/1000,
				info.memory/1024/1024)
		}
		nodeInfoString = nodeInfoString[:len(nodeInfoString)-1]
		utils.LogV(4, fmt.Sprintf("[node resource surplus] %s", nodeInfoString))
	}
	utils.LogV(3, fmt.Sprintf("[update scheduler weight]: %v", s.SchedulerWeight))
	s.SchedulerRWLock.RUnlock()
	s.PodRWLock.Unlock()
}

// RunSchedulerMain 运行scheduler main
func RunSchedulerMain(conf *apis.SchedulerMainConf) {
	s := NewSchedulerMain(conf)
	s.InitInformer()
	s.server(conf.Port)
}
