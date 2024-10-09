package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/I-Am-Dench/goverbuild/encoding/ldf"
	"github.com/I-Am-Dench/nimbus-launcher/cmd/nlpatcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/cmd/nlpatcher/patcher/protocols/netdevil"
	"golang.org/x/term"
)

var Patchers = map[string]patcher.Func{
	"netdevil": netdevil.New,
}

func GetCredentials() (string, []byte, error) {
	var username string
	fmt.Print("username: ")
	if _, err := fmt.Scanln(&username); err != nil {
		return "", nil, fmt.Errorf("get credentials: %w", err)
	}

	fmt.Print("password: ")
	bpassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", nil, fmt.Errorf("get credentials: %w", err)
	}

	return strings.TrimSpace(username), bpassword, nil
}

func main() {
	log.SetPrefix("nlpatcher: ")
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Fatal("expected patch protocol: {netdevil|nimbus}")
	}

	patcherFunc, ok := Patchers[os.Args[1]]
	if !ok {
		log.Fatal("expected patch protocol: {netdevil|nimbus}")
	}

	flagset := flag.NewFlagSet("nlpatcher", flag.ExitOnError)
	noLocal := flagset.Bool("nolocal", false, "Disallows using local file paths for resources.")
	patcherPath := flagset.String("patcher", "./patcher.json", "Path to the patcher configuration.")
	installationPath := flagset.String("installation", ".", "Path to installation directory. This directory should contain the client, version, and patcher directories.")
	outputBoot := flagset.Bool("boot", false, "Outputs the raw, marshalled boot.cfg.")
	flagset.Parse(os.Args[2:])

	config, err := os.Open(*patcherPath)
	if errors.Is(err, os.ErrNotExist) {
		log.Fatalf("file does not exist: %s", *patcherPath)
	}

	if err != nil {
		log.Fatal(err)
	}

	baseConfig := patcher.Config{
		ForceRemoteResources: *noLocal,
		CookieJar:            patcher.DefaultConfig.CookieJar,
	}

	p, err := patcherFunc(config, baseConfig)
	if err != nil {
		log.Fatal(err)
	}

	patch, err := p.GetPatch(patcher.PatchOptions{
		InstallDirectory: *installationPath,
	})
	if err != nil {
		log.Fatal(err)
	}

	boot, err := patch.Patch()
	if err != nil {
		log.Fatal(err)
	}

	if !*outputBoot {
		fmt.Println("nlpatcher: Patch complete.")
	} else {
		text, _ := ldf.MarshalText(boot)
		fmt.Println(string(text))
	}
}
