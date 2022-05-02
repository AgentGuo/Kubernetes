package scheduler

import (
	"crypto/md5"
	"k8s.io/kubernetes/pkg/schedulermain/apis"
	"k8s.io/kubernetes/pkg/schedulermain/podschedule"
	"log"
	"math/rand"
	"net/rpc"
	"time"
)

// HeartBeatInterval 心跳间隔
const HeartBeatInterval = 2 * time.Second

// call rpc调用通用call
func call(rpcName string, args interface{}, reply interface{}) error {
	client, err := rpc.Dial("tcp", "localhost:12345")
	if err != nil {
		log.Fatal("dial error:", err)
		return err
	}
	err = client.Call(apis.SchedulerMainName+"."+rpcName, args, reply)
	if err != nil {
		log.Fatal("call error:", err)
		return err
	}
	client.Close()
	return nil
}

// md5Sum32 字符串32位md5
func md5Sum32(str string) int {
	tmp := md5.Sum([]byte(str))
	// md5默认64位，这里截取了前32位
	return int((uint64(tmp[0]) | uint64(tmp[1])<<8 | uint64(tmp[2])<<16 | uint64(tmp[3])<<24 |
		uint64(tmp[4])<<32 | uint64(tmp[5])<<40 | uint64(tmp[6])<<48 | uint64(tmp[7])<<56) & 0x7fffffffffffffff)
}

// registerScheduler 注册调度器，注册成功获得SchedulerID，失败返回-1
func registerScheduler() (int, string) {
	reply := apis.RegisterReply{}
	err := call("RegisterScheduler", apis.RegisterArgs{}, &reply)
	if err == nil {
		log.Printf("scheduler registers successfully, schedulerID:%d\n", reply.SchedulerID)
		return reply.SchedulerID, reply.SchedulePartition
	}
	return -1, ""
}

// runSchedulerRequest 请求调度pod
func runSchedulerRequest(podName string, namespace string, schedulerID int) (int, bool) {
	rand.Seed(time.Now().Unix())
	// 由podName和podNamespace计算md5得到podID，防止冲突
	podID := md5Sum32(podName + namespace)
	args := apis.RequestScheduleArgs{
		SchedulerID: schedulerID,
		PodID:       podID,
	}
	reply := apis.RequestScheduleReply{}
	err := call("RequestSchedule", args, &reply)
	if err != nil {
		return -1, false
	}
	log.Printf("schedule pod: podName: %s, namespace: %s, podID: %d, isPermitted: %t\n",
		podName, namespace, podID, reply.IsPermitted)
	return podID, reply.IsPermitted
}

// runFinishSchedule 调度完成，更新pod状态
func runFinishSchedule(podID int) {
	args := apis.UpdatePodStatusArgs{
		PodID:  podID,
		Status: podschedule.PodFinished,
	}
	reply := apis.UpdatePodStatusReply{}
	err := call("UpdatePodStatus", args, &reply)
	if err != nil {
		return
	}
}

// runHeartBeat 运行心跳
func runHeartBeat(schedulerID int) {
	for {
		select {
		case <-time.After(HeartBeatInterval):
			reply := apis.HeartBeatReply{}
			err := call("HeartBeat", apis.HeartBeatArgs{SchedulerID: schedulerID}, &reply)
			if err != nil {
				return
			}
		}
	}
}
