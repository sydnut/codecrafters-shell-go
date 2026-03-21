package main

import (
	"fmt"
	"os"
)

type DirTrie struct {
	trie *Trie
	name string
}

func newFileTrie(file string) (*DirTrie, error) {
	return (&DirTrie{
		trie: initTrie(),
		name: file,
	}).initData()
}
func (dt *DirTrie) initData() (*DirTrie, error) {
	info, err := os.Stat(dt.name)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "dft_1 %v\n", err)
		return dt, err
	}
	entries, err := os.ReadDir(dt.name)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "dft_2 %v\n", err)
		return dt, err
	}
	for _, entry := range entries {
		// fmt.Printf("\ndebug: add_entry:%v\n", entry.Name())
		if entry.IsDir() {
			dt.trie.addWord(entry.Name() + "/")
		} else {
			dt.trie.addWord(entry.Name())
		}

	}
	return dt, err
}
func (dt *DirTrie) startWith(prefix string) (bool, []string) {
	if len(prefix) != 0 {
		return dt.trie.startWith(prefix)
	}
	var ret []string = make([]string, 0)
	for i := range 256 {
		if dt.trie.root[i] != nil {
			_, files := dt.trie.startWith(string(byte(i)))
			for _, file := range files {
				ret = append(ret, file)
			}
		}
	}
	return len(ret) != 0, ret
}

var workingDirTrie *DirTrie
