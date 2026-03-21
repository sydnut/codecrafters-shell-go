package main

type t_node struct {
	char     byte
	children [256]*t_node
	flag     bool
}
type Trie struct {
	root [256]*t_node
}

func initTrie() *Trie {
	return &Trie{
		root: [256]*t_node{},
	}
}
func (trie *Trie) addWord(word string) {
	if len(word) == 0 {
		return
	}
	if trie.root[word[0]] == nil {
		trie.root[word[0]] = doAddWord(word, 0)
		return
	}
	//考虑一个字符情况
	if len(word) == 1 {
		trie.root[word[0]].flag = true
		return
	}
	cnt := 1
	pre, ptr := trie.root[word[0]], trie.root[word[0]].children[word[cnt]]
	for cnt+1 < len(word) && ptr != nil {
		cnt++
		pre = ptr
		ptr = ptr.children[word[cnt]]
	}
	if cnt == len(word)-1 && ptr != nil {
		ptr.flag = true
		return
	}
	pre.children[word[cnt]] = doAddWord(word, cnt)
}
func doAddWord(word string, idx int) *t_node {
	if idx >= len(word) {
		return nil
	}
	var char byte = word[idx]
	newNode := &t_node{
		char:     char,
		children: [256]*t_node{},
		flag:     false,
	}
	if idx+1 < len(word) {
		newNode.children[word[idx+1]] = doAddWord(word, idx+1)
	} else {
		newNode.flag = true
	}
	return newNode
}
func (trie *Trie) startWith(prefix string) (bool, []string) {
	if len(prefix) == 0 {
		return false, nil
	}
	var ret_set = make([]string, 0)
	if trie.root[prefix[0]] == nil {
		return false, nil
	}
	//next idx of the next char
	idx := 1
	var ptr *t_node = trie.root[prefix[0]]
	for idx < len(prefix) && ptr != nil {
		ptr = ptr.children[prefix[idx]]
		idx++
	}
	//匹配失败
	if idx < len(prefix) || ptr == nil {
		return false, nil
	}
	if ptr.flag {
		ret_set = append(ret_set, prefix)
	}
	//完美匹配且可能有剩余
	for _, p := range ptr.children {
		if p != nil {
			search(&ret_set, p, prefix)
		}
	}
	return true, ret_set
}
func search(set *[]string, nowNode *t_node, prefix string) {
	oneAns := prefix + string(nowNode.char)
	if nowNode.flag {
		*set = append(*set, oneAns)
	}
	for _, p := range nowNode.children {
		if p != nil {
			search(set, p, oneAns)
		}
	}
}
