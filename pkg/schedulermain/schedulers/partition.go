package schedulers

import (
	"fmt"
	"k8s.io/kubernetes/pkg/schedulermain/apis"
	"log"
)

func (s *Schedulers) UpdateScheduler() {
	if len(s.NodeMap) >= len(s.SchedulerMap) { // node数量大于scheduler数量才能分配
		if len(s.SchedulerMap) == 0 {
			log.Printf("There is no scheduler\n")
			return
		}
		nodeNum := len(s.NodeMap) / len(s.SchedulerMap)
		nodeList := make([]string, 0, len(s.NodeMap))
		for node, _ := range s.NodeMap {
			nodeList = append(nodeList, node)
		}
		s.SchedulerPartition = make(map[int][]string)
		lastID, nodeIndex := 0, 0
		for id, _ := range s.SchedulerMap { // 进行分区，每个scheduler有nodeNum个节点
			partition := make([]string, 0, nodeNum)
			for i := 0; i < nodeNum; i++ {
				partition = append(partition, nodeList[nodeIndex+i])
			}
			s.SchedulerPartition[id] = partition
			nodeIndex += nodeNum
			lastID = id
		}
		for ; nodeIndex < len(nodeList); nodeIndex++ { // 未整除，都分配给最后的scheduler
			s.SchedulerPartition[lastID] = append(s.SchedulerPartition[lastID], nodeList[nodeIndex])
		}
		log.Printf("new partition:\n")
		for id, nodes := range s.SchedulerPartition {
			log.Printf("scheduler%d:%v\n", id, nodes)
		}
	} else {
		log.Printf("schedulerNum is larger than nodeNum\n")
	}
}

func (s *Schedulers) GetNodePartition(args apis.GetNodePartitionArgs, reply *apis.GetNodePartitionReply) error {
	s.SchedulerRWLock.RLock()
	defer s.SchedulerRWLock.RUnlock()
	if nodeList, ok := s.SchedulerPartition[args.SchedulerID]; ok {
		if len(nodeList) == 0 {
			return fmt.Errorf("node partiton is empty")
		}
		nodePartition := make(map[string]bool)
		for _, n := range nodeList {
			nodePartition[n] = true
		}
		(*reply).NodePartition = nodePartition
		return nil
	} else {
		return fmt.Errorf("scheduler not found")
	}
}
