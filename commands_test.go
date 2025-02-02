package cli

import (
	"flag"
	"os"
	"strings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"

	"fmt"

	"github.com/jawher/mow.cli/internal/container"
	"github.com/jawher/mow.cli/internal/flow"
)

func okCmd(t *testing.T, spec string, init CmdInitializer, args []string) {
	cmd := &Cmd{
		name:       "test",
		optionsIdx: map[string]*container.Container{},
		argsIdx:    map[string]*container.Container{},
	}
	cmd.Spec = spec
	cmd.ErrorHandling = flag.ContinueOnError
	init(cmd)

	err := cmd.doInit()
	require.Nil(t, err, "should parse")
	t.Logf("testing spec %q with args: %#v", spec, args)
	inFlow := &flow.Step{}
	err = cmd.parse(args, inFlow, inFlow, &flow.Step{})
	require.Nil(t, err, "cmd parse should't fail")
}

func failCmd(t *testing.T, spec string, init CmdInitializer, args []string) {
	cmd := &Cmd{
		name:       "test",
		optionsIdx: map[string]*container.Container{},
		argsIdx:    map[string]*container.Container{},
	}
	cmd.Spec = spec
	cmd.ErrorHandling = flag.ContinueOnError
	init(cmd)

	err := cmd.doInit()
	require.NoError(t, err, "should parse")
	t.Logf("testing spec %q with args: %#v", spec, args)
	inFlow := &flow.Step{}
	err = cmd.parse(args, inFlow, inFlow, &flow.Step{})
	require.Error(t, err, "cmd parse should have failed")
}

func badSpec(t *testing.T, spec string, init CmdInitializer) {
	cmd := &Cmd{
		name:       "test",
		optionsIdx: map[string]*container.Container{},
		argsIdx:    map[string]*container.Container{},
	}
	cmd.Spec = spec
	cmd.ErrorHandling = flag.ContinueOnError
	init(cmd)

	t.Logf("testing bad spec %s", spec)
	err := cmd.doInit()
	require.NotNil(t, err, "Bad spec %s should have failed to parse", spec)
	t.Logf("Bad spec %s did fail to parse with error: %v", spec, err)
}

func TestSpecBoolOpt(t *testing.T) {
	var f *bool
	init := func(c *Cmd) {
		f = c.BoolOpt("f force", false, "")
	}
	spec := "-f"

	okCases := []struct {
		args     []string
		expected bool
	}{
		{args: []string{"-f"}, expected: true},
		{args: []string{"--force"}, expected: true},
		{args: []string{"-f=true"}, expected: true},
		{args: []string{"--force=true"}, expected: true},
		{args: []string{"--force=false"}, expected: false},
	}

	for _, cas := range okCases {
		t.Run(fmt.Sprintf("okCase: %+v", cas.args), func(t *testing.T) {
			okCmd(t, spec, init, cas.args)
			require.Equal(t, cas.expected, *f)
		})
	}

	badCases := [][]string{
		{},
		{"-g"},
		{"-f", "-g"},
		{"-g", "-f"},
		{"-f", "true"},
		{"-f", "xxx"},
		{"xxx", "-f"},
	}
	for _, args := range badCases {
		t.Run(fmt.Sprintf("badCase: %+v", args), func(t *testing.T) {
			failCmd(t, spec, init, args)
		})
	}
}

func TestDefaultSpec(t *testing.T) {
	var (
		a *bool
		b *string
		c *string
	)

	type call struct {
		args []string
		a    bool
		b, c string
	}
	cases := []struct {
		init  func(cmd *Cmd)
		calls []call
	}{
		{
			func(cmd *Cmd) {
				a = cmd.BoolOpt("a", false, "")
				b = cmd.StringOpt("b", "", "")
				c = cmd.StringArg("C", "", "")
			},
			[]call{
				{[]string{"X"}, false, "", "X"},
				{[]string{"-a", "X"}, true, "", "X"},
				{[]string{"-b=Z", "X"}, false, "Z", "X"},
				{[]string{"-b=Z", "-a", "X"}, true, "Z", "X"},
				{[]string{"-a", "-b=Z", "X"}, true, "Z", "X"},
			},
		},
	}

	for _, cas := range cases {
		for _, cl := range cas.calls {
			t.Run(fmt.Sprintf("%+v", cl.args), func(t *testing.T) {
				okCmd(t, "", cas.init, cl.args)
				require.Equal(t, cl.a, *a)
				require.Equal(t, cl.b, *b)
				require.Equal(t, cl.c, *c)
			})
		}
	}

}

