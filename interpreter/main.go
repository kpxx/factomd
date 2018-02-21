package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	//"github.com/FactomProject/electiontesting/interpreter/interpreter"
	"github.com/FactomProject/electiontesting/controller"
)

type TopLevelInterpreter struct {
	*controller.Controller
}

func main() {
	c := controller.NewController(3, 3)
	//t := new(TopLevelInterpreter)
	//t.Controller = c
	Shell(c)
}

func grabInput(in *bufio.Reader) string {
	input, err := in.ReadString('\n')
	if err != nil {
		fmt.Println("Error: ", err)
		return ""
	}
	return strings.TrimRight(input, "\n")
}

func Shell(i *controller.Controller) {
	in := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		input := grabInput(in)
		i.Interpret(strings.NewReader(input))

	}
}
