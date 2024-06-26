package flags

import (
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

type TestComplete struct {
}

func (t *TestComplete) Complete(match string) []Completion {
	options := []string{
		"hello world",
		"hello universe",
		"hello multiverse",
	}

	ret := make([]Completion, 0, len(options))

	for _, o := range options {
		if strings.HasPrefix(o, match) {
			ret = append(ret, Completion{
				Item: o,
			})
		}
	}

	return ret
}

var completionTestOptions struct {
	Verbose  bool `short:"v" long:"verbose" description:"Verbose messages"`
	Debug    bool `short:"d" long:"debug" description:"Enable debug"`
	Info     bool `short:"i" description:"Display info"`
	Version  bool `long:"version" description:"Show version"`
	Required bool `long:"required" required:"true" description:"This is required"`
	Hidden   bool `long:"hidden" hidden:"true" description:"This is hidden"`

	AddCommand struct {
		Positional struct {
			Filename Filename
		} `positional-args:"yes"`
	} `command:"add" description:"add an item"`

	AddMultiCommand struct {
		Positional struct {
			Filename []Filename
		} `positional-args:"yes"`
		Extra []Filename `short:"f"`
	} `command:"add-multi" description:"add multiple items"`

	AddMultiCommandFlag struct {
		Files []Filename `short:"f"`
	} `command:"add-multi-flag" description:"add multiple items via flags"`

	RemoveCommand struct {
		Other bool     `short:"o"`
		File  Filename `short:"f" long:"filename"`
	} `command:"rm" description:"remove an item"`

	RenameCommand struct {
		Completed TestComplete `short:"c" long:"completed"`
	} `command:"rename" description:"rename an item"`

	HiddenCommand struct {
	} `command:"hidden" description:"hidden command" hidden:"true"`
}

type completionTest struct {
	Args             []string
	Completed        []string
	ShowDescriptions bool
}

var completionTests []completionTest

func makeLongName(option string) string {
	return defaultLongOptDelimiter + option
}

func makeShortName(option string) string {
	return string(defaultShortOptDelimiter) + option
}

func init() {
	_, sourcefile, _, _ := runtime.Caller(0)
	completionTestSourcedir := filepath.Join(filepath.SplitList(path.Dir(sourcefile))...)

	completionTestFilename := []string{filepath.Join(completionTestSourcedir, "completion.go"), filepath.Join(completionTestSourcedir, "completion_test.go")}

	completionTestSubdir := []string{
		filepath.Join(completionTestSourcedir, "examples/add.go"),
		filepath.Join(completionTestSourcedir, "examples/bash-completion"),
		filepath.Join(completionTestSourcedir, "examples/main.go"),
		filepath.Join(completionTestSourcedir, "examples/rm.go"),
	}

	completionTests = []completionTest{
		{
			// Short names
			[]string{makeShortName("")},
			[]string{makeLongName("debug"), makeLongName("required"), makeLongName("verbose"), makeLongName("version"), makeShortName("i")},
			false,
		},

		{
			// Short names full
			[]string{makeShortName("i")},
			[]string{makeShortName("i")},
			false,
		},

		{
			// Short names concatenated
			[]string{"-dv"},
			[]string{"-dv"},
			false,
		},

		{
			// Long names
			[]string{"--"},
			[]string{"--debug", "--required", "--verbose", "--version"},
			false,
		},

		{
			// Long names with descriptions
			[]string{"--"},
			[]string{
				"--debug     # Enable debug",
				"--required  # This is required",
				"--verbose   # Verbose messages",
				"--version   # Show version",
			},
			true,
		},

		{
			// Long names partial
			[]string{makeLongName("ver")},
			[]string{makeLongName("verbose"), makeLongName("version")},
			false,
		},

		{
			// Commands
			[]string{""},
			[]string{"add", "add-multi", "add-multi-flag", "rename", "rm"},
			false,
		},

		{
			// Commands with descriptions
			[]string{""},
			[]string{
				"add             # add an item",
				"add-multi       # add multiple items",
				"add-multi-flag  # add multiple items via flags",
				"rename          # rename an item",
				"rm              # remove an item",
			},
			true,
		},

		{
			// Commands partial
			[]string{"r"},
			[]string{"rename", "rm"},
			false,
		},

		{
			// Positional filename
			[]string{"add", filepath.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},

		{
			// Multiple positional filename (1 arg)
			[]string{"add-multi", filepath.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},
		{
			// Multiple positional filename (2 args)
			[]string{"add-multi", filepath.Join(completionTestSourcedir, "completion.go"), filepath.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},
		{
			// Multiple positional filename (3 args)
			[]string{"add-multi", filepath.Join(completionTestSourcedir, "completion.go"), filepath.Join(completionTestSourcedir, "completion.go"), filepath.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},

		{
			// Flag filename
			[]string{"rm", makeShortName("f"), filepath.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},

		{
			// Flag short concat last filename
			[]string{"rm", "-of", filepath.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},

		{
			// Flag concat filename
			[]string{"rm", "-f" + filepath.Join(completionTestSourcedir, "completion")},
			[]string{"-f" + completionTestFilename[0], "-f" + completionTestFilename[1]},
			false,
		},

		{
			// Flag equal concat filename
			[]string{"rm", "-f=" + filepath.Join(completionTestSourcedir, "completion")},
			[]string{"-f=" + completionTestFilename[0], "-f=" + completionTestFilename[1]},
			false,
		},

		{
			// Flag concat long filename
			[]string{"rm", "--filename=" + filepath.Join(completionTestSourcedir, "completion")},
			[]string{"--filename=" + completionTestFilename[0], "--filename=" + completionTestFilename[1]},
			false,
		},

		{
			// Flag long filename
			[]string{"rm", "--filename", filepath.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},

		{
			// To subdir
			[]string{"rm", "--filename", filepath.Join(completionTestSourcedir, "examples/bash-")},
			[]string{filepath.Join(completionTestSourcedir, "examples/bash-completion/")},
			false,
		},

		{
			// Subdirectory
			[]string{"rm", "--filename", filepath.Join(completionTestSourcedir, "examples") + "/"},
			completionTestSubdir,
			false,
		},

		{
			// Custom completed
			[]string{"rename", makeShortName("c"), "hello un"},
			[]string{"hello universe"},
			false,
		},
		{
			// Multiple flag filename
			[]string{"add-multi-flag", makeShortName("f"), filepath.Join(completionTestSourcedir, "completion")},
			completionTestFilename,
			false,
		},
	}
}

func TestCompletion(t *testing.T) {
	p := NewParser(&completionTestOptions, Default)
	c := &completion{parser: p}

	for _, test := range completionTests {
		if test.ShowDescriptions {
			continue
		}

		ret := c.complete(test.Args)
		items := make([]string, len(ret))

		for i, v := range ret {
			items[i] = v.Item
		}

		sort.Strings(items)
		sort.Strings(test.Completed)

		if !reflect.DeepEqual(items, test.Completed) {
			t.Errorf("Args: %#v, %#v\n  Expected: %#v\n  Got:     %#v", test.Args, test.ShowDescriptions, test.Completed, items)
		}
	}
}

func TestParserCompletion(t *testing.T) {
	for _, test := range completionTests {
		if test.ShowDescriptions {
			os.Setenv("GO_FLAGS_COMPLETION", "verbose")
		} else {
			os.Setenv("GO_FLAGS_COMPLETION", "1")
		}

		tmp := os.Stdout

		r, w, _ := os.Pipe()
		os.Stdout = w

		out := make(chan string)

		go func() {
			var buf bytes.Buffer

			io.Copy(&buf, r)

			out <- buf.String()
		}()

		p := NewParser(&completionTestOptions, None)

		p.CompletionHandler = func(items []Completion) {
			comp := &completion{parser: p}
			comp.print(items, test.ShowDescriptions)
		}

		_, err := p.ParseArgs(test.Args)

		w.Close()

		os.Stdout = tmp

		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}

		got := strings.Split(strings.Trim(<-out, "\n"), "\n")

		if !reflect.DeepEqual(got, test.Completed) {
			t.Errorf("Expected: %#v\nGot: %#v", test.Completed, got)
		}
	}

	os.Setenv("GO_FLAGS_COMPLETION", "")
}