func TestSpecOptFolding(t *testing.T) {
	var a, b, c *bool
	var d *string
	init := func(cmd *Cmd) {
		a = cmd.BoolOpt("a", false, "")
		b = cmd.BoolOpt("b", false, "")
		c = cmd.BoolOpt("c", false, "")

		d = cmd.StringOpt("d", "", "")
	}

	cases := []struct {
		spec    string
		args    []string
		a, b, c bool
		d       string
	}{
		{
			"[-abcd]", []string{},
			false, false, false,
			"",
		},
		{
			"[-abcd]", []string{"-ab"},
			true, true, false,
			"",
		},
		{
			"[-abcd]", []string{"-ba"},
			true, true, false,
			"",
		},

		{
			"[-abcd]", []string{"-ad", "TEST"},
			true, false, false,
			"TEST",
		},
		{
			"[-abcd]", []string{"-adTEST"},
			true, false, false,
			"TEST",
		},
		{
			"[-abcd]", []string{"-abd", "TEST"},
			true, true, false,
			"TEST",
		},
		{
			"[-abcd]", []string{"-abdTEST"},
			true, true, false,
			"TEST",
		},
		{
			"[-abcd]", []string{"-abcd", "TEST"},
			true, true, true,
			"TEST",
		},
		{
			"[-abcd]", []string{"-bcd", "TEST"},
			false, true, true,
			"TEST",
		},
		{
			"[-abcd]", []string{"-cbd", "TEST"},
			false, true, true,
			"TEST",
		},
		{
			"[-abcd]", []string{"-ac"},
			true, false, true,
			"",
		},
		{
			"[-abcd]", []string{"-ca"},
			true, false, true,
			"",
		},
		{
			"[-abcd]", []string{"-cab"},
			true, true, true,
			"",
		},
	}

	for _, cas := range cases {
		t.Run(fmt.Sprintf("%s :: %+v", cas.spec, cas.args), func(t *testing.T) {
			okCmd(t, cas.spec, init, cas.args)
			require.Equal(t, cas.a, *a)
			require.Equal(t, cas.b, *b)
			require.Equal(t, cas.c, *c)
			require.Equal(t, cas.d, *d)
		})
	}

}

func TestSpecStrOpt(t *testing.T) {
	var f *string
	init := func(c *Cmd) {
		f = c.StringOpt("f", "", "")
	}
	spec := "-f"

	cases := [][]string{
		{"-fValue"},
		{"-f", "Value"},
		{"-f=Value"},
	}
	for _, args := range cases {
		okCmd(t, spec, init, args)
		require.Equal(t, "Value", *f)
	}

	badCases := [][]string{
		{},
		{"-g"},
		{"-f", "-g"},
		{"-g", "-f"},
		{"-f", "xxx", "yyy"},
		{"xxx", "-f"},
	}
	for _, args := range badCases {
		t.Run(fmt.Sprintf("%+v", args), func(t *testing.T) {
			failCmd(t, spec, init, args)
		})
	}
}

func TestSpecIntOpt(t *testing.T) {
	var f *int
	init := func(c *Cmd) {
		f = c.IntOpt("f", -1, "")
	}

	spec := "-f"
	cases := [][]string{
		{"-f42"},
		{"-f", "42"},
		{"-f=42"},
	}
	for _, args := range cases {
		okCmd(t, spec, init, args)
		require.Equal(t, 42, *f)
	}

	badCases := [][]string{
		{},
		{"-f", "x"},
		{"-g"},
		{"-f", "-g"},
		{"-g", "-f"},
		{"-f", "xxx"},
		{"xxx", "-f"},
	}
	for _, args := range badCases {
		t.Run(fmt.Sprintf("%+v", args), func(t *testing.T) {
			failCmd(t, spec, init, args)
		})
	}
}

func TestSpecStrsOpt(t *testing.T) {
	var f *[]string
	init := func(c *Cmd) {
		f = c.StringsOpt("f", nil, "")
	}
	spec := "-f..."
	cases := [][]string{
		{"-fA"},
		{"-f", "A"},
		{"-f=A"},
	}
	for _, args := range cases {
		okCmd(t, spec, init, args)
		require.Equal(t, []string{"A"}, *f)
	}

	cases = [][]string{
		{"-fA", "-f", "B"},
		{"-f", "A", "-f", "B"},
		{"-f=A", "-fB"},
	}
	for _, args := range cases {
		t.Run(fmt.Sprintf("okCase: %#v", args), func(t *testing.T) {
			okCmd(t, spec, init, args)
			require.Equal(t, []string{"A", "B"}, *f)
		})
	}

	badCases := [][]string{
		{},
		{"-g"},
		{"-f", "-g"},
		{"-g", "-f"},
		{"-f", "xxx", "yyy"},
		{"xxx", "-f"},
	}
	for _, args := range badCases {
		t.Run(fmt.Sprintf("badCase: %#v", args), func(t *testing.T) {
			failCmd(t, spec, init, args)
		})
	}
}

func TestSpecIntsOpt(t *testing.T) {
	var f *[]int
	init := func(c *Cmd) {
		f = c.IntsOpt("f", nil, "")
	}
	spec := "-f..."
	cases := [][]string{
		{"-f1"},
		{"-f", "1"},
		{"-f=1"},
	}
	for _, args := range cases {
		okCmd(t, spec, init, args)
		require.Equal(t, []int{1}, *f)
	}

	cases = [][]string{
		{"-f1", "-f", "2"},
		{"-f", "1", "-f", "2"},
		{"-f=1", "-f2"},
	}
	for _, args := range cases {
		t.Run(fmt.Sprintf("okCase: %#v", args), func(t *testing.T) {
			okCmd(t, spec, init, args)
			require.Equal(t, []int{1, 2}, *f)
		})
	}

	badCases := [][]string{
		{},
		{"-f", "b"},
		{"-f", "3", "-f", "c"},
		{"-g"},
		{"-f", "-g"},
		{"-g", "-f"},
		{"-f", "xxx"},
		{"xxx", "-f"},
	}
	for _, args := range badCases {
		t.Run(fmt.Sprintf("badCase: %#v", args), func(t *testing.T) {
			failCmd(t, spec, init, args)
		})
	}
}

