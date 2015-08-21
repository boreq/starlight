package kbuckets

import (
	"github.com/boreq/netblog/network/node"
	"github.com/boreq/netblog/utils"
)

type sortEntries struct {
	e      []node.NodeInfo
	target node.ID
}

func (s sortEntries) Len() int {
	return len(s.e)
}

func (s sortEntries) Less(i, j int) bool {
	iDis, _ := distance(s.target, s.e[i].Id)
	jDis, _ := distance(s.target, s.e[j].Id)
	cmp, _ := utils.Compare(iDis, jDis)
	return cmp < 0
}

func (s sortEntries) Swap(i, j int) {
	tmp := s.e[i]
	s.e[i] = s.e[j]
	s.e[j] = tmp
}
