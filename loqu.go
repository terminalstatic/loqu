package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/peterh/liner"
	"github.com/terminalstatic/loqu/lib"
)

var (
	history_fn = filepath.Join(os.TempDir(), ".loqu_history")
	commands   = []string{"add", "modify", "activate", "switch", "deactivate", "remove", "status"}
)

type Nodes struct {
	NodeMap map[string]*lib.Node
}

func (nds Nodes) GetActive() []*lib.Node {
	nodes := make([]*lib.Node, 0)
	for _, v := range nds.NodeMap {
		if v.Active {
			nodes = append(nodes, v)
		}
	}
	return nodes
}

var nms = Nodes{}

func init() {
	nms.NodeMap = make(map[string]*lib.Node)
}

func main() {

	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(false)

	line.SetCompleter(func(line string) (c []string) {
		for _, n := range commands {
			if strings.HasPrefix(n, strings.ToLower(line)) {
				c = append(c, n)
			}
		}
		return
	})

	if f, err := os.Open(history_fn); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	for {
		if cmd, err := line.Prompt("> "); err == nil {
			ws := regexp.MustCompile(`\s+`)
			cmd = ws.ReplaceAllString(cmd, " ")
			tokens := strings.Split(cmd, " ")
			//fmt.Println(cmd)
			switch tokens[0] {
			case commands[0]:
				addCmd(tokens)
			case commands[1]:
				modifyCmd(tokens)
			case commands[2]:
				activateCmd(tokens)
			case commands[3]:
				switchCmd(tokens)
			case commands[4]:
				deactivateCmd(tokens)
			case commands[5]:
				removeCmd(tokens)
			case commands[6]:
				statusCmd()
			default:
				fmt.Println("Invalid command")
			}
			line.AppendHistory(cmd)
		} else {
			if f, err := os.Create(history_fn); err != nil {
				fmt.Printf("Error writing history file: %s\n", err)
			} else {
				line.WriteHistory(f)
				f.Close()
			}
			fmt.Println("Bye")
			break
		}

	}
}

func addCmd(tokens []string) {
	if len(tokens) != 5 {
		fmt.Println("Invalid input: add [Name] [Host] [DstUrl] [HealthPath]")
		return
	}

	if _, ok := nms.NodeMap[tokens[1]]; ok {
		fmt.Println(fmt.Sprintf("Node %s already exists, add node failed", tokens[1]))
		return
	}

	probeURL := fmt.Sprintf("%s%s", tokens[3], tokens[4])
	if lib.ProbeHttp(probeURL) != nil {
		fmt.Println(fmt.Sprintf("Health check on %s failed, add node failed", probeURL))
		return
	}

	if !lib.IsValidHost(tokens[2]) {
		fmt.Println(fmt.Sprintf("%s is not a valid local host, add node failed", tokens[2]))
		return
	}

	if !(lib.ContainsHost(nms.GetActive(), tokens[2]) || lib.ProbeTcp(tokens[2]) == nil) {
		fmt.Println(fmt.Sprintf("Host %s already in use, add node failed", tokens[2]))
		return
	}

	newNode := &lib.Node{Host: tokens[2], DestURL: tokens[3], HealthPath: tokens[4]}

	nms.NodeMap[tokens[1]] = newNode

	fmt.Printf(`Node added: %s => {Host: %s, DstUrl: "%s", HealthPath: "%s", Active: %t}`+"\n",
		tokens[1], nms.NodeMap[tokens[1]].Host, nms.NodeMap[tokens[1]].DestURL, nms.NodeMap[tokens[1]].HealthPath, nms.NodeMap[tokens[1]].Active)

}

func modifyCmd(tokens []string) {
	if len(tokens) != 5 {
		fmt.Println("Invalid input: modify [Name] [Host] [DstUrl] [HealthPath]")
		return
	}

	if _, ok := nms.NodeMap[tokens[1]]; !ok {
		fmt.Println(fmt.Sprintf("Node %s does not exist, node not modified", tokens[1]))
		return
	}

	if nms.NodeMap[tokens[1]].Active {
		fmt.Println(fmt.Sprintf("Node %s is currently active, node not modified", tokens[1]))
		return
	}

	probeURL := fmt.Sprintf("%s%s", tokens[3], tokens[4])
	if lib.ProbeHttp(probeURL) != nil {
		fmt.Println(fmt.Sprintf("Health check on %s failed, modify node failed", probeURL))
		return
	}

	if !lib.IsValidHost(tokens[2]) {
		fmt.Println(fmt.Sprintf("%s is not a valid local host, modify node failed", tokens[2]))
		return
	}

	if lib.ContainsHost(nms.GetActive(), tokens[2]) || lib.ProbeTcp(tokens[2]) == nil {
		fmt.Println(fmt.Sprintf("Host %s already in use, modify node failed", tokens[2]))
		return
	}

	newNode := &lib.Node{Host: tokens[2], DestURL: tokens[3], HealthPath: tokens[4]}

	nms.NodeMap[tokens[1]] = newNode

	fmt.Printf(`Node modified: %s => {Host: %s, DstUrl: "%s", HealthPath: "%s", Active: %t}`+"\n",
		tokens[1], nms.NodeMap[tokens[1]].Host, nms.NodeMap[tokens[1]].DestURL, nms.NodeMap[tokens[1]].HealthPath, nms.NodeMap[tokens[1]].Active)
}

