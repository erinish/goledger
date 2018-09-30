package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

// Task holds an individual work record
type Task struct {
	Desc   string
	Opened int64
	Closed int64
	TaskID string
}

func genTaskID() string {
	rand.Seed(time.Now().UnixNano())
	h := sha1.New()
	h.Write([]byte(string(rand.Intn(1000000))))
	taskid := fmt.Sprintf("%x", h.Sum(nil))
	return taskid
}

func config() {
	curUser, err := user.Current()
	if err != nil {
		fmt.Println("unknown user")
		os.Exit(1)
	}
	workDir := filepath.Join(curUser.HomeDir, ".goledger")
	if err := os.Mkdir(workDir, 0777); !os.IsExist(err) && err != nil {
		panic(err)
	}
}
func cli() {

	// override default Usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\tadd\t\tadd a new task\n")
		fmt.Fprintf(os.Stderr, "\tdump\t\tdump contents of task file\n")
		fmt.Fprintf(os.Stderr, "\thelp\t\tdisplay this help text\n")
		os.Exit(1)
	}

	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	dumpCmd := flag.NewFlagSet("dump", flag.ExitOnError)
	listCmd := flag.NewFlagSet("ls", flag.ExitOnError)

	// addCmd
	autoClosePtr := addCmd.Bool("closed", false, "automatically close a new task")
	func addTask() {
		var task Task
		f, err := os.OpenFile("/tmp/task", os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			f, err = os.Create("/tmp/task")
			if err != nil {
				fmt.Println("could not create file.")
				os.Exit(1)
			}
		}
		w := bufio.NewWriter(f)
		defer f.Close()

		if len(addCmd.Args()) < 1 {
			fmt.Println("missing required task description.")
			os.Exit(1)
		}

		if *autoClosePtr == true {
			task = Task{strings.Join(addCmd.Args(), " "), time.Now().Unix(), time.Now().Unix(), genTaskID()}
		} else {
			task = Task{strings.Join(addCmd.Args(), " "), time.Now().Unix(), 0, genTaskID()}
		}

		j, err := json.Marshal(&task)

		if err == nil {
			w.Write(j)
			w.WriteString("\n")
			w.Flush()
		} else {
			fmt.Println("error encoding JSON")
			os.Exit(1)
		}

	}

	// dumpCmd
	formatPtr := dumpCmd.String("format", "text", "<text|json|yaml>")
	func dumpTask() {
		f, err := os.Open("/tmp/task")
		if err != nil {
			fmt.Println("could not open file.")
			os.Exit(1)
		}
		scanner := bufio.NewScanner(f)
		if *formatPtr == "text" {
			for scanner.Scan() {
				fmt.Println(scanner.Text())
			}
		}
	}

	// listCmd
	func listTask() {
		var taskArray []Task
		f, err := os.Open("/tmp/task")
		if err != nil {
			fmt.Println("could not open file.")
			os.Exit(1)
		}
		for scanner.Scan() {
			var msg Task
			err := json.Unmarshal(scanner.Text(), &msg)
			if err != nil {
				fmt.Println("error unmarshalling JSON")
				os.Exit(1)
			}
			taskArray = append(taskArray, msg)
			fmt.Printf("%v", msg)
		}
	}

	// verify subcommand provided
	if len(os.Args) < 2 {
		flag.Usage()
	}

	switch os.Args[1] {
	case "add":
		addCmd.Parse(os.Args[2:])
	case "dump":
		dumpCmd.Parse(os.Args[2:])
	case "list":
		listCmd.Parse(os.Args[2:])
	case "--help", "-h", "help":
		flag.Usage()
	default:
		fmt.Printf("unknown subcommand: %v", os.Args[1])
		os.Exit(1)
	}

	if addCmd.Parsed() {
		addTask()
	} else if dumpCmd.Parsed() {
		dumpTask()
	} else if listCmd.Parsed() {
		listTask()
	}
}

func main() {
	config()
	cli()
}
