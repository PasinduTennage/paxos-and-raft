package src

import (
	"async-consensus/common"
	"async-consensus/proto"
	"fmt"
	"os"
	"strconv"
	"time"
)

// entry defines a single log entry in raft

type raftInstance struct {
	term      int64
	commands  *proto.ReplicaBatch
	decided   bool
	decisions *proto.ReplicaBatch
}

// raft defines the data used for Raft SMR

type Raft struct {
	name     int32
	numNodes int32

	log                  []raftInstance
	commitIndex          int64
	nextFreeIndex        int
	votedFor             int32
	currentTerm          int64
	lastProposedLogIndex int

	viewTimer         *common.TimerWithCancel
	startTime         time.Time
	lastCommittedTime time.Time
	lastProposedTime  time.Time

	state          string
	replica        *Replica
	pipelineLength int
}

/*
	init Raft Consensus data structs
*/

func InitRaftConsensus(numReplicas int, name int32, replica *Replica, pipelineLength int) *Raft {

	replicatedLog := make([]raftInstance, 0)
	// create the genesis slot

	replicatedLog = append(replicatedLog, raftInstance{
		term: 0,
		commands: &proto.ReplicaBatch{
			UniqueId: "nil",
			Requests: make([]*proto.ClientBatch, 0),
			Sender:   -1,
		},
		decided: true,
		decisions: &proto.ReplicaBatch{
			UniqueId: "nil",
			Requests: make([]*proto.ClientBatch, 0),
			Sender:   -1,
		},
	})

	return &Raft{
		name:                 name,
		numNodes:             int32(numReplicas),
		log:                  replicatedLog,
		commitIndex:          0,
		nextFreeIndex:        1,
		votedFor:             -1,
		currentTerm:          0,
		lastProposedLogIndex: 0,
		viewTimer:            nil,
		startTime:            time.Time{},
		lastCommittedTime:    time.Time{},
		lastProposedTime:     time.Time{},
		state:                "A",
		replica:              replica,
		pipelineLength:       pipelineLength,
	}
}

// start the initial leader

func (r *Raft) run() {
	r.startTime = time.Now()
	r.lastCommittedTime = time.Now()
	r.lastProposedTime = time.Now()
	initLeader := int32(2)

	if r.name == initLeader {
		r.replica.sendRequestVote()
	}

}

/*
	append N new instances to the log
*/

func (rp *Replica) createNRaftInstances(number int) {

	for i := 0; i < number; i++ {

		rp.raftConsensus.log = append(rp.raftConsensus.log, raftInstance{
			term:      rp.raftConsensus.currentTerm,
			commands:  nil,
			decided:   false,
			decisions: nil,
		})

		rp.raftConsensus.nextFreeIndex++
	}
}

/*
	check if the instance number instance is already there, if not create a new instance
*/

func (rp *Replica) createRaftInstanceIfMissing(instanceNum int) {

	numMissingEntries := instanceNum - rp.raftConsensus.nextFreeIndex + 1

	if numMissingEntries > 0 {
		rp.createNRaftInstances(numMissingEntries)
	}
}

/*
	handler for generic raft messages
*/

func (rp *Replica) handleRaftConsensus(message *proto.RaftConsensus) {

	if message.Type == 1 {
		rp.debug("Received a append request message from "+strconv.Itoa(int(message.Sender))+
			" for term "+strconv.Itoa(int(message.Term))+" for last log index "+strconv.Itoa(int(message.PrevLogIndex))+" at time "+fmt.Sprintf("%v", time.Now().Sub(rp.raftConsensus.startTime).Milliseconds()), 0)
		rp.handleAppendRequest(message)
	}

	if message.Type == 2 {
		rp.debug("Received a append response message from "+strconv.Itoa(int(message.Sender))+
			" for term "+strconv.Itoa(int(message.Term))+" at time "+fmt.Sprintf("%v", time.Now().Sub(rp.raftConsensus.startTime).Milliseconds()), 0)
		rp.handleAppendResponse(message)
	}

	if message.Type == 3 {
		rp.debug("Received a leader request message from "+strconv.Itoa(int(message.Sender))+
			" for try "+strconv.Itoa(int(message.Term))+" at time "+fmt.Sprintf("%v", time.Now().Sub(rp.raftConsensus.startTime).Milliseconds()), 0)
		rp.handleLeaderRequest(message)
	}

	if message.Type == 4 {
		rp.debug("Received a leader response message from "+strconv.Itoa(int(message.Sender))+
			" for try "+strconv.Itoa(int(message.Term))+" at time "+fmt.Sprintf("%v", time.Now().Sub(rp.raftConsensus.startTime).Milliseconds()), 0)
		rp.handleLeaderResponse(message)
	}

	if message.Type == 5 {
		rp.debug("Received an internal timeout message from "+strconv.Itoa(int(message.Sender))+
			" for try "+strconv.Itoa(int(message.Term))+" at time "+fmt.Sprintf("%v", time.Now().Sub(rp.raftConsensus.startTime).Milliseconds()), 0)
		rp.handleRaftInternalTimeout(message)
	}
}

/*
	Sets a timer, which once timeout will send an internal notification
*/

func (rp *Replica) setRaftViewTimer(term int32) {

	rp.raftConsensus.viewTimer = common.NewTimerWithCancel(time.Duration(rp.viewTimeout) * time.Microsecond)

	rp.raftConsensus.viewTimer.SetTimeoutFuntion(func() {

		// this function runs in a separate thread
		internalTimeoutNotification := proto.RaftConsensus{
			Sender:   rp.name,
			Receiver: rp.name,
			Type:     5,
			Term:     int64(term),
		}

		rpcPair := common.RPCPair{
			Code: rp.messageCodes.RaftConsensus,
			Obj:  &internalTimeoutNotification,
		}
		rp.sendMessage(rp.name, rpcPair)
		rp.debug("Sent an internal timeout notification for view "+strconv.Itoa(int(term))+" at time "+fmt.Sprintf("%v", time.Now().Sub(rp.raftConsensus.startTime).Milliseconds()), 0)

	})
	rp.raftConsensus.viewTimer.Start()
}

/*
	print the replicated log to check for log consistency
*/

func (rp *Replica) printRaftLogConsensus() {
	f, err := os.Create(rp.logFilePath + strconv.Itoa(int(rp.name)) + "-consensus.txt")
	if err != nil {
		panic(err.Error())
	}
	defer f.Close()

	for i := int64(1); i <= rp.raftConsensus.commitIndex; i++ {
		if rp.raftConsensus.log[i].decided == false {
			panic("should not happen")
		}
		for j := 0; j < len(rp.raftConsensus.log[i].decisions.Requests); j++ {
			for k := 0; k < len(rp.raftConsensus.log[i].decisions.Requests[j].Requests); k++ {
				_, _ = f.WriteString(strconv.Itoa(int(i)) + "-" + strconv.Itoa(int(j)) + "-" + strconv.Itoa(int(k)) + "-" + ":" + rp.raftConsensus.log[i].decisions.Requests[j].Requests[k].Command + "\n")

			}
		}
	}
}
