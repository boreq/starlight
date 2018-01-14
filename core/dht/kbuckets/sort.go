package kbuckets

import (
	"github.com/boreq/starlight/network/node"
	"github.com/boreq/starlight/utils"
)

type sortEntries struct {
	e      []node.NodeInfo
	target node.ID
}

func (s sortEntries) Len() int {
	return len(s.e)
}

func (s sortEntries) Less(i, j int) bool {
	iDis, _ := node.Distance(s.target, s.e[i].Id)
	jDis, _ := node.Distance(s.target, s.e[j].Id)
	cmp, _ := utils.Compare(iDis, jDis)
	return cmp < 0
}

func (s sortEntries) Swap(i, j int) {
	tmp := s.e[i]
	s.e[i] = s.e[j]
	s.e[j] = tmp
}
