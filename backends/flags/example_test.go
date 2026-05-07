package flags_test

import (
	"context"
	"fmt"
	"os"

	"golang.yandex/confetti"
	"golang.yandex/confetti/backends/flags"
)

func ExampleFrom() {
	originalArgs := os.Args
	os.Args = []string{"app", "-p", "/tmp/packages", "-r", "/usr/local/go"}
	defer func() { os.Args = originalArgs }()

	var gopath, goroot string
	err := confetti.NewLoader().Load(
		context.Background(),
		confetti.To(&gopath, flags.From("p")),
		confetti.To(&goroot, flags.From("r")),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(gopath)
	fmt.Println(goroot)
	// Output:
	// /tmp/packages
	// /usr/local/go
}

func ExampleFromArgs() {
	originalArgs := os.Args
	os.Args = []string{"app", "-g", "/tmp/packages", "-r", "/usr/local/go"}
	defer func() { os.Args = originalArgs }()

	var setup struct {
		GoPath string `flag:"g"`
		GoRoot string `flag:"r"`
	}
	err := confetti.NewLoader().Load(
		context.Background(),
		confetti.To(&setup, flags.FromArgs()),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(setup.GoPath)
	fmt.Println(setup.GoRoot)
	// Output:
	// /tmp/packages
	// /usr/local/go
}
