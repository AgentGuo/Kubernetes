package schedulers

import (
	"fmt"
	"k8s.io/kubernetes/pkg/schedulermain/apis"
	"k8s.io/kubernetes/pkg/schedulermain/utils"
)

// UpdateSchedulerPartition 更新scheduler worker负责的partition
func (s *Schedulers) UpdateSchedulerPartition() {
	logPrefix := "[update scheduler partition] "
	if s.NodePartition == nil || len(s.NodePartition) == 0 { // 未手动分区，随机均匀分配
		if len(s.NodeMap) >= len(s.SchedulerMap) { // node数量大于scheduler数量才能分配
			if len(s.SchedulerMap) == 0 {
				utils.LogV(2, fmt.Sprintf("%s there is no scheduler\n", logPrefix))
				return
			}
			nodeNum := len(s.NodeMap) / len(s.SchedulerMap)
			nodeList := make([]string, 0, len(s.NodeMap))
			for node, _ := range s.NodeMap {
				nodeList = append(nodeList, node)
			}
			s.SchedulerNodeList = make(map[int][]string)
			lastID, nodeIndex := 0, 0
			for id, _ := range s.SchedulerMap { // 进行分区，每个scheduler有nodeNum个节点
				partition := make([]string, 0, nodeNum)
				for i := 0; i < nodeNum; i++ {
					partition = append(partition, nodeList[nodeIndex+i])
				}
				s.SchedulerNodeList[id] = partition
				nodeIndex += nodeNum
				lastID = id
			}
			for ; nodeIndex < len(nodeList); nodeIndex++ { // 未整除，都分配给最后的scheduler
				s.SchedulerNodeList[lastID] = append(s.SchedulerNodeList[lastID], nodeList[nodeIndex])
			}
			logPrefix += "new partition: "
			for id, nodes := range s.SchedulerNodeList {
				logPrefix += fmt.Sprintf("scheduler%d:%v ", id, nodes)
			}
			utils.LogV(2, logPrefix)
		} else {
			utils.LogV(2, fmt.Sprintf("%s schedulerNum is larger than nodeNum\n", logPrefix))
		}
	} else { // 已手动分区，将空闲分区分配给scheduler worker
		s.SchedulerNodeList = make(map[int][]string)
		for partition, nodeList := range s.NodePartition { // 遍历所有分区
			scheID, ok := s.PartitionScheduler[partition]
			if !ok { // 未分配，跳过
				continue
			}
			// 已分配，重新检测每个pod的状态
			s.SchedulerNodeList[scheID] = []string{}
			for _, node := range nodeList {
				if s.NodeMap[node] {
					s.SchedulerNodeList[scheID] = append(s.SchedulerNodeList[scheID], node)
				}
			}
		}
		logPrefix += "new partition: "
		for id, nodes := range s.SchedulerNodeList {
			logPrefix += fmt.Sprintf("scheduler%d:%v ", id, nodes)
		}
		utils.LogV(2, logPrefix)
	}
}

// GetNodePartition 获得分区的node信息
func (s *Schedulers) GetNodePartition(args apis.GetNodePartitionArgs, reply *apis.GetNodePartitionReply) error {
	s.SchedulerRWLock.RLock()
	defer s.SchedulerRWLock.RUnlock()
	if nodeList, ok := s.SchedulerNodeList[args.SchedulerID]; ok {
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
