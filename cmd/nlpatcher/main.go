package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/I-Am-Dench/nimbus-launcher/patcher"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/origin"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/protocols/netdevil"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/undoer"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/term"
)

var (
	PatcherPath      string
	InstallationPath string
	ServerId         string
	Packed           bool
	Summary          bool
	UndoerPath       string
	UndoOnly         bool
)

var Environments = envs{
	"nd-nimbus": &netdevil.Environment{},
}

type envs map[string]patcher.Environment

func (e envs) Usage() string {
	keys := []string{}
	for k := range e {
		keys = append(keys, k)
	}
	return "<" + strings.Join(keys, "|") + ">"
}

type Config struct {
	ServiceUrl string          `json:"serviceUrl"`
	Config     json.RawMessage `json:"config"`
}

func (c *Config) GetEnvironment(patcherId string) (patcher.Environment, error) {
	env, ok := Environments[patcherId]
	if !ok {
		return nil, fmt.Errorf("unknown environment: %s: expected: %s", patcherId, Environments.Usage())
	}

	if err := json.Unmarshal(c.Config, env); err != nil {
		return nil, err
	}

	return env, nil
}

func GetConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	config := Config{}
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, err
	}

	return config, nil
}

var CookieJar, _ = cookiejar.New(&cookiejar.Options{
	PublicSuffixList: publicsuffix.List,
})

func GetCredentials() (string, []byte, error) {
	fmt.Println("\n\nEnter credentials")
	fmt.Println("=================")

	var username string
	fmt.Print("Username: ")
	if _, err := fmt.Scanln(&username); err != nil {
		return "", nil, fmt.Errorf("get credentials: %v", err)
	}

	fmt.Print("Password: ")
	bpassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", nil, fmt.Errorf("get credentials: %v", err)
	}

	return strings.TrimSpace(username), bpassword, nil
}

func PrintSummary(summary []patcher.PatchEntry) {
	tab := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(tab, "source\tdestination")
	for _, entry := range summary {
		fmt.Fprintf(tab, "%s\t%s\n", entry.Source, entry.Destination)
	}

	tab.Flush()
}

func GetPatcher(ctx context.Context, patcherId string, config Config) (patcher.Patcher, error) {
	resources, serviceUrl, err := origin.NewResources(config.ServiceUrl)
	if err != nil {
		return nil, err
	}

	if h, ok := resources.(*origin.Http); ok {
		h.Client = &http.Client{
			Jar:       CookieJar,
			Transport: http.DefaultTransport,
		}
	}

	env, err := config.GetEnvironment(patcherId)
	if err != nil {
		return nil, err
	}

	masterIndex, err := env.GetMasterIndex(ctx, serviceUrl, resources)
	if err != nil {
		return nil, err
	}

	if masterIndex.Config.Type != patcherId {
		return nil, fmt.Errorf("expected patcher %s but Master Index returned %s", patcherId, masterIndex.Config.Type)
	}

	if h, ok := resources.(*origin.Http); ok && len(masterIndex.Authentication) > 0 {
		resources = origin.WithAuthentication(h, GetCredentials, masterIndex.Authentication)
	}

	return env.NewPatcher(ctx, patcher.Options{
		Resources: resources,
		Log:       log.New(os.Stdout, patcherId+": ", 0),

		ConfigUrl:         masterIndex.Config.URL,
		AuthenticationUrl: masterIndex.Authentication,
		InstallDirectory:  InstallationPath,
		ServerId:          ServerId,
	})
}

func GetAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatal(err)
	}
	return strings.ToUpper(abs[:1]) + abs[1:] // Capitalizes drive name on windows
}

func main() {
	log.SetPrefix("nlpatcher: ")
	log.SetFlags(0)

	if len(os.Args) < 2 {
		log.Fatalf("expected environment: %s", Environments.Usage())
	}

	flagset := flag.NewFlagSet("nlpatcher", flag.ExitOnError)
	flagset.StringVar(&PatcherPath, "patcher", "patcher.json", "Path to a patcher configuration.")
	flagset.StringVar(&InstallationPath, "installation", ".", "Path to installation directory. This directory should contain the client, versions, and patcher directories.")
	flagset.StringVar(&ServerId, "serverId", "", "A unique ID which will act as a subdirectory that may store some patch resources.")
	flagset.BoolVar(&Packed, "packed", false, "Whether or not the client is packed.")
	flagset.BoolVar(&Summary, "summary", false, "Display a summary of the patch instead of installing it.")
	flagset.StringVar(&UndoerPath, "undoer", "changes.db", "Path to undoer db.")
	flagset.BoolVar(&UndoOnly, "undoOnly", false, "Undo a patch only.")
	flagset.Parse(os.Args[2:])

	InstallationPath = GetAbs(InstallationPath)

	config, err := GetConfig(PatcherPath)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	patcher, err := GetPatcher(ctx, os.Args[1], config)
	if err != nil {
		log.Fatal(err)
	}

	archive, err := patcher.GetVersion(ctx, Packed)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if archive != nil {
			if err := archive.Close(); err != nil {
				log.Println(err)
			}
		}
	}()

	undoer, err := undoer.NewSqlite(UndoerPath, InstallationPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Running undoer...")
	if err := undoer.Undo(archive); err != nil {
		log.Println(err)
		return
	}

	if UndoOnly {
		return
	}

	patch, err := patcher.GetPatch(ctx, archive)
	if err != nil {
		log.Println(err)
		return
	}

	if Summary {
		PrintSummary(patch.Summary())
		return
	}

	if err := patch.Run(ctx, undoer); err != nil {
		log.Println(err)
	}
}
