package internal

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/cupcakearmy/autorestic/internal/colors"
	"github.com/cupcakearmy/autorestic/internal/flags"
	"github.com/fatih/color"
)

func CheckIfCommandIsCallable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func CheckIfResticIsCallable() bool {
	return CheckIfCommandIsCallable(flags.RESTIC_BIN)
}

type ExecuteOptions struct {
	Command string
	Envs    map[string]string
	Dir     string
	Silent  bool
}

type ColoredWriter struct {
	target io.Writer
	color  *color.Color
}

func (w ColoredWriter) Write(p []byte) (n int, err error) {
	colored := []byte(w.color.Sprint(string(p)))
	w.target.Write(colored)
	return len(p), nil
}

func ExecuteCommand(options ExecuteOptions, args ...string) (int, string, error) {
	cmd := exec.Command(options.Command, args...)
	env := os.Environ()
	for k, v := range options.Envs {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env
	cmd.Dir = options.Dir

	if flags.VERBOSE {
		colors.Faint.Printf("> Executing: %s\n", cmd)
	}

	var out bytes.Buffer
	var error bytes.Buffer
	if flags.VERBOSE && !options.Silent {
		var colored ColoredWriter = ColoredWriter{
			target: os.Stdout,
			color:  colors.Faint,
		}
		mw := io.MultiWriter(colored, &out)
		cmd.Stdout = mw
	} else {
		cmd.Stdout = &out
	}
	cmd.Stderr = &error
	err := cmd.Run()
	if err != nil {
		code := -1
		if exitError, ok := err.(*exec.ExitError); ok {
			code = exitError.ExitCode()
		}
		return code, error.String(), err
	}
	return 0, out.String(), nil
}

func ExecuteResticCommand(options ExecuteOptions, args ...string) (int, string, error) {
	options.Command = flags.RESTIC_BIN
	var c = GetConfig()
	var optionsAsString = getOptions(c.Global, []string{"all"})
	args = append(optionsAsString, args...)
	return ExecuteCommand(options, args...)
}

func CopyFile(from, to string) error {
	original, err := os.Open(from)
	if err != nil {
		return nil
	}
	defer original.Close()

	new, err := os.Create(to)
	if err != nil {
		return nil
	}
	defer new.Close()

	if _, err := io.Copy(new, original); err != nil {
		return err
	}
	return nil
}

func CheckIfVolumeExists(volume string) bool {
	_, _, err := ExecuteCommand(ExecuteOptions{Command: "docker"}, "volume", "inspect", volume)
	return err == nil
}

func ArrayContains[T comparable](arr []T, needle T) bool {
	for _, item := range arr {
		if item == needle {
			return true
		}
	}
	return false
}

func TopologicalSort[T comparable](adjacencyList map[T][]T, reverse bool) ([]T, error) {
	// https://www.geeksforgeeks.org/topological-sorting-indegree-based-solution/

	var result []T

	if len(adjacencyList) == 0 {
		return result, nil
	}

	if reverse {
		adjacencyListReverse := make(map[T][]T)
		// to keep original sorting order, initialize reverse adjacency list with all nodes
		for node := range adjacencyList {
			adjacencyListReverse[node] = []T{}
		}
		// save the reverse edges
		for node, edges := range adjacencyList {
			for _, adjacentNode := range edges {
				adjacencyListReverse[adjacentNode] = append(adjacencyListReverse[adjacentNode], node)
			}
		}
		adjacencyList = adjacencyListReverse
	}

	var queue []T
	indegree := make(map[T]int)
	// build indegree

	for _, edges := range adjacencyList {
		uniqueMap := make(map[T]int)
		for _, adjacentNode := range edges {
			uniqueMap[adjacentNode]++
			if uniqueMap[adjacentNode] != 1 {
				continue
			}

			indegree[adjacentNode]++

		}
	}

	// fill queue with nodes that have indegree 0
	for elem := range adjacencyList {
		if indegree[elem] == 0 {
			queue = append(queue, elem)
		}
	}

	for len(queue) > 0 {
		var currentNode T
		currentNode, queue = queue[0], queue[1:]
		result = append(result, currentNode)
		uniqueMap := make(map[T]int)
		for _, adjacentNode := range adjacencyList[currentNode] {
			uniqueMap[adjacentNode]++
			if uniqueMap[adjacentNode] != 1 {
				continue
			}
			indegree[adjacentNode]--

			if indegree[adjacentNode] == 0 {
				queue = append(queue, adjacentNode)
			}
		}
	}

	if len(result) != len(adjacencyList) {
		return nil, fmt.Errorf("cyclic dependency")
	}

	return result, nil
}