func activateCmd(tokens []string) {
	if len(tokens) != 2 {
		fmt.Println("Invalid input: activate [Name]")
		return
	}
	if _, ok := nms.NodeMap[tokens[1]]; !ok {
		fmt.Println(fmt.Sprintf("Node %s does not exist, node not activated", tokens[1]))
		return
	}

	if lib.ContainsHost(nms.GetActive(), nms.NodeMap[tokens[1]].Host) {
		fmt.Println(fmt.Sprintf("Node with same host already active, node %s not activated", tokens[1]))
		return
	}

	tmp := nms.NodeMap[tokens[1]]

	tmp.Active = true

	nms.NodeMap[tokens[1]] = tmp

	go tmp.Serve()
	fmt.Println(fmt.Sprintf("Node %s now set to active", tokens[1]))

}

func switchCmd(tokens []string) {
	shutdown := false
	if len(tokens) != 3 {
		fmt.Println("Invalid input: activate [NameFrom] [NameTo]")
		return
	} else if _, ok := nms.NodeMap[tokens[1]]; !ok {
		fmt.Println(fmt.Sprintf("Node %s does not exist, node not activated", tokens[1]))
		return
	} else if _, ok := nms.NodeMap[tokens[2]]; !ok {
		fmt.Println(fmt.Sprintf("Node %s does not exist, node not activated", tokens[1]))
		return
	} else if !nms.NodeMap[tokens[1]].Active {
		fmt.Println(fmt.Sprintf("Node %s is currently not active, switching from inactive node not possible", tokens[1]))
		return
	} else if nms.NodeMap[tokens[1]].Host != nms.NodeMap[tokens[2]].Host {
		fmt.Print(fmt.Sprintf("Node %s host(%s) is different from node %s host(%s), are you sure [Y/n]? ",
			tokens[1], nms.NodeMap[tokens[1]].Host, tokens[2], nms.NodeMap[tokens[2]].Host))

		var buffer [1]byte
		os.Stdin.Read(buffer[:])
		fmt.Println()
		if string(buffer[0]) == "Y" {
			shutdown = true
		} else {
			fmt.Println("Switching aborted")
			return
		}

	}

	if shutdown {
		go nms.NodeMap[tokens[1]].ShutdownAndServe(nms.NodeMap[tokens[2]])
	} else {
		nms.NodeMap[tokens[1]].SwitchTo(nms.NodeMap[tokens[2]])
	}

	fmt.Println(fmt.Sprintf("Node %s now set to active", tokens[2]))
}

func deactivateCmd(tokens []string) {
	if len(tokens) != 2 {
		fmt.Println("Invalid input: deactivate [Name]")
		return
	}
	if _, ok := nms.NodeMap[tokens[1]]; !ok {
		fmt.Println(fmt.Sprintf("Node %s does not exist, node not deactivated", tokens[1]))
		return
	}

	tmp := nms.NodeMap[tokens[1]]

	tmp.Active = false

	nms.NodeMap[tokens[1]] = tmp

	tmp.Shutdown()

	fmt.Println(fmt.Sprintf("Node %s now deactivated", tokens[1]))

}

func removeCmd(tokens []string) {
	if len(tokens) != 2 {
		fmt.Println("Invalid input: remove [Name]")
		return
	}
	if _, ok := nms.NodeMap[tokens[1]]; !ok {
		fmt.Println(fmt.Sprintf("Node %s does not exist, node not removed", tokens[1]))
		return
	}

	if nms.NodeMap[tokens[1]].Active {
		fmt.Println(fmt.Sprintf("Node %s is currently active, node not removed", tokens[1]))
		return
	}

	delete(nms.NodeMap, tokens[1])

	fmt.Println(fmt.Sprintf("Node %s removed", tokens[1]))

}

func statusCmd() {
	for k, v := range nms.NodeMap {
		fmt.Printf(`%s => {Host: %s, DstUrl: "%s", HealthPath: "%s", Active: %t, Reqs: %d}`+"\n", k, v.Host, v.DestURL, v.HealthPath, v.Active, v.ReqCount)
	}
}
