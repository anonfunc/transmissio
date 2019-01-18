// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var Default = Build

func Build() error {
	mg.Deps(Lint)
	fmt.Println("Building...")
	cmd := exec.Command("go", "build", "-o", "transmissio", ".")
	return cmd.Run()
}

func Test() error {
	return sh.Run("go", "test", "./...")
}

func Run() error {
	mg.Deps(Build)
	return sh.RunV("./transmissio")
}

// Clean up after yourself
func Clean() error {
	fmt.Println("Cleaning...")
	return os.RemoveAll("transmissio")
}

// Lint lints
func Lint() error {
	mg.Deps(Format)
	return sh.RunV("golangci-lint", "run", "--enable-all", "-D", "gochecknoglobals")
}

// Format runs goimports on everything
func Format() error {
	return sh.Run("find", ".", "-name", "*.go", "-exec", "goimports", "-w", "{}", ";")
}

// Docker builds the docker image
func Docker() error {
	return sh.RunV("docker", "build", "-t", "anonfunc/transmissio", ".")
}

// DockerPush pushes up the docker image
func DockerPush() error {
	return sh.RunV("docker", "push", "anonfunc/transmissio:latest")
}

// DockerRun runs the docker image
func DockerRun() error {
	mg.Deps(Docker)
	if err := os.MkdirAll("config", 0755); err != nil {
		return err
	}
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	return sh.RunV("docker", "run",
		"--volume", dir+"/config:/config",
		"--volume", dir+"/blackhole:/blackhole",
		"--volume", dir+"/download:/download",
		"--publish", "9091:9091",
		"--name", "anonfunc/transmissio",
		"--rm", "anonfunc/transmissio")
}
