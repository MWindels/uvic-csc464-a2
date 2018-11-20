package byzantine

import "time"

type Order uint

const (
	Attack Order = iota
	Retreat
	totalOrders
)

const DefaultOrder Order = Retreat

var Timeout time.Duration = 3000

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
	messenger chan Order
	ltns []uint
	children [](*consensusNode)
}

type ConsensusTree struct {
	root *consensusNode
}

func initConsensusNode(m uint, c uint, ls []uint) *consensusNode {
	node := consensusNode{cmdr: c, messenger: make(chan Order), ltns: ls, children: make([](*consensusNode), 0, len(ls))}
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

func (cn *consensusNode) send(o Order) {
	if Timeout > 0 {
		select {
		case cn.messenger <- o:
			return
		case <- time.After(Timeout * time.Millisecond):
			return
		}
	}else{
		cn.messenger <- o
	}
}

func (cn *consensusNode) receive() Order {
	if Timeout > 0 {
		select {
		case o := <- cn.messenger:
			return o
		case <- time.After(Timeout * time.Millisecond):
			return DefaultOrder
		}
	}else{
		return <- cn.messenger
	}
}

func (g General) recursiveOM(cn *consensusNode, o Order) (Order, bool) {
	if cn.cmdr != g.id {
		vi := cn.receive()
		vj := make([]Order, 0, len(cn.children))
		for _, child := range cn.children {
			if result, valid := g.recursiveOM(child, vi); valid {
				vj = append(vj, result)
			}
		}
		return Majority(append(vj, vi)...), true
	}else{
		if !g.traitor {
			for range cn.ltns {
				cn.send(o)
			}
		}else{
			for _, ltn := range cn.ltns {
				if ltn % 2 == 0 {
					cn.send((o + 1) % totalOrders)
				}else{
					cn.send(o)
				}
			}
		}
		return DefaultOrder, false
	}
}

func (g General) OM(ct ConsensusTree) (Order, bool) {
	return g.recursiveOM(ct.root, DefaultOrder)
}

func (g General) OMLeader(ct ConsensusTree, initialOrder Order) {
	g.recursiveOM(ct.root, initialOrder)
}