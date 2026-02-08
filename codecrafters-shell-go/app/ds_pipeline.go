package main

import (
	"fmt"
	"io"
	"os"
	"sync"
)

var waiter sync.WaitGroup

type channel struct {
	cnl        chan struct{}
	isFinished bool
	process    *os.Process
}

var rw_mutex sync.RWMutex
var ctx []*channel

type headCmd struct {
	command     string
	params      []string
	output, err io.Writer
	next        *nodeCmd
}
type nodeCmd struct {
	input       io.Reader
	command     string
	params      []string
	output, err io.Writer
	next        *nodeCmd
	idx         int
}
type pipeline struct {
	line *headCmd
	cnt  *nodeCmd
}

func initPipeline(command string, params []string) *pipeline {
	ctx = make([]*channel, 0)
	ctx = append(ctx, &channel{
		cnl:        make(chan struct{}),
		isFinished: false,
	})
	return &pipeline{
		line: &headCmd{
			command: command,
			params:  params,
			output:  nil,
			next:    nil,
			err:     os.Stderr,
		},
	}
}
func (pipe *pipeline) finish(out, err io.Writer) *pipeline {
	if pipe.line.next == nil {
		pipe.line.output, pipe.line.err = out, err
		return pipe
	}
	ptr := pipe.line.next
	for ptr.next != nil {
		ptr = ptr.next
	}
	ptr.output, ptr.err = out, err
	return pipe
}
func (pipe *pipeline) addNext(cmd string, params []string) *pipeline {
	r, w, err := os.Pipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "add_next:%v\n", err)
		return nil
	}
	ctx = append(ctx, &channel{
		cnl:        make(chan struct{}),
		isFinished: false,
	})
	ptr := pipe.line.next
	if ptr == nil {
		pipe.line.output = w
		pipe.line.next = &nodeCmd{
			input:   r,
			command: cmd,
			params:  params,
			err:     os.Stderr,
			idx:     1,
		}
		//记录第一个node
		if pipe.cnt == nil {
			pipe.cnt = pipe.line.next
		}
		return pipe
	}
	for ptr.next != nil {
		ptr = ptr.next
	}
	ptr.output = w
	ptr.next = &nodeCmd{
		input:   r,
		command: cmd,
		params:  params,
		err:     os.Stderr,
		idx:     ptr.idx + 1,
	}
	return pipe
}
func (pipe *pipeline) getHead() (
	command string,
	params []string,
	output, err io.Writer,
) {
	head := pipe.line
	return head.command, head.params, head.output, head.err
}
func (pipe *pipeline) hasNext() bool {
	return pipe.cnt != nil
}
func (pipe *pipeline) next() (
	input io.Reader,
	command string,
	params []string,
	output, err io.Writer,
	idx int,
) {
	node := pipe.cnt
	pipe.cnt = pipe.cnt.next
	return node.input, node.command, node.params, node.output, node.err, node.idx
}
func (pipe *pipeline) peek() {
	fmt.Printf("pipe:%v\n", *pipe)
	if pipe.line != nil {
		fmt.Printf("head:%v\n", *(pipe.line))
		for pipe.hasNext() {
			a, b, c, d, e, f := pipe.next()
			fmt.Printf("node input:%v,cmd:%v,params:%v,output:%v,err:%v,idx:%d\n", a, b, c, d, e, f)
		}
	}
}

/*
close the redirect file if happens
*/
func (pl *pipeline) closeOpenFiles() {
	var (
		output io.Writer
		err    io.Writer
	)
	if pl.line.next == nil {
		output, err = pl.line.output, pl.line.err
	} else {
		ptr := pl.line.next
		for ptr.next != nil {
			ptr = ptr.next
		}
		output, err = ptr.output, ptr.err
	}
	if file, ok := output.(*os.File); ok {
		if file.Fd() != os.Stdout.Fd() {
			defer file.Close()
		}
	}
	if file, ok := err.(*os.File); ok {
		if file.Fd() != os.Stderr.Fd() {
			defer file.Close()
		}
	}
}
