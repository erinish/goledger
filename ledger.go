package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

// Task holds an individual work record
type Task struct {
	Desc   string
	Opened int64
	Closed int64
	TaskID string
}

// Global Config
var gWorkDir string
var gTaskFile string
var gDisplay *tabwriter.Writer

func genTaskID() string {
	rand.Seed(time.Now().UnixNano())
	h := sha1.New()
	h.Write([]byte(string(rand.Intn(1000000))))
	taskid := fmt.Sprintf("%x", h.Sum(nil))
	return taskid
}

func matchTaskID(s string, tasks []Task) (int, error) {
	matches := 0
	matchIdx := 0
	for i, t := range tasks {
		if t.TaskID[0:len(s)] == s {
			matches++
			matchIdx = i
		}
	}

	if matches == 0 {
		return -1, errors.New("no matches")
	} else if matches > 1 {
		return -1, errors.New("too many matches")
	}
	return matchIdx, nil
}

func config() (string, string, *tabwriter.Writer) {
	// user dir and task file
	curUser, err := user.Current()
	if err != nil {
		fmt.Println("unknown user")
		os.Exit(1)
	}
	workDir := filepath.Join(curUser.HomeDir, ".goledger")
	if err := os.Mkdir(workDir, 0777); !os.IsExist(err) && err != nil {
		panic(err)
	}
	taskFile := filepath.Join(workDir, "tasks.json")

	// writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	return workDir, taskFile, w
}

func addTask(autoClose *bool, args []string) {
	var task Task
	f, err := os.OpenFile(gTaskFile, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		f, err = os.Create(gTaskFile)
		if err != nil {
			fmt.Println("could not create file.")
			os.Exit(1)
		}
	}
	w := bufio.NewWriter(f)
	defer f.Close()

	if len(args) < 1 {
		fmt.Println("missing required task description.")
		os.Exit(1)
	}

	if *autoClose == true {
		task = Task{strings.Join(args, " "), time.Now().Unix(), time.Now().Unix(), genTaskID()}
	} else {
		task = Task{strings.Join(args, " "), time.Now().Unix(), 0, genTaskID()}
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

func dumpTask(format *string) {
	f, err := os.Open(gTaskFile)
	if err != nil {
		fmt.Println("could not open file.")
		os.Exit(1)
	}
	scanner := bufio.NewScanner(f)
	if *format == "text" {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}
}

func getTasks() []Task {
	var taskSlice []Task
	f, err := os.Open(gTaskFile)
	if err != nil {
		fmt.Println("could not open file.")
		os.Exit(1)
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var msg Task
		err := json.Unmarshal(scanner.Bytes(), &msg)
		if err != nil {
			fmt.Println("error unmarshalling JSON")
			os.Exit(1)
		}
		taskSlice = append(taskSlice, msg)
	}
	return taskSlice
}

func listTask(oLong *bool, oAll *bool) {

	taskSlice := getTasks()

	fmt.Fprintln(gDisplay, "ID\tOPENED\tSTATUS\tTASK")

	var tID string

	for _, task := range taskSlice {
		if *oLong == true {
			tID = task.TaskID
		} else {
			tID = fmt.Sprintf("%s..", task.TaskID[0:7])
		}
		if *oAll == true {
			fmt.Fprintf(gDisplay, "%s\t%v\t%s\n", tID, time.Unix(task.Opened, 0), task.Desc)
		} else if task.Closed == 0 {
			fmt.Fprintf(gDisplay, "%s\t%v\t%s\n", tID, time.Unix(task.Opened, 0), task.Desc)
		}
	}
	gDisplay.Flush()
}

func rmTask(args []string) {
	taskSlice := getTasks()
	fmt.Println(args[0])
	taskIndex, err := matchTaskID(args[0], taskSlice)
	if err != nil {
		fmt.Println("err on rm")
		os.Exit(1)
	}

	fmt.Printf("%v", taskSlice[taskIndex])
}

func cli() {

	// override default Usage
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\tadd\t\tadd a new task\n")
		fmt.Fprintf(os.Stderr, "\tdump\t\tdump contents of task file\n")
		fmt.Fprintf(os.Stderr, "\tls\t\tdisplay list of tasks\n")
		fmt.Fprintf(os.Stderr, "\trm\t\tremove a task\n")
		fmt.Fprintf(os.Stderr, "\thelp\t\tdisplay this help text\n")
		os.Exit(1)
	}

	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	dumpCmd := flag.NewFlagSet("dump", flag.ExitOnError)
	listCmd := flag.NewFlagSet("ls", flag.ExitOnError)
	rmCmd := flag.NewFlagSet("rm", flag.ExitOnError)

	// addCmd
	autoCloseAddPtr := addCmd.Bool("closed", false, "automatically close a new task")

	// dumpCmd
	formatDumpPtr := dumpCmd.String("format", "text", "<text|json|yaml>")

	// listCmd
	longListPtr := listCmd.Bool("l", false, "print long ID")
	allListPtr := listCmd.Bool("a", false, "print long ID")

	// verify subcommand provided
	if len(os.Args) < 2 {
		flag.Usage()
	}

	switch os.Args[1] {
	case "add":
		addCmd.Parse(os.Args[2:])
	case "dump":
		dumpCmd.Parse(os.Args[2:])
	case "ls":
		listCmd.Parse(os.Args[2:])
	case "rm":
		rmCmd.Parse(os.Args[2:])
	case "--help", "-h", "help":
		flag.Usage()
	default:
		fmt.Printf("unknown subcommand: %v", os.Args[1])
		os.Exit(1)
	}

	if addCmd.Parsed() {
		addTask(autoCloseAddPtr, addCmd.Args())
	} else if dumpCmd.Parsed() {
		dumpTask(formatDumpPtr)
	} else if listCmd.Parsed() {
		listTask(longListPtr, allListPtr)
	} else if rmCmd.Parsed() {
		rmTask(rmCmd.Args())
	}
}

func main() {
	gWorkDir, gTaskFile, gDisplay = config()
	cli()
}
