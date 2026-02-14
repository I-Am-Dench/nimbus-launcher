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
	"github.com/I-Am-Dench/nimbus-launcher/patcher/tracker"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/term"
)

var (
	PatcherPath      string
	InstallationPath string
	ServerId         string
	Packed           bool
	Summary          bool
	TrackerPath      string
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

func GetCredentials(ctx origin.AuthContext) (string, []byte, error) {
	if len(ctx.Message) > 0 {
		fmt.Println(ctx.Message)
	}

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

func PrintSummary(summary patcher.Summary) {
	tab := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintln(tab, strings.Join(summary.Header, "\t"))
	for _, entry := range summary.Rows {
		fmt.Fprintf(tab, "%s\n", strings.Join(entry, "\t"))
	}

	tab.Flush()
}

func GetServers(ctx context.Context, patcherId string, config Config) ([]patcher.Server, error) {
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

	if masterIndex.UniverseConfig.Type != patcherId {
		return nil, fmt.Errorf("expected patcher %s but Master Index returned %s", patcherId, masterIndex.UniverseConfig.Type)
	}

	if h, ok := resources.(*origin.Http); ok && len(masterIndex.Authentication) > 0 {
		resources = origin.WithAuthentication(h, GetCredentials, masterIndex.Authentication)
	}

	return env.GetServerList(ctx, resources, masterIndex)
}

func SelectServer(servers []patcher.Server) patcher.Server {
	fmt.Println("\n\nSelect Server")
	fmt.Println("=============")
	for i, server := range servers {
		fmt.Printf("[%d] %s\n", i, server.Info().Name)
	}
	fmt.Println()

	var index int
	for {
		fmt.Print("Enter server index: ")
		fmt.Scan(&index)

		if index >= 0 && index < len(servers) {
			return servers[index]
		}
	}
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
	flagset.StringVar(&TrackerPath, "tracker", "defaultclients", "Path to client tracker.")
	flagset.BoolVar(&UndoOnly, "undoOnly", false, "Undo a patch only.")
	flagset.Parse(os.Args[2:])

	InstallationPath = GetAbs(InstallationPath)

	config, err := GetConfig(PatcherPath)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	servers, err := GetServers(ctx, os.Args[1], config)
	if err != nil {
		log.Fatal(err)
	}

	if len(servers) == 0 {
		log.Fatal("Environment has no servers")
	}

	server := servers[0]
	if len(servers) > 1 {
		server = SelectServer(servers)
	}

	patcher, err := server.GetPatcher(patcher.Options{
		Log: log.New(os.Stdout, os.Args[1]+": ", 0),

		InstallDirectory: InstallationPath,
		ServerId:         ServerId,
	})
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

	hashedPath, err := tracker.HashFilepath(InstallationPath)
	if err != nil {
		log.Fatal(err)
	}

	tracker, err := tracker.New(filepath.Join(TrackerPath, hashedPath), InstallationPath)
	if err != nil {
		log.Fatal(err)
	}
	defer tracker.Close()

	log.Println("Running undoer...")
	if err := tracker.Undo(ctx); err != nil {
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

	if err := patch.Run(ctx, tracker); err != nil {
		log.Println(err)
	}
}