func TestSpecOptionalOpt(t *testing.T) {
	var f *bool
	init := func(c *Cmd) {
		f = c.BoolOpt("f", false, "")
	}
	spec := "[-f]"
	//TODO: table driven
	okCmd(t, "[-f]", init, []string{"-f"})
	require.True(t, *f)

	okCmd(t, spec, init, []string{})
	require.False(t, *f)

	badCases := [][]string{
		{"-g"},
		{"-f", "-g"},
		{"-g", "-f"},
		{"-f", "xxx"},
		{"xxx", "-f"},
	}
	for _, args := range badCases {
		t.Run(fmt.Sprintf("badCase: %#v", args), func(t *testing.T) {
			failCmd(t, spec, init, args)
		})
	}
}

func TestSpecArg(t *testing.T) {
	var s *string
	init := func(c *Cmd) {
		s = c.StringArg("ARG", "", "")
	}
	spec := "ARG"
	okCmd(t, spec, init, []string{"value"})
	require.Equal(t, "value", *s)

	badCases := [][]string{
		{},
		{"-g"},
		{"-f", "xxx"},
		{"xxx", "-f"},
	}
	for _, args := range badCases {
		t.Run(fmt.Sprintf("badCase: %#v", args), func(t *testing.T) {
			failCmd(t, spec, init, args)
		})
	}
}

func TestSpecOptionalArg(t *testing.T) {
	var s *string
	init := func(c *Cmd) {
		s = c.StringArg("ARG", "", "")
	}
	spec := "[ARG]"

	okCmd(t, spec, init, []string{"value"})
	require.Equal(t, "value", *s)

	okCmd(t, spec, init, []string{})
	require.Equal(t, "", *s)

	badCases := [][]string{
		{"-g"},
		{"-g", "-f"},
		{"-f", "xxx"},
		{"xxx", "-f"},
	}
	for _, args := range badCases {
		t.Run(fmt.Sprintf("badCase: %#v", args), func(t *testing.T) {
			failCmd(t, spec, init, args)
		})
	}

}

func TestSpecOptionChoice(t *testing.T) {
	var f, g *bool
	init := func(c *Cmd) {
		f = c.BoolOpt("f", false, "")
		g = c.BoolOpt("g", false, "")
	}
	spec := "-f|-g"
	//TODO: table driven
	okCmd(t, spec, init, []string{"-f"})
	require.True(t, *f)
	require.False(t, *g)

	okCmd(t, spec, init, []string{"-g"})
	require.False(t, *f)
	require.True(t, *g)

	badCases := [][]string{
		{},
		{"-f", "-g"},
		{"-f", "-s"},
		{"-g", "-f"},
		{"-f", "xxx"},
		{"xxx", "-f"},
	}
	for _, args := range badCases {
		t.Run(fmt.Sprintf("badCase: %#v", args), func(t *testing.T) {
			failCmd(t, spec, init, args)
		})
	}
}

func TestSpecOptional2OptionChoice(t *testing.T) {
	var f, g *bool
	init := func(c *Cmd) {
		f = c.BoolOpt("f", false, "")
		g = c.BoolOpt("g", false, "")
	}
	spec := "[-f|-g]"
	//TODO:
	okCmd(t, spec, init, []string{})
	require.False(t, *f)
	require.False(t, *g)

	okCmd(t, spec, init, []string{"-f"})
	require.True(t, *f)
	require.False(t, *g)

	okCmd(t, spec, init, []string{"-g"})
	require.False(t, *f)
	require.True(t, *g)

	badCases := [][]string{
		{"-s"},
		{"-f", "-g"},
		{"-g", "-f"},
		{"-f", "xxx"},
		{"xxx", "-f"},
	}
	for _, args := range badCases {
		t.Run(fmt.Sprintf("badCase: %#v", args), func(t *testing.T) {
			failCmd(t, spec, init, args)
		})
	}
}

func TestSpecRepeatable2OptionChoice(t *testing.T) {
	var f, g *bool
	init := func(c *Cmd) {
		f = c.BoolOpt("f", false, "")
		g = c.BoolOpt("g", false, "")
	}
	spec := "(-f|-g)..."
	//TODO:
	okCmd(t, spec, init, []string{"-f"})
	require.True(t, *f)
	require.False(t, *g)

	okCmd(t, spec, init, []string{"-g"})
	require.False(t, *f)
	require.True(t, *g)

	okCmd(t, spec, init, []string{"-f", "-g"})
	require.True(t, *f)
	require.True(t, *g)

	okCmd(t, spec, init, []string{"-g", "-f"})
	require.True(t, *f)
	require.True(t, *g)

	badCases := [][]string{
		{"-s"},
		{"-f", "xxx"},
		{"xxx", "-f"},
	}
	for _, args := range badCases {
		t.Run(fmt.Sprintf("badCase: %#v", args), func(t *testing.T) {
			failCmd(t, spec, init, args)
		})
	}
}

