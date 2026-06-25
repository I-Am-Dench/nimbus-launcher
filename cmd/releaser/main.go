// Because of Fyne's dependency on OpenGL, we need a C compiler for
// each target GOOS. Fyne has a tool called [fyne-cross](https://github.com/fyne-io/fyne-cross)
// which uses Docker for cross compiling. However, this tool also spits out
// extra dist and tmp folders, an Icon.png file, and it pre-compresses the output.
// I just want to compile the executables only to do my own packaging, so this tool
// just utilizes the fyne-cross Docker images and spits out the final exe.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/execabs"
)

var (
	Info  = log.New(os.Stdout, "releaser: ", 0)
	Error = log.New(os.Stderr, "releaser: ", 0)
)

const (
	containerProjectDir = "/app"
	containerCacheDir   = "/go/go-build"
	containerOutputDir  = "/out"
)

type MountPoint struct {
	Name          string
	HostName      string
	ContainerName string
}

type Volume struct {
	HostProjectDir string
	HostCacheDir   string
	HostOutputDir  string
}

func (v Volume) MountPoints() []MountPoint {
	return []MountPoint{
		{Name: "project", HostName: v.HostProjectDir, ContainerName: containerProjectDir},
		{Name: "cache", HostName: v.HostCacheDir, ContainerName: containerCacheDir},
		{Name: "output", HostName: v.HostOutputDir, ContainerName: containerOutputDir},
	}
}

func (v Volume) MkdirAll() error {
	for _, dir := range []string{
		v.HostProjectDir,
		v.HostCacheDir,
		v.HostOutputDir,
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("volumn: failed to initialize volumn: %s: %v", dir, err)
		}
	}
	return nil
}

type Container struct {
	Docker string
	Image  string
	Env    map[string]string
	Volumn Volume
}

func (c Container) RunCmd(commandName string, commandArgs ...string) *exec.Cmd {
	args := []string{
		"run", "--rm", "-t",
		"-w", containerProjectDir, // set workdir
	}

	mountFormat := "%s:%s:z"
	// if runtime.GOOS == "darwin" {
	// 	mountFormat = "%s:%s"
	// }

	// Apply volumns
	for _, m := range c.Volumn.MountPoints() {
		args = append(args, "-v", fmt.Sprintf(mountFormat, m.HostName, m.ContainerName))
	}

	arch := "amd64"
	if runtime.GOARCH == "arm64" && runtime.GOOS == "darwin" {
		arch = runtime.GOARCH
	}

	args = append(args, "--platform", "linux/"+arch)
	if runtime.GOOS != "windows" {
		u, err := user.Current()
		if err == nil {
			args = append(args, "--user", u.Uid)
			args = append(args, "-e", "HOME=/tmp")
		}
	}

	// Apply go build environment variables
	args = append(args,
		"-e", "CGO_ENABLED=1",
		"-e", "GOCACHE="+containerCacheDir,
		"-e", "GOTOOLCHAIN=auto",
	)

	// Apply custom environment variables
	for k, v := range c.Env {
		args = append(args, "-e", k+"="+v)
	}

	// Docker image to use
	args = append(args, c.Image)

	// Command to run in container
	args = append(args, commandName)
	args = append(args, commandArgs...)
	Info.Print(c.Docker, " ", strings.Join(args, " "))

	return exec.Command(c.Docker, args...)
}

type Arch struct {
	Name string
	CC   string
	CXX  string
}

type Target struct {
	OS    string
	Image string
	Archs []Arch
}

func (t Target) Arch(arch string) (Arch, bool) {
	for _, a := range t.Archs {
		if a.Name == arch {
			return a, true
		}
	}
	return Arch{}, false
}

var Targets = map[string]Target{
	"windows": {
		OS:    "windows",
		Image: "fyneio/fyne-cross-images:windows",
		Archs: []Arch{
			{
				Name: "amd64",
				CC:   "zig cc -target x86_64-windows-gnu -Wdeprecated-non-prototype -Wl,--subsystem,windows",
				CXX:  "zig c++ -target x86_64-windows-gnu -Wdeprecated-non-prototype -Wl,--subsystem,windows",
			},
		},
	},
	"linux": {
		OS:    "linux",
		Image: "fyneio/fyne-cross-images:linux",
		Archs: []Arch{
			{
				Name: "amd64",
				CC:   "zig cc -target x86_64-linux-gnu -isystem /usr/include -L/usr/lib/x86_64-linux-gnu",
				CXX:  "zig c++ -target x86_64-linux-gnu -isystem /usr/include -L/usr/lib/x86_64-linux-gnu",
			},
		},
	},
}

