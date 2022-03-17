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
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
	"log"
	"net"
	"net/rpc"
	"path/filepath"
	"time"
)

type SchedulerMain struct {
	NodeMetricsClientSet *metricsv.Clientset
	schedulers.Schedulers
	podschedule.PodSchedule
}

func (s *SchedulerMain) server(port int) {
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
	go s.DeletePod()
	go func() {
		for {
			s.updateSchedulerWeight()
			time.Sleep(3 * time.Second)
		}
	}()
}

func NewSchedulerMain() SchedulerMain {
	return SchedulerMain{
		NodeMetricsClientSet: NewNodeMetricsClientSet(),
		Schedulers:           schedulers.NewSchedulers(),
		PodSchedule:          podschedule.NewPodSchedule(),
	}
}

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

type nodeResource struct {
	cpu, memory float64
}

func (s *SchedulerMain) updateSchedulerWeight() {
	metricsNodes, err := s.NodeMetricsClientSet.MetricsV1beta1().NodeMetricses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	nodesResource := make(map[string]nodeResource)
	cpuSum := resource.Quantity{}
	memorySum := resource.Quantity{}
	for _, node := range metricsNodes.Items {
		node2, err := s.NodeLister.Get(node.Name)
		if err != nil {
			panic(err.Error())
		}
		//fmt.Printf("%s [usage]: cpu:%s, memory:%s, storge:%s pod:%s \n",
		//	node.Name,
		//	node.Usage.Cpu(),
		//	node.Usage.Memory(),
		//	node.Usage.Storage(),
		//	node.Usage.Pods())
		//fmt.Printf("  [allocatable]: cpu:%s, memory:%s, storge:%s pod:%s \n",
		//	node2.Status.Allocatable.Cpu(),
		//	node2.Status.Allocatable.Memory(),
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
	s.PodRWLock.Lock()
	s.SchedulerRWLock.RLock()
	s.SchedulerWeight = make(map[int]float64)
	for id, nodes := range s.SchedulerPartition {
		s.SchedulerWeight[id] = 0
		for _, n := range nodes {
			s.SchedulerWeight[id] += (nodesResource[n].cpu/float64(cpuSum.MilliValue()) +
				nodesResource[n].memory/float64(memorySum.Value())) / 2
		}
	}
	fmt.Printf("%v\n", s.SchedulerWeight)
	s.SchedulerRWLock.RUnlock()
	s.PodRWLock.Unlock()
	//fmt.Printf("%v\n", nodesResource)
}

func RunSchedulerMain(port int) {
	s := NewSchedulerMain()
	s.InitInformer()
	s.server(port)
	fmt.Println("[enter \"q\" or \"quit\" to stop]")
	input := ""
	for input != "q" && input != "quit" {
		fmt.Scanf("%s", &input)
	}
	fmt.Printf("%v\n", s.SchedulerMap)
}