func TestSpecRepeatableOptional2OptionChoice(t *testing.T) {
	var f, g *bool
	init := func(c *Cmd) {
		f = c.BoolOpt("f", false, "")
		g = c.BoolOpt("g", false, "")
	}
	spec := "[-f|-g]..."
	//TODO:
	okCmd(t, spec, init, []string{})
	require.False(t, *f)
	require.False(t, *g)

	okCmd(t, spec, init, []string{"-f"})
	require.True(t, *f)
	require.False(t, *g)

	okCmd(t, spec, init, []string{"-g"})
	require.False(t, *f)
	require.True(t, *g)

	okCmd(t, spec, init, []string{"-f", "-g"})
	require.True(t, *f)
	require.True(t, *g)

	okCmd(t, spec, init, []string{"-g", "-f"})
	require.True(t, *f)
	require.True(t, *g)

	badCases := [][]string{
		{"-s"},
		{"-f", "xxx"},
		{"xxx", "-f"},
	}
	for _, args := range badCases {
		t.Run(fmt.Sprintf("badCase: %#v", args), func(t *testing.T) {
			failCmd(t, spec, init, args)
		})
	}
}

func TestSpecOption3Choice(t *testing.T) {
	var f, g, h *bool
	init := func(c *Cmd) {
		f = c.BoolOpt("f", false, "")
		g = c.BoolOpt("g", false, "")
		h = c.BoolOpt("x", false, "")
	}
	spec := "-f|-g|-x"

	//TODO
	okCmd(t, spec, init, []string{"-f"})
	require.True(t, *f)
	require.False(t, *g)
	require.False(t, *h)

	okCmd(t, spec, init, []string{"-g"})
	require.False(t, *f)
	require.True(t, *g)
	require.False(t, *h)

	okCmd(t, spec, init, []string{"-x"})
	require.False(t, *f)
	require.False(t, *g)
	require.True(t, *h)
}

func TestSpecOptionalOption3Choice(t *testing.T) {
	var f, g, h *bool
	init := func(c *Cmd) {
		f = c.BoolOpt("f", false, "")
		g = c.BoolOpt("g", false, "")
		h = c.BoolOpt("x", false, "")
	}
	spec := "[-f|-g|-x]"

	okCmd(t, spec, init, []string{})
	require.False(t, *f)
	require.False(t, *g)
	require.False(t, *h)

	okCmd(t, spec, init, []string{"-f"})
	require.True(t, *f)
	require.False(t, *g)
	require.False(t, *h)

	okCmd(t, spec, init, []string{"-g"})
	require.False(t, *f)
	require.True(t, *g)
	require.False(t, *h)

	okCmd(t, spec, init, []string{"-x"})
	require.False(t, *f)
	require.False(t, *g)
	require.True(t, *h)
}

func TestSpecC1(t *testing.T) {
	var f, g *bool
	init := func(c *Cmd) {
		f = c.BoolOpt("f", false, "")
		g = c.BoolOpt("g", false, "")
	}
	spec := "-f|-g..."
	// spec = "[-f|-g...]"

	okCmd(t, spec, init, []string{"-f"})
	require.True(t, *f)
	require.False(t, *g)

	okCmd(t, spec, init, []string{"-g"})
	require.False(t, *f)
	require.True(t, *g)

	okCmd(t, spec, init, []string{"-g", "-g"})
	require.False(t, *f)
	require.True(t, *g)
}

func TestSpecC2(t *testing.T) {
	var f, g *bool
	init := func(c *Cmd) {
		f = c.BoolOpt("f", false, "")
		g = c.BoolOpt("g", false, "")
	}
	spec := "[-f|-g...]"

	okCmd(t, spec, init, []string{})
	require.False(t, *f)
	require.False(t, *g)

	okCmd(t, spec, init, []string{"-f"})
	require.True(t, *f)
	require.False(t, *g)

	okCmd(t, spec, init, []string{"-g"})
	require.False(t, *f)
	require.True(t, *g)

	okCmd(t, spec, init, []string{"-g", "-g"})
	require.False(t, *f)
	require.True(t, *g)
}

func TestSpecCpCase(t *testing.T) {
	var f, g *[]string
	init := func(c *Cmd) {
		f = c.StringsArg("SRC", nil, "")
		g = c.StringsArg("DST", nil, "")
	}
	spec := "SRC... DST"

	okCmd(t, spec, init, []string{"A", "B"})
	require.Equal(t, []string{"A"}, *f)
	require.Equal(t, []string{"B"}, *g)

	okCmd(t, spec, init, []string{"A", "B", "C"})
	require.Equal(t, []string{"A", "B"}, *f)
	require.Equal(t, []string{"C"}, *g)

	okCmd(t, spec, init, []string{"A", "B", "C", "D"})
	require.Equal(t, []string{"A", "B", "C"}, *f)
	require.Equal(t, []string{"D"}, *g)
}

