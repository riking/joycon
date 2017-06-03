package consoleiface

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/pkg/errors"
	"github.com/riking/joycon/prog4/jcpc"
)

func filterCtrlZ(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

func (m *Manager) readStdin() {
	l, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[1m[console]\033[m> ",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		FuncFilterInputRune: filterCtrlZ,
	})
	if err != nil {
		fmt.Println("[FATAL] Failed to initialize console")
		panic(err)
	}
	defer l.Close()

	for {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		m.handleCommand(strings.Fields(line))
	}
	close(m.consoleExit)
}

func findCommand(name string) commandMeta {
	for _, v := range commands {
		for _, cName := range v.Aliases {
			if name == cName {
				return v
			}
		}
	}
	return commandMeta{}
}

func (m *Manager) handleCommand(argv []string) {
	if len(argv) == 0 {
		return
	}
	meta := findCommand(argv[0])
	if meta.F == nil {
		fmt.Println("unknown command", argv[0])
		return
	}
	meta.F(m, argv[1:])
}

var rgxSelectUnpaired = regexp.MustCompile(`u([0-9]+)`)
var rgxSelectControllerSide = regexp.MustCompile(`c([0-9]+)([lr]?)`)

func selectJoyCon(m *Manager, argv []string) (jc jcpc.JoyCon, newArgv []string, err error) {
	if len(argv) == 0 {
		return nil, argv, errors.Errorf("No arguments")
	}
	str := argv[0]
	if match := rgxSelectUnpaired.FindStringSubmatch(str); match != nil {
		num, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, argv, errors.Wrap(err, fmt.Sprintf("Could not select JoyCon '%s'", str))
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if num < 1 || num > len(m.unpaired) {
			return nil, argv, errors.Errorf("Unpaired JoyCon number %s out of range (have %d)", str, len(m.unpaired))
		}
		jc = m.unpaired[num-1].jc
		return jc, argv[1:], nil
	} else if match := rgxSelectControllerSide.FindStringSubmatch(str); match != nil {
		num, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, argv, errors.Wrap(err, fmt.Sprintf("Could not select JoyCon '%s'", str))
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if num < 1 || num > len(m.paired) {
			return nil, argv, errors.Errorf("Controller number %s out of range (have %d)", str, len(m.paired))
		}
		c := m.paired[num-1]
		if len(c.jc) == 1 {
			return c.jc[0], argv[1:], nil
		}
		if len(match[2]) == 0 || !(match[2] == "l" || match[2] == "r") {
			return nil, argv, errors.Errorf("Controller number %s is a dual, please specify l/r suffix", str)
		}
		if match[2] == "l" {
			return c.jc[0], argv[1:], nil
		} else {
			return c.jc[1], argv[1:], nil
		}
	} else {
		return nil, argv, errors.Errorf("Not a valid JoyCon selector: '%s'", str)
	}
}

const batteryBarGraph = ""
const colorBad = "\033[1m\033[41m\033[37m"
const colorMid = "\033[1m\033[33m"
const colorGood = "\033[1m\033[32m"
const colorReset = "\033[m"

var batteryStatus = []string{
	"🔋 " + "❓" + " ",
	"🔋 " + colorBad + "▁ " + colorReset,
	"🔋 " + colorBad + "▂ " + colorReset,
	"🔋 " + colorBad + "▃ " + colorReset,
	"🔋 " + colorMid + "▄ " + colorReset,
	"🔋 " + colorMid + "▅ " + colorReset,
	"🔋 " + colorMid + "▆ " + colorReset,
	"🔋 " + colorMid + "▇ " + colorReset,
	"🔋 " + colorGood + "█ " + colorReset,
	"🔋 " + "⚡ ",
}

func renderBattery(l int8) string {
	return batteryStatus[l]
}

func printConnectedJoyCons(m *Manager) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fmt.Println("Connected JoyCons:")
	for i, up := range m.unpaired {
		fmt.Printf("  u%d: %s %s %s\n", i+1, up.jc.Type().String(), up.jc.Serial(), renderBattery(up.jc.Battery()))
	}
	for i, c := range m.paired {
		if len(c.jc) == 2 {
			fmt.Printf("  c%dl: %s %s\n", i+1, c.jc[0].Serial(), renderBattery(c.jc[0].Battery()))
			fmt.Printf("  c%dr: %s %s\n", i+1, c.jc[1].Serial(), renderBattery(c.jc[1].Battery()))
		} else {
			fmt.Printf("  c%d: %s %s %s\n", i+1, c.jc[0].Type().String(), c.jc[0].Serial(), renderBattery(c.jc[0].Battery()))
		}
	}
	fmt.Println()
}

type commandMeta struct {
	F       func(*Manager, []string)
	Aliases []string
	Help    string
}

func (m *commandMeta) Name() string {
	return m.Aliases[0]
}

var commands []commandMeta

func addCommand(F func(*Manager, []string), help string, names ...string) struct{} {
	commands = append(commands, commandMeta{
		F:       F,
		Help:    help,
		Aliases: names,
	})
	return struct{}{}
}

func cmdHelp(m *Manager, argv []string) {
	fmt.Println("Commands:")
	for _, v := range commands {
		fmt.Printf("  %s - %s\n", v.Aliases[0], v.Help)
	}
}

var _ = addCommand(cmdHelp, "Display this help text.", "help", "?", "hlep")
var _ = addCommand(cmdList, "Show the names of all Joy-Cons connected to the system.", "list", "ls")
var _ = addCommand(cmdRecheck, "Recheck for JoyCons connected by the system.", "rescan", "recheck")
var _ = addCommand(cmdDisconnect, "Disconnect the specified JoyCon.", "disconnect")
var _ = addCommand(cmdSetPlayerLights, "Set the player lights on the JoyCon", "setPlayerLights")
var _ = addCommand(cmdSetHomeLights, "Set the home light pattern.", "setHomeLights")

func cmdList(m *Manager, argv []string) {
	printConnectedJoyCons(m)
}

func cmdRecheck(m *Manager, argv []string) {
	m.SearchDevices()
}

func cmdDisconnect(m *Manager, argv []string) {
	jc, argv, err := selectJoyCon(m, argv)
	if err != nil {
		fmt.Println(err)
		return
	}

	jc.Shutdown()
}

func cmdSetPlayerLights(m *Manager, argv []string) {
	jc, argv, err := selectJoyCon(m, argv)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(argv) == 0 {
		fmt.Println("must specify a value: setPlayerLights [jc] 0x8")
		return
	}
	var value uint64
	if strings.HasPrefix(argv[0], "x") {
		value, err = strconv.ParseUint(argv[0][1:], 16, 8)
	} else {
		value, err = strconv.ParseUint(argv[0], 0, 8)
	}
	if err != nil {
		fmt.Println("must specify a value: setPlayerLights [jc] 0x8")
		fmt.Println(err)
		return
	}

	jcpc.SetPlayerLights(jc, byte(value))
}

func cmdSetHomeLights(m *Manager, argv []string) {
	jc, argv, err := selectJoyCon(m, argv)
	if err != nil {
		fmt.Println(err)
		return
	}

	if len(argv) == 0 {
		fmt.Println("must specify a value: setHomeLights [jc] 0x8 0xFF ...")
		return
	}

	pattern := make([]byte, len(argv))
	for i := range pattern {
		val, err := strconv.ParseUint(argv[i], 0, 8)
		if err != nil {
			fmt.Println("invalid number", argv[i], err)
		}
		pattern[i] = byte(val)
	}

	jcpc.SetHomeLightPulse(jc, pattern)
}
