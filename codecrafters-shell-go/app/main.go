package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"golang.org/x/term"
)

var buildInCommand map[string]Handler
var autoCompletionTrie *Trie
var acRecord map[string]bool

const INVALID_SUFFIX string = "command not found"

func main() {
	//init
	fd := int(os.Stdin.Fd())
	//关闭行缓冲，自动回显和ctrl-c机制
	oldTerminalState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error:%v", err)
		os.Exit(1)
	}
	defer func() {
		_ = term.Restore(fd, oldTerminalState)
		// fmt.Println("已经恢复正常模式")
	}()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)
	go func() {
		<-sigChan
		_ = term.Restore(fd, oldTerminalState)
		// fmt.Println("收到ctrl-c")
		os.Exit(0)
	}()
	//built-in
	initEnv()
	//path
	loadPath()
	//history
	initHistory()
	scanner := bufio.NewReader(os.Stdin)
	//缓存输入字符
	var inputBuf strings.Builder
	initPrompt()
	for {
		char, err := scanner.ReadByte()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(-1)
		}
		switch char {
		//auto-completion
		case '\t':
			prompt := inputBuf.String()
			exist, cmds := autoCompletionTrie.startWith(prompt)
			if !exist {
				fmt.Printf("\x07")
				continue
			}
			//多个匹配
			isLcp := lcp(cmds)
			if tmp, _ := acRecord[prompt]; len(cmds) > 1 && !isLcp && !tmp {
				//第一次触发响铃
				fmt.Printf("\x07")
				acRecord[prompt] = true
				continue
			} else if tmp && !isLcp {
				//第二次触发
				fmt.Printf("\r\n%s\r\n$ %s", strings.Join(cmds, "  "), prompt)
				acRecord[prompt] = false
				continue
			} else if isLcp && len(cmds) > 1 {
				fmt.Printf("\r$ %s", cmds[0])
				inputBuf.Reset()
				inputBuf.WriteString(cmds[0])
				continue
			}
			//只显示第一个
			fmt.Printf("\r$ %s ", cmds[0])
			inputBuf.Reset()
			inputBuf.WriteString(cmds[0])
			inputBuf.WriteByte(' ')
		//submit input
		case '\r', '\n':
			command := strings.TrimSpace(inputBuf.String())
			if len(command) == 0 {
				fmt.Println()
				initPrompt()
				continue
			}
			//移动到下一行
			fmt.Print("\r\n")
			execLine(command, func() {
				_ = term.Restore(fd, oldTerminalState)
			}, func() {
				oldTerminalState, _ = term.MakeRaw(fd)
			})
			initPrompt()
			inputBuf.Reset()
		case 0x03:
			//ctrl-c
			_ = term.Restore(fd, oldTerminalState)
			// fmt.Println("收到ctrl-c")
			os.Exit(0)
		//backspace
		case 0x7f:
			if inputBuf.Len() > 0 {
				past := inputBuf.String()
				len := len(past)
				inputBuf.Reset()
				inputBuf.WriteString(past[:len-1])
				fmt.Printf("\r$ %s ", past[:len-1])
				fmt.Printf("\r$ %s", past[:len-1])
			}
		//up-arrow or down-arrow
		case 0x1b:
			next, _ := scanner.ReadByte()
			if next == '[' {
				arrow, _ := scanner.ReadByte()
				switch arrow {
				case 'A':
					//向上
					inputBuf.Reset()
					//刷新终端
					fmt.Printf("\r%40s", "")
					initPrompt()
					lastRecord := getCursorLine(true)
					fmt.Print(lastRecord)
					inputBuf.WriteString(lastRecord)
				case 'B':
					//向下
					inputBuf.Reset()
					fmt.Printf("\r%40s", "")
					initPrompt()
					nextRecord := getCursorLine(false)
					fmt.Print(nextRecord)
					inputBuf.WriteString(nextRecord)
				}
			} else {
				inputBuf.WriteByte(char)
				fmt.Printf("%c", char)
			}
		default:
			inputBuf.WriteByte(char)
			fmt.Printf("%c", char)
			flushCursor()
		}
	}

}
func initPrompt() {
	fmt.Print("\r$ ")
}
func invalidCommand(command string) {
	fmt.Println(strings.Join([]string{command, INVALID_SUFFIX}, ": "))
}
func initEnv() {
	buildInCommand = make(map[string]Handler)
	buildInCommand["exit"] = &Exit{}
	buildInCommand["echo"] = &Echo{}
	buildInCommand["type"] = &Type{}
	buildInCommand["pwd"] = &Pwd{}
	buildInCommand["cd"] = &Cd{}
	buildInCommand["history"] = &History{}
	autoCompletionTrie = initTrie()
	for _, cmd := range built_in_cmds {
		autoCompletionTrie.addWord(cmd)
	}
	acRecord = make(map[string]bool)
}