func TestSpecC3(t *testing.T) {
	var f, g *[]string
	init := func(c *Cmd) {
		f = c.StringsArg("SRC", nil, "")
		g = c.StringsArg("DST", nil, "")
	}
	spec := "(SRC... DST) | SRC"

	okCmd(t, spec, init, []string{"A"})
	require.Equal(t, []string{"A"}, *f)
	require.Equal(t, 0, len(*g))

	okCmd(t, spec, init, []string{"A", "B"})
	require.Equal(t, []string{"A"}, *f)
	require.Equal(t, []string{"B"}, *g)

	okCmd(t, spec, init, []string{"A", "B", "C"})
	require.Equal(t, []string{"A", "B"}, *f)
	require.Equal(t, []string{"C"}, *g)

	okCmd(t, spec, init, []string{"A", "B", "C", "D"})
	require.Equal(t, []string{"A", "B", "C"}, *f)
	require.Equal(t, []string{"D"}, *g)
}

func TestSpecC5(t *testing.T) {
	var f, g *[]string
	var x *bool
	init := func(c *Cmd) {
		f = c.StringsArg("SRC", nil, "")
		g = c.StringsArg("DST", nil, "")
		x = c.BoolOpt("x", false, "")
	}
	spec := "(SRC... -x DST) | (SRC... DST)"

	okCmd(t, spec, init, []string{"A", "B"})
	require.Equal(t, []string{"A"}, *f)
	require.Equal(t, []string{"B"}, *g)
	require.False(t, *x)

	okCmd(t, spec, init, []string{"A", "B", "C"})
	require.Equal(t, []string{"A", "B"}, *f)
	require.Equal(t, []string{"C"}, *g)
	require.False(t, *x)

	okCmd(t, spec, init, []string{"A", "B", "-x", "C"})
	require.Equal(t, []string{"A", "B"}, *f)
	require.Equal(t, []string{"C"}, *g)
	require.True(t, *x)

}

func TestSpecOptionsEndExplicit(t *testing.T) {
	var x *[]string
	init := func(c *Cmd) {
		x = c.StringsArg("X", nil, "")
	}
	spec := "-- X..."

	okCmd(t, spec, init, []string{"A"})
	require.Equal(t, []string{"A"}, *x)

	okCmd(t, spec, init, []string{"--", "A"})
	require.Equal(t, []string{"A"}, *x)

	okCmd(t, spec, init, []string{"--", "-x"})
	require.Equal(t, []string{"-x"}, *x)

	okCmd(t, spec, init, []string{"--", "A", "B"})
	require.Equal(t, []string{"A", "B"}, *x)
}

func TestSpecOptionsEndImplicit(t *testing.T) {
	var x *[]string
	var f *bool
	init := func(c *Cmd) {
		x = c.StringsArg("X", nil, "")
		f = c.BoolOpt("f", false, "")
	}
	spec := "-f|X..."

	okCmd(t, spec, init, []string{"A"})
	require.Equal(t, []string{"A"}, *x)

	okCmd(t, spec, init, []string{"--", "A"})
	require.Equal(t, []string{"A"}, *x)

	okCmd(t, spec, init, []string{"--", "-f"})
	require.Equal(t, []string{"-f"}, *x)

	okCmd(t, spec, init, []string{"-f"})
	require.True(t, *f)
}

func TestSpecChoiceWithOptionsEndInLastPos(t *testing.T) {
	var x *[]string
	var f *bool
	init := func(c *Cmd) {
		x = c.StringsArg("X", nil, "")
		f = c.BoolOpt("f", false, "")
	}
	spec := "-f|(-- X...)"

	okCmd(t, spec, init, []string{"A", "B"})
	require.Equal(t, []string{"A", "B"}, *x)

	okCmd(t, spec, init, []string{"--", "-f", "B"})
	require.Equal(t, []string{"-f", "B"}, *x)

	okCmd(t, spec, init, []string{"--", "A", "B"})
	require.Equal(t, []string{"A", "B"}, *x)

	okCmd(t, spec, init, []string{"-f"})
	require.True(t, *f)
}

func TestSpecChoiceWithOptionsEnd(t *testing.T) {
	init := func(c *Cmd) {
		c.StringsArg("X", nil, "")
		c.BoolOpt("f", false, "")
	}
	spec := "(-- X...)|-f"

	badSpec(t, spec, init)
}

func TestSpecOptionAfterOptionsEnd(t *testing.T) {
	init := func(c *Cmd) {
		c.StringsArg("X", nil, "")
		c.BoolOpt("f", false, "")
	}

	spec := "-- X... -f"
	badSpec(t, spec, init)
}

func TestSpecOptionAfterOptionsEndInAChoice(t *testing.T) {
	init := func(c *Cmd) {
		c.StringsArg("X", nil, "")
		c.BoolOpt("f", false, "")
		c.BoolOpt("d", false, "")
	}

	spec := "-f | (-- X...) -d"
	badSpec(t, spec, init)
}

func TestSpecOptionAfterOptionalOptionsEnd(t *testing.T) {
	init := func(c *Cmd) {
		c.StringsArg("X", nil, "")
		c.BoolOpt("f", false, "")
		c.BoolOpt("d", false, "")
	}

	spec := "-f [-- X] -d"
	badSpec(t, spec, init)
}

func TestSpecOptionAfterOptionalOptionsEndInAChoice(t *testing.T) {
	init := func(c *Cmd) {
		c.StringsArg("X", nil, "")
		c.BoolOpt("f", false, "")
		c.BoolOpt("d", false, "")
	}

	spec := "(-f | [-- X]) -d"
	badSpec(t, spec, init)
}

