package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const canExec uint = 0o111

var nowDir string = ""
var built_in_cmds []string = []string{
	"exit", "echo", "type", "pwd", "cd", "history", //"xyz_bee", "xyz_bee_rat", "xyz_bee_rat_pig",
}

type Handler interface {
	//todo 暂时忽略内置命令的错误
	handle(params []string, output, err io.Writer) error
}
type Exit struct {
}
type Echo struct {
}
type Type struct {
}
type Pwd struct {
}
type Cd struct {
}

func (*Exit) handle(params []string, output, err io.Writer) error {
	//save
	saveHisToFile()
	os.Exit(0)
	return nil
}
func (*Echo) handle(params []string, output, err io.Writer) error {
	var sb strings.Builder
	for _, str := range params {
		sb.WriteString(str)
		sb.WriteByte(' ')
	}
	len := sb.Len()
	fmt.Fprintln(output, sb.String()[:len-1])
	return nil
}
func (*Type) handle(params []string, output, err io.Writer) error {
	_, ok := buildInCommand[params[0]]
	if ok {
		fmt.Fprintln(output, params[0]+" is a shell builtin")
	} else {
		//consider the situation that it comes from Path env
		if fullPath, ok := findExternalCommand(params[0]); ok {
			fmt.Fprintln(output, params[0]+" is "+fullPath)
		} else {
			fmt.Fprintln(output, params[0]+": not found")
		}
	}
	return nil
}
func (*Pwd) handle(params []string, output, Err io.Writer) error {
	if len(nowDir) == 0 {
		var err error
		nowDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(Err, "%v\n", err)
			return err
		}
	}
	fmt.Fprintln(output, nowDir)
	return nil
}
func (*Cd) handle(path []string, output, Err io.Writer) error {
	//异常
	if len(path[0]) == 0 {
		fmt.Fprintln(Err, "cd: : No such file or directory")
	}
	var cleanPath string = filepath.Clean(path[0])
	//绝对路径
	if path[0][0] == filepath.Separator {
		if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
			fmt.Fprintf(Err, "cd: %s: No such file or directory\n", cleanPath)
		} else {
			nowDir = cleanPath
		}
		return nil
	}
	//用户目录开始(也是绝对路径)
	if path[0][0] == '~' {
		hd, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return nil
		}
		fullPath := filepath.Clean(strings.Join([]string{hd, path[0][1:]}, string(filepath.Separator)))
		cleanPath = strings.ReplaceAll(fullPath, "~", hd)
		if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
			fmt.Fprintf(Err, "cd: %s: No such file or directory\n", cleanPath)
		} else {
			nowDir = cleanPath
		}
		return nil
	}
	//相对路径
	cleanPath = filepath.Clean(strings.Join([]string{nowDir, path[0]}, string(filepath.Separator)))
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		fmt.Fprintf(Err, "cd: %s: No such file or directory\n", cleanPath)
	} else {
		nowDir = cleanPath
	}
	return nil
}