func execLine(input string, Recover, Reraw func()) {
	Recover()
	defer Reraw()
	//tmp consists of cmd and args
	pipeline := splitUnit(input)
	cmd, params, output, err := pipeline.getHead()
	defer pipeline.closeOpenFiles()
	defer waiter.Wait()
	if _, valid := buildInCommand[cmd]; valid {
		if err := handlePipe(cmd, params, 1, output, err, os.Stdin, 0); err != nil {
			return
		}
	} else if execProgram, ok := findExternalCommand(cmd); ok {
		if err := handlePipe(execProgram, params, 2, output, err, os.Stdin, 0); err != nil {
			return
		}
	} else {
		handlePipe(cmd, nil, 0, output, err, os.Stdin, 0)
		return
	}
	if !pipeline.hasNext() {
		return
	}
	// var buf [1024]byte
	for pipeline.hasNext() {
		input, cmd, params, output, err, idx := pipeline.next()
		// n, _ := input.Read(buf[:])
		// params = append(params, string(buf[:n]))
		if _, valid := buildInCommand[cmd]; valid {
			if err := handlePipe(cmd, params, 1, output, err, input, idx); err != nil {
				return
			}
		} else if execProgram, ok := findExternalCommand(cmd); ok {
			if err := handlePipe(execProgram, params, 2, output, err, input, idx); err != nil {
				return
			}
		} else {
			handlePipe(cmd, nil, 0, output, err, os.Stdin, idx)
			return
		}
	}

}

/*
status 0:invalid cmd,return the whole input
status 1:built-in cmd,return cmd and params
status 2:external cmd,return fullPath and params
*/
func handlePipe(command string, params []string, status uint8, output, err io.Writer, in io.Reader, idx int) error {
	waiter.Add(1)
	//并发启动
	switch status {
	case 1:
		{
			err := buildInCommand[command].handle(params, output, err)
			tailWork(idx, in, output)
			go monitor(idx)
			return err
		}
	case 2:
		{
			var re_err error
			go func() {
				cmd := exec.Command(command)
				cmd.Args = append([]string{filepath.Base(command)}, params...)
				cmd.Stdin = in
				cmd.Stdout = output
				cmd.Stderr = err
				//标记启动的process
				ctx[idx].process = cmd.Process

				re_err = cmd.Run()
				tailWork(idx, in, output)
			}()
			//monitor
			go monitor(idx)
			return re_err
		}
	default:
		{
			//直接返回了
			invalidCommand(command)
			tailWork(idx, in, output)
			return nil
		}
	}
}

func lcp(strs []string) bool {
	sort.Slice(strs, func(i, j int) bool {
		return len(strs[i]) < len(strs[j])
	})
	for i := 0; i < len(strs)-1; i++ {
		if !strings.HasPrefix(strs[i+1], strs[i]) {
			return false
		}
	}
	return true
}
func monitor(idx int) {
	<-ctx[idx].cnl
	//强行终止进程
	if ctx[idx].process != nil {
		ctx[idx].process.Signal(syscall.SIGTERM)
	}
	rw_mutex.Lock()
	if !ctx[idx].isFinished {
		waiter.Done()
		ctx[idx].isFinished = true
	}
	rw_mutex.Unlock()

}
func tailWork(idx int, in io.Reader, output io.Writer) {

	defer func() {
		//关闭输入流
		if in, yes := in.(*os.File); idx > 0 && yes {
			in.Close()
		}
		//关闭输出流
		if output, yes := output.(*os.File); idx < len(ctx)-1 && yes {
			output.Close()
		}
	}()
	rw_mutex.Lock()
	if !ctx[idx].isFinished {
		waiter.Done()
		ctx[idx].isFinished = true
		//关闭自己的
		close(ctx[idx].cnl)
	}
	rw_mutex.Unlock()
	//通知之前的所有任务
	rw_mutex.RLock()
	for i := 0; i < idx; i++ {
		if !ctx[i].isFinished {
			ctx[i].cnl <- struct{}{}
			close(ctx[i].cnl)
		}
	}
	rw_mutex.RUnlock()
}