func TestSpecOptionAfterOptionsEndIsParsedAsArg(t *testing.T) {
	init := func(c *Cmd) {
		c.StringArg("CMD", "", "")
		c.StringsArg("ARG", nil, "")
	}

	spec := "-- CMD [ARG...]"
	cases := [][]string{
		{"ls"},
		{"ls", "-l"},
		{"ls", "--test"},
		{"ls", "--test=true"},
		{"ls", "--test", "-f"},
	}

	for _, cas := range cases {
		t.Run(fmt.Sprintf("%+v", cas), func(t *testing.T) {
			okCmd(t, spec, init, cas)
		})
	}
}

func TestSpecSingleDash(t *testing.T) {
	var path *string
	var f *bool

	init := func(c *Cmd) {
		path = c.StringArg("PATH", "", "'-' can be used to read from stdin' ")
		f = c.BoolOpt("f", false, "")
	}

	spec := "[-f] PATH"

	okCmd(t, spec, init, []string{"TEST"})
	require.Equal(t, "TEST", *path)
	require.False(t, *f)

	okCmd(t, spec, init, []string{"-f", "TEST"})
	require.Equal(t, "TEST", *path)
	require.True(t, *f)

	okCmd(t, spec, init, []string{"-"})
	require.Equal(t, "-", *path)
	require.False(t, *f)

	okCmd(t, spec, init, []string{"-f", "-"})
	require.Equal(t, "-", *path)
	require.True(t, *f)

	okCmd(t, spec, init, []string{"--", "-"})
	require.Equal(t, "-", *path)
	require.False(t, *f)

	okCmd(t, spec, init, []string{"-f", "--", "-"})
	require.Equal(t, "-", *path)
	require.True(t, *f)
}

func TestSpecOptOrdering(t *testing.T) {
	var a, b, c *bool
	var d, e, f *string

	init := func(cmd *Cmd) {
		a = cmd.BoolOpt("a", false, "")
		b = cmd.BoolOpt("b", false, "")
		c = cmd.BoolOpt("c", false, "")
		d = cmd.StringOpt("d", "", "")

		e = cmd.StringArg("E", "", "")
		f = cmd.StringArg("F", "", "")
	}

	cases := []struct {
		spec    string
		args    []string
		a, b, c bool
		d, e, f string
	}{
		{
			"-a -b",
			[]string{"-a", "-b"},
			true, true, false,
			"", "", "",
		},
		{
			"-a -b",
			[]string{"-b", "-a"},
			true, true, false,
			"", "", "",
		},

		{
			"-a [-b]",
			[]string{"-a"},
			true, false, false,
			"", "", "",
		},
		{
			"-a [-b]",
			[]string{"-b", "-a"},
			true, true, false,
			"", "", "",
		},
		{
			"-a [-b]",
			[]string{"-b", "-a"},
			true, true, false,
			"", "", "",
		},

		{
			"[-a -b]",
			[]string{"-a", "-b"},
			true, true, false,
			"", "", "",
		},
		{
			"[-a -b]",
			[]string{"-b", "-a"},
			true, true, false,
			"", "", "",
		},

		{
			"[-a [-b]]",
			[]string{"-a"},
			true, false, false,
			"", "", "",
		},
		{
			"[-a [-b]]",
			[]string{"-a", "-b"},
			true, true, false,
			"", "", "",
		},
		{
			"[-a [-b]]",
			[]string{"-b", "-a"},
			true, true, false,
			"", "", "",
		},

		{
			"-a | -b -c",
			[]string{"-a", "-c"},
			true, false, true,
			"", "", "",
		},
		{
			"-a | -b -c",
			[]string{"-c", "-a"},
			true, false, true,
			"", "", "",
		},
		{
			"-a | -b -c",
			[]string{"-b", "-c"},
			false, true, true,
			"", "", "",
		},
		{
			"-a | -b -c",
			[]string{"-c", "-b"},
			false, true, true,
			"", "", "",
		},

		{
			"(-a | -b) (-c | -d)",
			[]string{"-a", "-c"},
			true, false, true,
			"", "", "",
		},
		{
			"(-a | -b) (-c | -d)",
			[]string{"-c", "-a"},
			true, false, true,
			"", "", "",
		},

		{
			"(-a | -b) [-c | -d]",
			[]string{"-a"},
			true, false, false,
			"", "", "",
		},
		{
			"(-a | -b) [-c | -d]",
			[]string{"-a", "-c"},
			true, false, true,
			"", "", "",
		},
		{
			"(-a | -b) [-c | -d]",
			[]string{"-d=X", "-b"},
			false, true, false,
			"X", "", "",
		},

		{
			"-a -b E -c -d",
			[]string{"-a", "-b", "E", "-c", "-d", "D"},
			true, true, true,
			"D", "E", "",
		},

		{
			"-a -b E -c -d",
			[]string{"-a", "-b", "E", "-c", "-d", "D"},
			true, true, true,
			"D", "E", "",
		},
		{
			"-a -b E -c -d",
			[]string{"-b", "-a", "E", "-c", "-d", "D"},
			true, true, true,
			"D", "E", "",
		},
		{
			"-a -b E -c -d",
			[]string{"-a", "-b", "E", "-d", "D", "-c"},
			true, true, true,
			"D", "E", "",
		},

		{
			"-a -d...",
			[]string{"-a", "-d", "1"},
			true, false, false,
			"1", "", "",
		},
		{
			"-a -d...",
			[]string{"-d", "1", "-d", "2", "-a"},
			true, false, false,
			"2", "", "",
		},
	}

	for _, cas := range cases {
		t.Run(fmt.Sprintf("%q :: %#v", cas.spec, cas.args), func(t *testing.T) {
			okCmd(t, cas.spec, init, cas.args)
			require.Equal(t, cas.a, *a)
			require.Equal(t, cas.b, *b)
			require.Equal(t, cas.c, *c)
			require.Equal(t, cas.d, *d)
			require.Equal(t, cas.e, *e)
			require.Equal(t, cas.f, *f)
		})
	}

}

