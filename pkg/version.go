package main

import (
	"fmt"
)

type CmdGroupVersion struct {
}

var versionCmdName = "version"
var versionCmdGroup CmdGroupVersion

// TODO: include this into goreleaser
func (g *CmdGroupVersion) Execute(args []string) error {
	fmt.Println("c8y-certificate-cli version: 0.1.0")
	return nil
}