type Flags struct {
	Architecture string
	Tags         string
	Ldflags      string
	Output       string
}

func (f Flags) AppendBuildFlags(args []string) []string {
	if len(f.Tags) > 0 {
		args = append(args, "-tags", f.Tags)
	}

	if len(f.Ldflags) > 0 {
		args = append(args, "-ldflags", f.Ldflags)
	}

	if len(f.Output) > 0 {
		args = append(args, "-o", f.Output)
	} else {
		args = append(args, "-o", containerOutputDir)
	}

	return args
}

var (
	Args = Flags{}

	Usage = "usage: releaser <GOOS> [options] [input]"
)

func buildLocally(arch Arch, input string, flags Flags) *exec.Cmd {
	args := flags.AppendBuildFlags([]string{"build"})
	args = append(args, input)

	Info.Println("go", strings.Join(args, " "))

	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOARCH="+arch.Name)
	return cmd
}

func buildWithDocker(target Target, arch Arch, input string, flags Flags) *exec.Cmd {
	volumeOutputDir := "."
	if len(flags.Output) > 0 {
		if stat, err := os.Stat(flags.Output); err != nil {
			if err := os.MkdirAll(filepath.Dir(flags.Output), 0755); err != nil {
				Error.Fatal(err)
			}
			volumeOutputDir = filepath.Dir(flags.Output)
			flags.Output = path.Join(containerOutputDir, filepath.Base(flags.Output))
		} else if stat.IsDir() {
			volumeOutputDir = flags.Output
			flags.Output = containerOutputDir
		} else {
			volumeOutputDir = flags.Output
			flags.Output = path.Join(containerOutputDir, filepath.Base(flags.Output))
		}
	}

	var err error
	volumeOutputDir, err = filepath.Abs(volumeOutputDir)
	if err != nil {
		Error.Fatal(err)
	}

	goInput := "."
	if strings.HasSuffix(input, ".go") {
		goInput = filepath.Base(input)
	}

	projectDir, err := filepath.Abs(input)
	if err != nil {
		Error.Fatal(err)
	}

	if stat, err := os.Stat(projectDir); err != nil {
		Error.Fatal(err)
	} else if !stat.IsDir() {
		projectDir = filepath.Dir(projectDir)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		Error.Fatalf("get cache: %v", err)
	}
	cacheDir = filepath.Join(cacheDir, "nimbus-launcher.releaser")

	dockerBin, err := execabs.LookPath("docker")
	if err != nil {
		Error.Fatal(err)
	}

	volumn := Volume{
		HostProjectDir: projectDir,
		HostCacheDir:   cacheDir,
		HostOutputDir:  volumeOutputDir,
	}
	if err := volumn.MkdirAll(); err != nil {
		Error.Fatal(err)
	}

	env := map[string]string{
		"GOOS":   target.OS,
		"GOARCH": arch.Name,
		"CC":     arch.CC,
		"CXX":    arch.CXX,
	}

	args := flags.AppendBuildFlags([]string{"build"})
	args = append(args, goInput)

	container := Container{
		Docker: dockerBin,
		Image:  target.Image,
		Env:    env,
		Volumn: volumn,
	}
	return container.RunCmd("go", args...)
}

func main() {
	if len(os.Args) < 2 {
		Error.Fatal(Usage)
	}

	flagset := flag.NewFlagSet("releaser", flag.ExitOnError)
	flagset.StringVar(&Args.Architecture, "arch", "", "Required (amd64, arm64)")
	flagset.StringVar(&Args.Tags, "tags", "", "Build tags")
	flagset.StringVar(&Args.Ldflags, "ldflags", "", "")
	flagset.StringVar(&Args.Output, "o", "", "Output name")
	flagset.Parse(os.Args[2:])

	goos := os.Args[1]
	goarch := Args.Architecture
	if len(goarch) == 0 {
		Error.Fatalf("Missing value for -arch option: %s", flag.Lookup("arch").Usage)
	}

	input := flagset.Arg(0)
	if len(input) == 0 {
		input = "."
	}

	target, ok := Targets[goos]
	if !ok {
		Error.Fatalf("unhandled GOOS: %s", goos)
	}

	arch, ok := target.Arch(goarch)
	if !ok {
		Error.Fatalf("unhandled GOARCH: %s", goarch)
	}

	var cmd *exec.Cmd
	if runtime.GOOS == goos {
		Info.Println("Building locally")
		cmd = buildLocally(arch, input, Args)
	} else {
		Info.Println("Building with Docker")
		cmd = buildWithDocker(target, arch, input, Args)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		Error.Fatal(err)
	}
}
