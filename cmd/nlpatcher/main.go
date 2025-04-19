package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/client"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/protocols/netdevil"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/remote"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/undoer"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/term"
)

var (
	PatcherPath      string
	InstallationPath string
	Packed           bool
	Summary          bool
	OnlyUndo         bool
	NoLocal          bool
)

type envs map[string]patcher.Environment

var Environments = envs{
	"nd-nimbus": &netdevil.Config{},
}

func (envs envs) Usage() string {
	keys := []string{}
	for k := range envs {
		keys = append(keys, k)
	}
	return fmt.Sprint("{", strings.Join(keys, "|"), "}")
}

var (
	CookieJar, _ = cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
)

func GetCredentials() (string, []byte, error) {
	fmt.Println("\n\nEnter credentials")
	fmt.Println("=================")

	var username string
	fmt.Print("Username: ")
	if _, err := fmt.Scanln(&username); err != nil {
		return "", nil, fmt.Errorf("get credentials: %w", err)
	}

	fmt.Print("Password: ")
	bpassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", nil, fmt.Errorf("get credentials: %w", err)
	}

	return strings.TrimSpace(username), bpassword, nil
}

func GetEnvironmentConfig(path string) (patcher.Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return patcher.Config{}, fmt.Errorf("could not find environment config: %s", path)
	}

	if err != nil {
		return patcher.Config{}, err
	}

	config := patcher.Config{}
	if err := json.Unmarshal(data, &config); err != nil {
		return patcher.Config{}, err
	}

	return config, nil
}

func GetEnvironment(patcherId string, config patcher.Config) (patcher.Environment, error) {
	env, ok := Environments[patcherId]
	if !ok {
		return nil, fmt.Errorf("unknown environment: %s: expected: %s", patcherId, Environments.Usage())
	}

	if err := json.Unmarshal(config.Config, &env); err != nil {
		return nil, err
	}

	return env, nil
}

func PrintSummary(summary []patcher.PatchEntry) {
	tab := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(tab, "source\tdestination")
	for _, entry := range summary {
		fmt.Fprintf(tab, "%s\t%s\n", entry.Source, entry.Destination)
	}

	tab.Flush()
}

func main() {
	log.SetPrefix("nlpatcher: ")
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Fatalf("expected environment: %s", Environments.Usage())
	}

	flagset := flag.NewFlagSet("nlpatcher", flag.ExitOnError)
	flagset.StringVar(&PatcherPath, "patcher", "patcher.json", "Path to a patcher configuration.")
	flagset.StringVar(&InstallationPath, "installation", ".", "Patch to installation directory. This directory should contain the client, version, and patcher directories.")
	flagset.BoolVar(&Packed, "packed", false, "Whether or not the client is packed.")
	flagset.BoolVar(&Summary, "summary", false, "Display a summary of the patch instead of installing it.")
	flagset.BoolVar(&OnlyUndo, "onlyundo", false, "Undo a patch only.")
	flagset.BoolVar(&NoLocal, "nolocal", false, "Disallows using local file paths for resources.")
	// outputBoot := flagset.Bool("boot", false, "Outputs the raw, marshalled boot.cfg.")
	flagset.Parse(os.Args[2:])

	config, err := GetEnvironmentConfig(PatcherPath)
	if err != nil {
		log.Fatal(err)
	}

	scheme, uri, err := remote.ParseScheme(config.ServiceUrl)
	if err != nil {
		log.Fatal(err)
	}
	config.ServiceUrl = uri

	undoer, err := undoer.NewSqlite("changes.db", InstallationPath)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var resources remote.Resources
	switch scheme {
	case remote.FileScheme:
		if NoLocal {
			log.Fatal("local file paths are disallowed")
		}
		resources = remote.File()
	case remote.HttpScheme:
		resources = remote.Http(&http.Client{
			Jar:       CookieJar,
			Transport: http.DefaultTransport,
		})
	default:
		panic(fmt.Errorf("unknown scheme: %v", scheme))
	}

	env, err := GetEnvironment(os.Args[1], config)
	if err != nil {
		log.Fatal(err)
	}

	masterIndex, err := patcher.GetMasterIndex(ctx, resources, env.FormatMasterIndexUrl(config.ServiceUrl, resources.Scheme()))
	if err != nil {
		log.Fatal(err)
	}

	if h, ok := resources.(remote.HttpResources); ok && len(masterIndex.Authentication) > 0 {
		resources = remote.WithAuthentication(h, GetCredentials, masterIndex.Authentication)
	}

	patcher, err := env.NewPatcher(ctx, masterIndex, patcher.Options{
		InstallDirectory: InstallationPath,

		Log:       log.New(os.Stdout, os.Args[1]+": ", 0),
		Resources: resources,
	})
	if err != nil {
		log.Fatal(err)
	}

	patch, err := patcher.GetPatch(ctx, Packed)
	if err != nil {
		log.Fatal(err)
	}

	if Summary {
		PrintSummary(patch.Summary())
		return
	}

	var archive *client.Archive
	if catalog, ok := patch.Catalog(); ok {
		archive = client.NewArchive(catalog, InstallationPath)
		defer func() {
			if err := archive.Close(); err != nil {
				log.Println(err)
			}
		}()
	}

	log.Println("Running undoer...")
	// Reset client to original state
	if err := undoer.Undo(archive); err != nil {
		log.Println(err)
	}

	if OnlyUndo {
		return
	}

	if err := patch.Run(ctx, undoer); err != nil {
		log.Fatal(err)
	}

	// bootConfig := patcher.GetBoot(*packed)
	// if len(bootConfig.PasswordURL) == 0 {
	// 	bootConfig.PasswordURL = boot.DefaultConfig.PasswordURL
	// }

	// if len(bootConfig.SigninURL) == 0 {
	// 	bootConfig.SigninURL = boot.DefaultConfig.SigninURL
	// }

	// if len(bootConfig.SignupURL) == 0 {
	// 	bootConfig.SignupURL = boot.DefaultConfig.SignupURL
	// }

	// if len(bootConfig.RegisterURL) == 0 {
	// 	bootConfig.RegisterURL = boot.DefaultConfig.RegisterURL
	// }
}
