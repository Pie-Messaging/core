package routing

import (
	"context"
	"github.com/Pie-Messaging/core/pie"
	"github.com/Pie-Messaging/core/pie/pb"
	"math/big"
	"sync"
	"time"
)

func (r *Table) FindTrackerOnce(ctx context.Context, id []byte, candidates []*Tracker, recvTimeout time.Duration) {
	wg := &sync.WaitGroup{}
	for _, tracker := range candidates {
		tracker := tracker
		wg.Add(1)
		go func() {
			defer wg.Done()
			stream, err := tracker.Session().OpenStream()
			if err != nil {
				return
			}
			if stream.SendMessage(&pb.NetMessage{
				Body: &pb.NetMessage_FindTrackerReq{FindTrackerReq: &pb.FindTrackerReq{
					Id: id,
				}},
			}) != nil {
				return
			}
			message, err := stream.RecvMessage(time.Now().Add(recvTimeout))
			stream.Close()
			if err != nil {
				return
			}
			findTrackerRes := message.GetFindTrackerRes()
			if findTrackerRes == nil {
				return
			}
			if findTrackerRes.Status != pb.Status_OK {
				return
			}
			for _, candidate := range findTrackerRes.Candidates {
				tracker := &Tracker{ID: (&big.Int{}).SetBytes(candidate.Id)}
				tracker.Addr = candidate.Addr
				r.AddAndConnectTracker(ctx, tracker)
			}
		}()
	}
	wg.Wait()
}

func (r *Table) FindTracker(ctx context.Context, id *big.Int, numRequest int, recvTimeout time.Duration) {
	visited := make(map[pie.IDA]struct{})
	for {
		neighbors := r.GetNeighbors(id, numRequest)
		if testAll(len(neighbors), func(i int) bool {
			_, exists := visited[*(*pie.IDA)(neighbors[i].ID.Bytes())]
			return exists
		}) {
			return
		}
		r.FindTrackerOnce(ctx, id.Bytes(), neighbors, recvTimeout)
		for _, tracker := range neighbors {
			visited[*(*pie.IDA)(tracker.ID.Bytes())] = struct{}{}
		}
	}
}
