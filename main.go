/*

rtop - the remote system monitoring utility

Copyright (c) 2015 RapidLoop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"
)

const VERSION = "2.0"
const DEFAULT_REFRESH = 5 // default refresh interval in seconds

func usage(code int) {
	fmt.Printf(
		`rtop %s - (c) 2020 RapidLoop - MIT Licensed - http://rtop-monitor.org
rtop monitors server statistics over an ssh connection

Usage: rtop [-i private-key-file] [user@]host[:port] [interval]

	-i configre.json
		The directory where the rtop configuration file is located

`, VERSION)
	os.Exit(code)
}

var filePath = flag.String("i", "", "The path of the configuration file")

func main() {
	flag.Parse()

	var val []map[string]interface{}

	log.SetPrefix("rtop: ")
	log.SetFlags(0)

	if !exists(*filePath) {
		log.Println("The configuration file does not exist, rtop exits")
		os.Exit(0)
	}

	content, err := ioutil.ReadFile(*filePath)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(content, &val); err != nil {
		panic(err)
	}

	if !parseSshConfig(val) {
		os.Exit(0)
	}

	interval := DEFAULT_REFRESH * time.Second

	servers := sshConnect(val)

	output := getOutput()
	// the loop
	showStats(output, servers)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	timer := time.Tick(interval)
	done := false
	for !done {
		select {
		case <-sig:
			done = true
			fmt.Println()
		case <-timer:
			showStats(output, servers)
		}
	}
}

func showStats(output io.Writer, servers []*Server) {
	stats := Stats{}

	for i := 0; i < len(servers); i++ {
		getAllStats(servers[i].client, &stats)
		clearConsole()
		used := stats.MemTotal - stats.MemFree - stats.MemBuffers - stats.MemCached
		fmt.Println(servers[i].name)
		fmt.Fprintf(output,
			`%s%s%s%s up %s%s%s

Load:
    %s%s %s %s%s

CPU:
    %s%.2f%s%% user, %s%.2f%s%% sys, %s%.2f%s%% nice, %s%.2f%s%% idle, %s%.2f%s%% iowait, %s%.2f%s%% hardirq, %s%.2f%s%% softirq, %s%.2f%s%% guest

Processes:
    %s%s%s running of %s%s%s total

Memory:
    free    = %s%s%s
    used    = %s%s%s
    buffers = %s%s%s
    cached  = %s%s%s
    swap    = %s%s%s free of %s%s%s

`,
			escClear,
			escBrightWhite, stats.Hostname, escReset,
			escBrightWhite, fmtUptime(&stats), escReset,
			escBrightWhite, stats.Load1, stats.Load5, stats.Load10, escReset,
			escBrightWhite, stats.CPU.User, escReset,
			escBrightWhite, stats.CPU.System, escReset,
			escBrightWhite, stats.CPU.Nice, escReset,
			escBrightWhite, stats.CPU.Idle, escReset,
			escBrightWhite, stats.CPU.Iowait, escReset,
			escBrightWhite, stats.CPU.Irq, escReset,
			escBrightWhite, stats.CPU.SoftIrq, escReset,
			escBrightWhite, stats.CPU.Guest, escReset,
			escBrightWhite, stats.RunningProcs, escReset,
			escBrightWhite, stats.TotalProcs, escReset,
			escBrightWhite, fmtBytes(stats.MemFree), escReset,
			escBrightWhite, fmtBytes(used), escReset,
			escBrightWhite, fmtBytes(stats.MemBuffers), escReset,
			escBrightWhite, fmtBytes(stats.MemCached), escReset,
			escBrightWhite, fmtBytes(stats.SwapFree), escReset,
			escBrightWhite, fmtBytes(stats.SwapTotal), escReset,
		)
		if len(stats.FSInfos) > 0 {
			fmt.Println("Filesystems:")
			for _, fs := range stats.FSInfos {
				fmt.Fprintf(output, "    %s%8s%s: %s%s%s free of %s%s%s\n",
					escBrightWhite, fs.MountPoint, escReset,
					escBrightWhite, fmtBytes(fs.Free), escReset,
					escBrightWhite, fmtBytes(fs.Used+fs.Free), escReset,
				)
			}
			fmt.Println()
		}
		if len(stats.NetIntf) > 0 {
			fmt.Println("Network Interfaces:")
			keys := make([]string, 0, len(stats.NetIntf))
			for intf := range stats.NetIntf {
				keys = append(keys, intf)
			}
			sort.Strings(keys)
			for _, intf := range keys {
				info := stats.NetIntf[intf]
				fmt.Fprintf(output, "    %s%s%s - %s%s%s",
					escBrightWhite, intf, escReset,
					escBrightWhite, info.IPv4, escReset,
				)
				if len(info.IPv6) > 0 {
					fmt.Fprintf(output, ", %s%s%s\n",
						escBrightWhite, info.IPv6, escReset,
					)
				} else {
					fmt.Fprintf(output, "\n")
				}
				fmt.Fprintf(output, "      rx = %s%s%s, tx = %s%s%s\n",
					escBrightWhite, fmtBytes(info.Rx), escReset,
					escBrightWhite, fmtBytes(info.Tx), escReset,
				)
				fmt.Println()
			}
			fmt.Println()
		}
	}

}

const (
	escClear       = "\033[H\033[2J"
	escRed         = "\033[31m"
	escReset       = "\033[0m"
	escBrightWhite = "\033[37;1m"
)

func exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