func TestSpecOptInlineValue(t *testing.T) {
	var f, g, x *string
	var y *[]string
	init := func(c *Cmd) {
		f = c.StringOpt("f", "", "")
		g = c.StringOpt("giraffe", "", "")
		x = c.StringOpt("x", "", "")
		y = c.StringsOpt("y", nil, "")
	}
	spec := "-x=<wolf-name> [ -f=<fish-name> | --giraffe=<giraffe-name> ] -y=<dog>..."

	okCmd(t, spec, init, []string{"-x=a", "-y=b"})
	require.Equal(t, "a", *x)
	require.Equal(t, []string{"b"}, *y)

	okCmd(t, spec, init, []string{"-x=a", "-y=b", "-y=c"})
	require.Equal(t, "a", *x)
	require.Equal(t, []string{"b", "c"}, *y)

	okCmd(t, spec, init, []string{"-x=a", "-f=f", "-y=b"})
	require.Equal(t, "a", *x)
	require.Equal(t, "f", *f)
	require.Equal(t, []string{"b"}, *y)

	okCmd(t, spec, init, []string{"-x=a", "--giraffe=g", "-y=b"})
	require.Equal(t, "a", *x)
	require.Equal(t, "g", *g)
	require.Equal(t, []string{"b"}, *y)
}

// https://github.com/jawher/mow.cli/issues/28
func TestWardDoesntRunTooSlowly(t *testing.T) {
	init := func(cmd *Cmd) {
		_ = cmd.StringOpt("login", "", "Login for credential, e.g. username or email.")
		_ = cmd.StringOpt("realm", "", "Realm for credential, e.g. website or WiFi AP name.")
		_ = cmd.StringOpt("note", "", "Note for credential.")
		_ = cmd.BoolOpt("no-copy", false, "Do not copy generated password to the clipboard.")
		_ = cmd.BoolOpt("gen", false, "Generate a password.")
		_ = cmd.IntOpt("length", 0, "Password length.")
		_ = cmd.IntOpt("min-length", 30, "Minimum length password.")
		_ = cmd.IntOpt("max-length", 40, "Maximum length password.")
		_ = cmd.BoolOpt("no-upper", false, "Exclude uppercase characters in password.")
		_ = cmd.BoolOpt("no-lower", false, "Exclude lowercase characters in password.")
		_ = cmd.BoolOpt("no-digit", false, "Exclude digit characters in password.")
		_ = cmd.BoolOpt("no-symbol", false, "Exclude symbol characters in password.")
		_ = cmd.BoolOpt("no-similar", false, "Exclude similar characters in password.")
		_ = cmd.IntOpt("min-upper", 0, "Minimum number of uppercase characters in password.")
		_ = cmd.IntOpt("max-upper", -1, "Maximum number of uppercase characters in password.")
		_ = cmd.IntOpt("min-lower", 0, "Minimum number of lowercase characters in password.")
		_ = cmd.IntOpt("max-lower", -1, "Maximum number of lowercase characters in password.")
		_ = cmd.IntOpt("min-digit", 0, "Minimum number of digit characters in password.")
		_ = cmd.IntOpt("max-digit", -1, "Maximum number of digit characters in password.")
		_ = cmd.IntOpt("min-symbol", 0, "Minimum number of symbol characters in password.")
		_ = cmd.IntOpt("max-symbol", -1, "Maximum number of symbol characters in password.")
		_ = cmd.StringOpt("exclude", "", "Exclude specific characters from password.")
	}

	spec := "[--login] [--realm] [--note] [--no-copy] [--gen [--length] [--min-length] [--max-length] [--no-upper] [--no-lower] [--no-digit] [--no-symbol] [--no-similar] [--min-upper] [--max-upper] [--min-lower] [--max-lower] [--min-digit] [--max-digit] [--min-symbol] [--max-symbol] [--exclude]]"

	okCmd(t, spec, init, []string{})
	okCmd(t, spec, init, []string{"--gen", "--length", "42"})
	okCmd(t, spec, init, []string{"--length", "42", "--gen"})
	okCmd(t, spec, init, []string{"--min-length", "10", "--length", "42", "--gen"})
	okCmd(t, spec, init, []string{"--min-length", "10", "--no-symbol", "--no-lower", "--length", "42", "--gen"})

}

