package apis

import "net/rpc"

const SchedulerMainName = "SchedulerMain"

type RegisterArgs struct {
}

type RegisterReply struct {
	SchedulerID       int
	SchedulePartition string
}

type RequestScheduleArgs struct {
	SchedulerID int
	PodID       int
}

type RequestScheduleReply struct {
	IsPermitted bool
}

type UpdatePodStatusArgs struct {
	PodID  int
	Status int
}

type UpdatePodStatusReply struct {
}

type HeartBeatArgs struct {
	SchedulerID int
}

type HeartBeatReply struct {
}

type GetNodePartitionArgs struct {
	SchedulerID int
}

type GetNodePartitionReply struct {
	NodePartition map[string]bool
}

// SchedulerMainService rpc接口定义
type SchedulerMainService interface {
	// RegisterScheduler scheduler worker节点注册rpc
	RegisterScheduler(args RegisterArgs, reply *RegisterReply) error
	// RequestSchedule 请求调度pod rpc
	RequestSchedule(args RequestScheduleArgs, reply *RequestScheduleReply) error
	// UpdatePodStatus 更新pod状态rpc
	UpdatePodStatus(args UpdatePodStatusArgs, reply *UpdatePodStatusReply) error
	// HeartBeat scheduler worker心跳rpc
	HeartBeat(args HeartBeatArgs, reply *HeartBeatReply) error
	// GetNodePartition 获得当前分区节点rpc
	GetNodePartition(args GetNodePartitionArgs, reply *GetNodePartitionReply) error
}

// RegisterService 注册rpc
func RegisterService(service SchedulerMainService) error {
	return rpc.RegisterName(SchedulerMainName, service)
}
