package monitor

import "stormsim/internal/core/gnbcontext"

type GnbGroupType int

const (
	GroupTypeGround GnbGroupType = iota
	GroupTypeNTN
)

func (t GnbGroupType) String() string {
	switch t {
	case GroupTypeGround:
		return "ground"
	case GroupTypeNTN:
		return "ntn"
	default:
		return "unknown"
	}
}

type NetworkCondition struct {
	PacketLoss float64
	LatencyMs  int
	JitterMs   int
}

type GnbGroup struct {
	Name             string
	Type             GnbGroupType
	Gnbs             []*gnbcontext.GnbContext
	NetworkCondition *NetworkCondition
}

func NewGnbGroup(name string, groupType GnbGroupType) *GnbGroup {
	return &GnbGroup{
		Name: name,
		Type: groupType,
		Gnbs: make([]*gnbcontext.GnbContext, 0),
		NetworkCondition: &NetworkCondition{
			PacketLoss: 0,
			LatencyMs:  0,
			JitterMs:   0,
		},
	}
}

func (g *GnbGroup) AddGnb(gnb *gnbcontext.GnbContext) {
	g.Gnbs = append(g.Gnbs, gnb)
}

func (g *GnbGroup) RemoveGnb(gnbId string) {
	for i, gnb := range g.Gnbs {
		if gnb.GetId() == gnbId {
			g.Gnbs = append(g.Gnbs[:i], g.Gnbs[i+1:]...)
			return
		}
	}
}

func (g *GnbGroup) GetGnb(gnbId string) *gnbcontext.GnbContext {
	for _, gnb := range g.Gnbs {
		if gnb.GetId() == gnbId {
			return gnb
		}
	}
	return nil
}

func (g *GnbGroup) Size() int {
	return len(g.Gnbs)
}

func (g *GnbGroup) SetNetworkCondition(condition *NetworkCondition) {
	g.NetworkCondition = condition
}

func (g *GnbGroup) GetNetworkCondition() *NetworkCondition {
	return g.NetworkCondition
}

func (g *GnbGroup) GetGnbIds() []string {
	ids := make([]string, len(g.Gnbs))
	for i, gnb := range g.Gnbs {
		ids[i] = gnb.GetId()
	}
	return ids
}
