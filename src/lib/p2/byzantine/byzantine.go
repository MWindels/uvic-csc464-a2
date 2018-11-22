package byzantine

import "time"

type Order uint

const (
	Attack Order = iota
	Retreat
	totalOrders
)

const DefaultOrder Order = Retreat

var Timeout time.Duration = 0	//Defaults to 0, but can be set by the user.

func Majority(orders ...Order) Order {
	sums := make(map[Order]uint)
	for _, o := range orders {
		if s, valid := sums[o]; valid {
			sums[o] = s + 1
		}else{
			sums[o] = 1
		}
	}
	
	oMax := DefaultOrder
	sMax, valid := sums[oMax]
	for o, s := range sums {
		if valid {
			if sMax < s {
				oMax = o
				sMax = s
			}
		}else{
			oMax = o
			sMax = s
			valid = true
		}
	}
	return oMax
}

type General struct {
	id uint
	traitor bool
}

func InitGeneral(me uint, t bool) General {
	return General{id: me, traitor: t}
}

func (g General) ID() uint {
	return g.id
}

func (g General) Traitor() bool {
	return g.traitor
}

type consensusNode struct {
	cmdr uint
	messengers map[uint](chan Order)
	children [](*consensusNode)
}

type ConsensusTree struct {
	root *consensusNode
}

func initConsensusNode(m uint, c uint, ls []uint) *consensusNode {
	msngrs := make(map[uint](chan Order))
	for _, l := range ls {
		msngrs[l] = make(chan Order)
	}
	node := consensusNode{cmdr: c, messengers: msngrs, children: make([](*consensusNode), 0, len(ls))}
	if m > 0 && len(ls) > 1 {
		for i := 0; i < len(ls); i++ {
			lsNext := append([]uint(nil), ls...)
			lsNext = append(lsNext[:i], lsNext[i + 1:]...)
			node.children = append(node.children, initConsensusNode(m - 1, ls[i], lsNext))
		}
	}
	return &node
}

func InitConsensusTree(m uint, totalGenerals uint) ConsensusTree {
	if totalGenerals < 2 {
		panic("Cannot create a consensus tree with less than two generals.")
	}
	
	lieutenants := make([]uint, 0, int(totalGenerals) - 1)
	for i := uint(1); i < totalGenerals; i++ {
		lieutenants = append(lieutenants, i)
	}
	return ConsensusTree{root: initConsensusNode(m, 0, lieutenants)}
}

func send(ch chan Order, o Order) {
	if Timeout > 0 {
		select {
		case ch <- o:
			return
		case <- time.After(Timeout * time.Millisecond):
			return
		}
	}else{
		ch <- o
	}
}

func receive(ch chan Order) Order {
	if Timeout > 0 {
		select {
		case o := <- ch:
			return o
		case <- time.After(Timeout * time.Millisecond):
			return DefaultOrder
		}
	}else{
		return <- ch
	}
}

func (g General) recursiveOM(cn *consensusNode, o Order) (Order, bool) {
	if cn.cmdr != g.id {
		vi := receive(cn.messengers[g.id])
		vj := make([]Order, 0, len(cn.children))
		for _, child := range cn.children {
			if result, valid := g.recursiveOM(child, vi); valid {
				vj = append(vj, result)
			}
		}
		return Majority(append(vj, vi)...), true
	}else{
		if !g.traitor {
			for _, msngr := range cn.messengers {
				send(msngr, o)
			}
		}else{
			for id, msngr := range cn.messengers {
				if id % 2 == 0 {
					send(msngr, (o + 1) % totalOrders)
				}else{
					send(msngr, o)
				}
			}
		}
		return o, false
	}
}

func (g General) OM(ct ConsensusTree) Order {
	if g.id == 0 {
		panic("OM cannot be called by the commanding general.")
	}
	result, _ := g.recursiveOM(ct.root, DefaultOrder)
	return result
}

func (g General) OMLeader(ct ConsensusTree, initialOrder Order) {
	if g.id != 0 {
		panic("OMLeader cannot be called by a subordinate general.")
	}
	g.recursiveOM(ct.root, initialOrder)
}