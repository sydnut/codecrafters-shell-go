package main

//load the PATH into choices
import (
	// "fmt"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const PermBits = 0644

// 存储所有的可执行命令与完整路径的映射
var execCmdMap map[string]string
var pathList []string

func loadPath() {
	execCmdMap = make(map[string]string)
	pathList = make([]string, 0)
	//获取环境变量路径目录
	path, ok := os.LookupEnv("PATH")
	if !ok || len(path) == 0 {
		return
	}
	dirs := strings.SplitSeq(path, string(os.PathListSeparator))
	//校验存在与否并保存
	for dir := range dirs {
		//exist
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			pathList = append(pathList, dir)
		}
	}
	for _, dir := range pathList {
		if info, _ := os.Stat(dir); info.IsDir() {
			files, err := os.ReadDir(dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			} else {
				for _, file := range files {
					if file.IsDir() {
						continue
					}
					fullPath := filepath.Join(dir, file.Name())
					//exist and executable
					if fileInfo, _ := os.Stat(fullPath); (uint(fileInfo.Mode().Perm()) & canExec) != 0 {
						if _, e := execCmdMap[file.Name()]; e {
							continue
						}
						execCmdMap[file.Name()] = fullPath
						autoCompletionTrie.addWord(file.Name())
					}
				}
			}
		} else {
			//a file
			if (uint(info.Mode().Perm()) & canExec) != 0 {
				if _, e := execCmdMap[info.Name()]; e {
					continue
				}
				execCmdMap[info.Name()] = dir
				autoCompletionTrie.addWord(info.Name())
			}
		}
	}
}
func findExternalCommand(command string) (string, bool) {
	fullPath, exist := execCmdMap[command]
	if exist {
		return fullPath, true
	}
	return "", false
}

func execExternalCommand(command string, params []string, output, errout io.Writer, in io.Reader) error {
	cmd := exec.Command(command)
	cmd.Args = append([]string{filepath.Base(command)}, params...)
	cmd.Stdin = in
	cmd.Stdout = output
	cmd.Stderr = errout
	err := cmd.Run()
	return err
}

