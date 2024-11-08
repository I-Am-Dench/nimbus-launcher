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
	"github.com/I-Am-Dench/nimbus-launcher/patcher/protocols/netdevil"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/resources"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/term"
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
	os.Exit(0)
}

func main() {
	log.SetPrefix("nlpatcher: ")
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Fatalf("expected environment: %s", Environments.Usage())
	}

	flagset := flag.NewFlagSet("nlpatcher", flag.ExitOnError)
	patcherPath := flagset.String("patcher", "./patcher.json", "Path to the patcher configuration.")
	installationPath := flagset.String("installation", ".", "Path to installation directory. This directory should contain the client, version, and patcher directories.")
	packed := flagset.Bool("packed", false, "Whether or not the client is packed.")
	summary := flagset.Bool("summary", false, "Display a summary of patch instead of installing it.")
	noLocal := flagset.Bool("nolocal", false, "Disallows using local file paths for resources.")
	// outputBoot := flagset.Bool("boot", false, "Outputs the raw, marshalled boot.cfg.")
	flagset.Parse(os.Args[2:])

	config, err := GetEnvironmentConfig(*patcherPath)
	if err != nil {
		log.Fatal(err)
	}

	scheme, uri, err := resources.ParseScheme(config.ServiceUrl)
	if err != nil {
		log.Fatal(err)
	}
	config.ServiceUrl = uri

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var res resources.Resources
	switch scheme {
	default:
		panic(fmt.Errorf("unknown scheme: %v", scheme))
	case resources.FileScheme:
		if *noLocal {
			log.Fatal("local file paths are disallowed")
		}
		res = resources.File(ctx)
	case resources.HttpScheme:
		res = resources.Http(ctx, &http.Client{
			Jar:       CookieJar,
			Transport: http.DefaultTransport,
		})
	}

	env, err := GetEnvironment(os.Args[1], config)
	if err != nil {
		log.Fatal(err)
	}

	masterIndex, err := patcher.GetMasterIndex(res, env.FormatMasterIndexUrl(config.ServiceUrl, res.Scheme()))
	if err != nil {
		log.Fatal(err)
	}

	patcher, err := env.NewPatcher(ctx, masterIndex, patcher.Options{
		InstallDirectory: *installationPath,

		Log:       log.New(os.Stdout, os.Args[1]+": ", 0),
		Resources: res,
	})
	if err != nil {
		log.Fatal(err)
	}

	patch, err := patcher.GetPatch(ctx, *packed)
	if err != nil {
		log.Fatal(err)
	}

	if *summary {
		PrintSummary(patch.Summary())
	}

	if err := patch.Run(ctx); err != nil {
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