func TestEnvOverrideOk(t *testing.T) {
	defer os.Unsetenv("envopt")

	cases := []struct {
		setenv bool
		spec   string
		args   []string
		envval string
	}{
		// pickup the value from the environment variable
		{true, "--envopt --other", []string{"--other", "otheropt"}, "fromenv"},
		{true, "[--envopt] --other", []string{"--other", "otheropt"}, "fromenv"},
		{true, "--envopt", []string{}, "fromenv"},
		{true, "--envopt", []string{"--"}, "fromenv"},

		// override on command line
		{true, "--envopt", []string{"-e", "fromopt"}, "fromopt"},
		{true, "--envopt", []string{"--envopt", "fromopt"}, "fromopt"},

		// no env set
		{false, "--envopt", []string{"--envopt", "fromopt"}, "fromopt"},
		{false, "--envopt", []string{"-e", "fromopt"}, "fromopt"},

		// no env var, fallback to default
		{false, "[--envopt]", []string{}, "envdefault"},
		{false, "[--envopt] --other", []string{"--other", "otheropt"}, "envdefault"},
	}

	for _, cas := range cases {
		t.Run(fmt.Sprintf("%q :: %#v", cas.spec, cas.args), func(t *testing.T) {
			var envopt *string
			var otheropt *string

			init := func(c *Cmd) {
				os.Unsetenv("envopt")
				if cas.setenv {
					os.Setenv("envopt", "fromenv")
				}
				envopt = c.String(StringOpt{
					Name:   "e envopt",
					Value:  "envdefault",
					EnvVar: "envopt",
				})
				if strings.Contains(cas.spec, "other") {
					otheropt = c.StringOpt("o other", "", "")
				}
			}
			okCmd(t, cas.spec, init, cas.args)
			if strings.Contains(cas.spec, "other") {
				// if the test spec defined --other, make sure it was actually set
				assert.Equal(t, "otheropt", *otheropt)
			}
			// ensure --envopt was actually set to the test's expectations
			assert.Equal(t, cas.envval, *envopt)
		})
	}
}

// Anti-regression test for infinite loop case with envvar backed opts
// https://github.com/jawher/mow.cli/pull/49
func TestEnvOptSeq(t *testing.T) {
	defer os.Unsetenv("envopt")

	var envopt *string
	var arg *string

	init := func(c *Cmd) {
		defer os.Unsetenv("envopt")

		envopt = c.String(StringOpt{
			Name:   "e envopt",
			Value:  "envdefault",
			EnvVar: "envopt",
		})

		arg = c.StringArg("ARG", "", "")
	}

	os.Setenv("envopt", "envval")
	okCmd(t, "", init, []string{"argval"})

	assert.Equal(t, "envval", *envopt)
	assert.Equal(t, "argval", *arg)
}

// Test that not setting an environment variable correctly causes
// required options to fail if no value is supplied in args.
func TestEnvOverrideFail(t *testing.T) {
	os.Unsetenv("envopt")

	cases := []struct {
		spec   string
		args   []string
		envval string
	}{
		// no env var, not optional; should fail
		{"--envopt", []string{}, ""},
		{"--envopt --other", []string{"--other", "otheropt"}, ""},
	}

	for _, cas := range cases {
		t.Run(fmt.Sprintf("%q :: %#v", cas.spec, cas.args), func(t *testing.T) {
			init := func(c *Cmd) {
				c.String(StringOpt{
					Name:   "e envopt",
					Value:  "envdefault",
					EnvVar: "envopt",
				})
				if strings.Contains(cas.spec, "other") {
					c.StringOpt("o other", "", "")
				}
			}
			failCmd(t, cas.spec, init, cas.args)
		})
	}
}

func TestJoinStrings(t *testing.T) {
	cases := []struct {
		input    []string
		expected string
	}{
		{nil, ""},
		{[]string{""}, ""},
		{[]string{" "}, ""},
		{[]string{"\t"}, ""},
		{[]string{"", " ", "\t"}, ""},
		{[]string{"a"}, "a"},
		{[]string{"a", "b c"}, "a b c"},
		{[]string{"", "a", " ", "b", "\t"}, "a b"},
	}

	for _, cas := range cases {
		t.Run(fmt.Sprintf("%+v", cas.input), func(t *testing.T) {
			t.Logf("Testing %#v", cas.input)
			actual := joinStrings(cas.input...)

			require.Equal(t, cas.expected, actual)
		})
	}
}

func TestPrintTabbedRow(t *testing.T) {
	cases := []struct {
		s1, s2   string
		expected string
	}{
		{"", "", "  \t\n"},
		{"a", "", "  a\t\n"},
		{"a", "b", "  a\tb\n"},
		{"a", "  b", "  a\tb\n"},
		{"a", "b\n", "  a\tb\n  \t\n"},
		{"a", "b\nc", "  a\tb\n  \tc\n"},
	}

	for _, cas := range cases {
		t.Run(fmt.Sprintf("%q, %q", cas.s1, cas.s2), func(t *testing.T) {
			t.Logf("Testing %q, %q", cas.s1, cas.s2)
			var out strings.Builder
			printTabbedRow(&out, cas.s1, cas.s2)

			require.Equal(t, cas.expected, out.String())
		})
	}
}