// spilt units from params with space or single quote
// 遇到重定向字符和管道符直接break loop,在循环外单独处理(后续命令名和文件名简单好处理)
func splitUnit(params string) *pipeline {
	var pl *pipeline
	var res []string = make([]string, 0)
	var buf strings.Builder
	var (
		hasSinQuote   bool = false
		hasDouQuote   bool = false
		hasMoreCmds   bool = false
		hasRedirect   bool = false
		isOutRedirect bool = true
		isAppend      bool = false
	)
	//记录history
	addHistoryRecord(params)
	for i := 0; i < len(params); {
		//没遇到
		if !hasSinQuote && !hasDouQuote {
			switch params[i] {
			case ' ':
				if buf.Len() > 0 {
					res = append(res, buf.String())
					buf.Reset()
				}
				for i < len(params) && params[i] == ' ' {
					i++
				}
			case '\'':
				hasSinQuote = true
				i++
			case '"':
				hasDouQuote = true
				i++
			//consider '\'
			case '\\':
				buf.WriteByte(params[i+1])
				i += 2
			//consider pipe
			case '|':
				//初始化pl
				if !hasMoreCmds {
					//初始化pl
					if buf.Len() > 0 {
						res = append(res, buf.String())
						buf.Reset()
					}
					pl = initPipeline(res[0], getValidParams(res))
					//clear
					res = make([]string, 0)
				} else {
					if buf.Len() > 0 {
						res = append(res, buf.String())
						buf.Reset()
					}
					pl.addNext(res[0], getValidParams(res))
					//clear
					res = make([]string, 0)
				}

				i++
				hasMoreCmds = true
			//consider redirection
			case '>':
				isAppend = i+1 < len(params) && params[i+1] == '>'
				//初始化pl
				if !hasMoreCmds {
					//初始化pl
					if buf.Len() > 0 {
						res = append(res, buf.String())
						buf.Reset()
					}
					pl = initPipeline(res[0], getValidParams(res))
				} else {
					if buf.Len() > 0 {
						res = append(res, buf.String())
						buf.Reset()
					}
					pl.addNext(res[0], getValidParams(res))
					//clear
					res = make([]string, 0)
				}
				hasRedirect = true
				hasMoreCmds = true
				if isAppend {
					i += 2
				} else {
					i++
				}
			case '1':
				if i+1 < len(params) && params[i+1] == '>' {
					i++
					isAppend = i+1 < len(params) && params[i+1] == '>'
					//初始化pl
					if !hasMoreCmds {
						//初始化pl
						if buf.Len() > 0 {
							res = append(res, buf.String())
							buf.Reset()
						}
						pl = initPipeline(res[0], getValidParams(res))
					} else {
						if buf.Len() > 0 {
							res = append(res, buf.String())
							buf.Reset()
						}
						pl.addNext(res[0], getValidParams(res))
						//clear
						res = make([]string, 0)
					}
					hasRedirect = true
					hasMoreCmds = true
					if isAppend {
						i += 2
					} else {
						i++
					}
				} else {
					buf.WriteByte(params[i])
					i++
				}
			case '2':
				if i+1 < len(params) && params[i+1] == '>' {
					i++
					isAppend = i+1 < len(params) && params[i+1] == '>'
					//初始化pl
					if !hasMoreCmds {
						//初始化pl
						if buf.Len() > 0 {
							res = append(res, buf.String())
							buf.Reset()
						}
						pl = initPipeline(res[0], getValidParams(res))
					} else {
						if buf.Len() > 0 {
							res = append(res, buf.String())
							buf.Reset()
						}
						pl.addNext(res[0], getValidParams(res))
						//clear
						res = make([]string, 0)
					}
					hasRedirect = true
					hasMoreCmds = true
					isOutRedirect = false
					if isAppend {
						i += 2
					} else {
						i++
					}
				} else {
					buf.WriteByte(params[i])
					i++
				}
			default:
				buf.WriteByte(params[i])
				i++
			}
		} else if hasSinQuote {
			if params[i] == '\'' {
				hasSinQuote = false
			} else {
				buf.WriteByte(params[i])
			}
			i++
		} else if hasDouQuote {
			if params[i] == '"' {
				hasDouQuote = false
			} else if params[i] != '\\' {
				buf.WriteByte(params[i])
			} else {
				// params[i]==\
				if i < len(params) && ((params[i+1] == '"') || (params[i+1] == '\\')) {
					buf.WriteByte(params[i+1])
					i++
				} else {
					buf.WriteByte(params[i])
				}
			}
			i++
		}
	}
	//初始化
	if buf.Len() > 0 && !hasMoreCmds {
		res = append(res, buf.String())
		pl = initPipeline(res[0], getValidParams(res))
	} else if buf.Len() > 0 && hasMoreCmds && !hasRedirect {
		res = append(res, buf.String())
		pl.addNext(res[0], getValidParams(res))
	}
	//重定向
	if !hasRedirect {
		pl.finish(os.Stdout, os.Stderr)
	} else {
		filename := buf.String()
		var file *os.File
		var flag int = os.O_CREATE | os.O_WRONLY
		if isAppend {
			flag |= os.O_APPEND
		} else {
			//非追加，清空
			flag |= os.O_TRUNC
		}
		file, err := os.OpenFile(filename, flag, PermBits)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return nil
		}
		if isOutRedirect {
			pl.finish(file, os.Stderr)
		} else {
			pl.finish(os.Stdout, file)
		}
	}
	return pl
}

//the funcs down are the supporting funcs which is make it readable

func getValidParams(res []string) []string {
	if len(res) == 1 {
		return make([]string, 0)
	} else {
		return res[1:]
	}
}
