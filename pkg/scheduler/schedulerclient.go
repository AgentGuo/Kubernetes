package scheduler

import (
	"crypto/md5"
	"fmt"
	"k8s.io/kubernetes/pkg/schedulermain"
	"k8s.io/kubernetes/pkg/schedulermain/apis"
	"log"
	"math/rand"
	"net/rpc"
	"time"
)

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

func md5Sum32(str string) int {
	tmp := md5.Sum([]byte(str))
	return int((uint64(tmp[0]) | uint64(tmp[1])<<8 | uint64(tmp[2])<<16 | uint64(tmp[3])<<24 |
		uint64(tmp[4])<<32 | uint64(tmp[5])<<40 | uint64(tmp[6])<<48 | uint64(tmp[7])<<56) & 0x7fffffffffffffff)
}

func registerScheduler() int {
	reply := apis.RegisterReply{}
	err := call("RegisterScheduler", apis.RegisterArgs{}, &reply)
	if err == nil {
		fmt.Printf("scheduler registers successfully, schedulerID:%d\n", reply.SchedulerID)
		return reply.SchedulerID
	}
	return -1
}

func runSchedulerRequest(podName string, namespace string, schedulerID int) (int, bool) {
	rand.Seed(time.Now().Unix())
	podID := md5Sum32(podName + namespace)
	fmt.Printf("schedule pod: podName: %s, namespace: %s, podID: %d\n", podName, namespace, podID)
	args := apis.RequestScheduleArgs{
		SchedulerID: schedulerID,
		PodID:       podID,
	}
	reply := apis.RequestScheduleReply{}
	err := call("RequestSchedule", args, &reply)
	if err != nil {
		return -1, false
	}
	fmt.Println("isPermitted:", reply.IsPermitted)
	return podID, true
}

func runFinishSchedule(podID int) {
	args := apis.UpdatePodStatusArgs{
		PodID:  podID,
		Status: schedulermain.PodFinished,
	}
	reply := apis.UpdatePodStatusReply{}
	err := call("UpdatePodStatus", args, &reply)
	if err != nil {
		return
	}
}
