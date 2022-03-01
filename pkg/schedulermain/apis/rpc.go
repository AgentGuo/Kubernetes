package apis

import "net/rpc"

const SchedulerMainName = "SchedulerMain"

type RegisterArgs struct {
}

type RegisterReply struct {
	SchedulerID int
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

type SchedulerMainService interface {
	RegisterScheduler(args RegisterArgs, reply *RegisterReply) error
	RequestSchedule(args RequestScheduleArgs, reply *RequestScheduleReply) error
	UpdatePodStatus(args UpdatePodStatusArgs, reply *UpdatePodStatusReply) error
}

func RegisterService(service SchedulerMainService) error {
	return rpc.RegisterName(SchedulerMainName, service)
}
