package routing

import (
	"container/list"
	"context"
	"github.com/Pie-Messaging/core/pie"
	"math"
	"math/big"
	"sync"
)

var (
	MaxRedundancy = math.Max(pie.MetaDataRedundancy, pie.FileDataRedundancy)
)

type Table struct {
	Protocol    string
	trackerMap  map[pie.IDA]*list.Element
	trackerList *list.List
	trackerTree *TreeNode
	mutex       sync.RWMutex
}

type TreeNode struct {
	parent   *TreeNode
	depth    int
	value    *Tracker
	children [2]*TreeNode
}

func (r *Table) Init(ctx context.Context, trackers []*Tracker) {
	if len(trackers) == 0 {
		pie.Logger.Println("No tracker for bootstrap, so I have to wait other trackers to join my network")
	}
	r.trackerMap = make(map[pie.IDA]*list.Element, len(trackers))
	r.trackerList = list.New()
	wg := &sync.WaitGroup{}
	for _, tracker := range trackers {
		if tracker.ID.BitLen() == 0 {
			wg.Add(1)
			pie.Logger.Println("Connecting to tracker:", tracker.Addr)
			go func() {
				defer wg.Done()
				err := tracker.Connect(ctx, r.Protocol)
				if err == nil {
					r.AddTracker(tracker)
				}
			}()
		} else {
			r.AddAndConnectTracker(ctx, tracker)
		}
	}
	wg.Wait()
}

func (r *Table) GetTracker(id []byte) *Tracker {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if element, exists := r.trackerMap[*(*pie.IDA)(id)]; exists {
		return element.Value.(*Tracker)
	}
	return nil
}

func (r *Table) AddAndConnectTracker(ctx context.Context, tracker *Tracker) {
	r.AddTracker(tracker)
	go func() {
		err := tracker.Connect(ctx, r.Protocol)
		if err != nil {
			r.RemoveTracker(tracker.ID)
		}
	}()
}

func (r *Table) AddTracker(tracker *Tracker) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.trackerList.PushFront(tracker)
	r.trackerMap[*(*pie.IDA)(tracker.ID.Bytes())] = r.trackerList.Front()
	node := r.trackerTree
	for i := 0; i < pie.IDLen; i++ {
		if node.children[tracker.ID.Bit(i)] == nil {
			node.children[tracker.ID.Bit(i)] = &TreeNode{parent: node, depth: i + 1}
		}
		node = node.children[tracker.ID.Bit(i)]
	}
	node.value = tracker
}

func (r *Table) RemoveTracker(id *big.Int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	ida := *(*pie.IDA)(id.Bytes())
	if element, ok := r.trackerMap[ida]; ok {
		r.trackerList.Remove(element)
		delete(r.trackerMap, ida)
		var remove func(*TreeNode, int) bool
		remove = func(node *TreeNode, depth int) bool {
			if node.value != nil {
				node.value = nil
				return true
			}
			if remove(node.children[id.Bit(depth)], depth+1) && node.children[-id.Bit(depth)+1] == nil {
				node.children[id.Bit(depth)] = nil
				return true
			}
			return false
		}
		remove(r.trackerTree, 0)
	}
}

func (r *Table) GetNeighbors(targetIDInt *big.Int, num int, excludeID ...*big.Int) []*Tracker {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	result := make([]*Tracker, 0, num)
	var next func(node *TreeNode)
	next = func(node *TreeNode) {
		if len(result) == num || node == nil {
			return
		}
		if node.value != nil && (len(excludeID) == 0 || node.value.ID.Cmp(excludeID[0]) != 0) {
			result = append(result, node.value)
			return
		}
		next(node.children[targetIDInt.Bit(node.depth)])
		next(node.children[-targetIDInt.Bit(node.depth)+1])
	}
	next(r.trackerTree)
	return result
}

func (t *TreeNode) iterateNeighbors(ctx context.Context, c chan *Tracker, targetIDInt *big.Int) {
	// TODO: lock
	if t.value != nil {
		select {
		case <-ctx.Done():
		case c <- t.value:
		}
		return
	}
	children := append(make([]int, len(t.children)), int(targetIDInt.Bit(t.depth)), -int(targetIDInt.Bit(t.depth))+1)
	for _, child := range children {
		treeNode := t.children[child]
		if treeNode != nil {
			treeNode.iterateNeighbors(ctx, c, targetIDInt)
		}
	}
}
