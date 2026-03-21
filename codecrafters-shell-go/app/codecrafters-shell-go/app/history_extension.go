package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

var historyRecords []string = make([]string, 0)
var cursor int = -1

type History struct {
}

func initHistory() {
	histFile, ok := os.LookupEnv("HISTFILE")
	if !ok || len(histFile) == 0 {
		return
	}
	file, err := os.OpenFile(histFile, os.O_RDONLY, PermBits)
	if err != nil {
		return
	}
	defer file.Close()
	buf := [1024]byte{}
	n, _ := file.Read(buf[:])
	loadFileToMem(buf[:n])
}
func saveHisToFile() {
	histFile, ok := os.LookupEnv("HISTFILE")
	if !ok || len(histFile) == 0 {
		return
	}
	file, err := os.OpenFile(histFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, PermBits)
	if err != nil {
		return
	}
	defer file.Close()
	writeToFile(file)
}
func (*History) handle(params []string, output, err io.Writer) error {
	if len(params) == 0 {
		for idx, record := range historyRecords {
			fmt.Fprintf(output, "%5d  %s\n", idx+1, record)
		}
		return nil
	} else if len(params) == 1 {
		limit, Err := strconv.Atoi(params[0])
		if Err != nil {
			fmt.Fprintf(err, "invalid input\n")
			return Err
		}
		//assume limit >= 0
		for i := max(len(historyRecords)-limit, 0); i < len(historyRecords); i++ {
			fmt.Fprintf(output, "%5d  %s\n", i+1, historyRecords[i])
		}
		return nil
	} else {
		for i := 0; i < len(params); {
			if params[i][0] == '-' && i < len(params)-1 {
				filename := params[i+1]
				switch params[i][1] {
				case 'r':
					file, Err := os.OpenFile(filename, os.O_RDONLY, PermBits)
					defer file.Close()
					if Err != nil {
						fmt.Fprintf(err, "invalid file\n")
						return Err
					}
					var buf [1024]byte
					n, _ := file.Read(buf[:])
					loadFileToMem(buf[:n])
				case 'w':
					file, Err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, PermBits)
					defer file.Close()
					if Err != nil {
						fmt.Fprintf(err, "invalid file\n")
						return Err
					}
					if err := writeToFile(file); err != nil {
						return err
					}
				case 'a':
					file, Err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, PermBits)
					defer file.Close()
					if Err != nil {
						fmt.Fprintf(err, "invalid file\n")
						return Err
					}
					if err := writeToFile(file); err != nil {
						return err
					}
					clearHistory()
				default:
				}
				i += 2
			}
		}
		return nil
	}
}
func addHistoryRecord(record string) {
	historyRecords = append(historyRecords, record)
}
func getCursorLine(signal bool) string {
	if signal {
		//向上
		//未初始化
		if cursor == -1 {
			cursor = len(historyRecords) - 1
			return historyRecords[cursor]
		}
		cursor--
		if cursor < 0 {
			cursor = 0
		}
		return historyRecords[cursor]
	} else {
		if cursor == -1 {
			//暂时不考虑特殊情况
			return ""
		}
		cursor++
		if cursor >= len(historyRecords) {
			cursor = len(historyRecords) - 1
		}
		return historyRecords[cursor]
	}
}
func flushCursor() {
	cursor = -1
}
func loadFileToMem(buf []byte) {
	str := string(buf)
	strings.TrimSpace(str)
	for _, record := range strings.Split(str, "\n") {
		record = strings.TrimSpace(record)
		if len(record) > 0 {
			addHistoryRecord(record)
		}
	}
}
func writeToFile(file *os.File) error {
	var sb strings.Builder
	for _, record := range historyRecords {
		sb.WriteString(record)
		sb.WriteByte('\n')
	}
	output := sb.String()
	_, err := file.Write([]byte(output))
	return err
}
func clearHistory() {
	historyRecords = make([]string, 0)
	cursor = -1
}
