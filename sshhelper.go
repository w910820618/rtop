/*

rtop - the remote system monitoring utility

Copyright (c) 2015-17 RapidLoop

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
	"bytes"
	"time"

	"golang.org/x/crypto/ssh"
)

type Server struct {
	name   string
	client *ssh.Client
}

func sshConnect(val []map[string]interface{}) []*Server {

	servers := make([]*Server, 0, len(val)-1)

	for i := 0; i < len(val); i++ {
		c := val[i]
		auths := []ssh.AuthMethod{ssh.Password(c["password"].(string))}
		config := &ssh.ClientConfig{
			Timeout:         5 * time.Second, //ssh 连接time out 时间一秒钟, 如果ssh验证错误 会在一秒内返回
			User:            c["username"].(string),
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Auth:            auths,
		}

		config.SetDefaults()

		port := "22"

		if _, ok := c["port"]; ok {
			port = c["port"].(string)
		}

		sshClient, err := ssh.Dial("tcp", c["remote"].(string)+":"+port, config)

		if err != nil {
			continue
		}

		server := &Server{
			name:   c["name"].(string),
			client: sshClient,
		}

		servers = append(servers, server)
	}

	return servers
}

func runCommand(client *ssh.Client, command string) (stdout string, err error) {
	session, err := client.NewSession()
	if err != nil {
		//log.Print(err)
		return
	}
	defer session.Close()

	var buf bytes.Buffer
	session.Stdout = &buf
	err = session.Run(command)
	if err != nil {
		//log.Print(err)
		return
	}
	stdout = string(buf.Bytes())

	return
}
