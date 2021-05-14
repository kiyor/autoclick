package autoclick

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aybabtme/rgbterm"
)

type Point struct {
	Name  string
	X     int
	Y     int
	Color Hex
}

type RGB struct {
	Red   uint8
	Green uint8
	Blue  uint8
}

type Hex string

func (h Hex) RGB() (RGB, error) {
	hex := strings.Replace(string(h), "#", "", -1)
	var rgb RGB
	values, err := strconv.ParseUint(string(hex), 16, 32)

	if err != nil {
		return RGB{}, err
	}

	rgb = RGB{
		Red:   uint8(values >> 16),
		Green: uint8((values >> 8) & 0xFF),
		Blue:  uint8(values & 0xFF),
	}

	return rgb, nil
}

func NewPoint(x, y int, color string, name ...string) *Point {
	var n string
	if len(name) > 0 {
		n = name[0]
	}
	return &Point{
		Name:  n,
		X:     x,
		Y:     y,
		Color: Hex(color),
	}
}

func (p *Point) String() string {
	rgb, _ := p.Color.RGB()
	c := rgbterm.FgString(string(p.Color), rgb.Red, rgb.Green, rgb.Blue)
	if len(p.Name) > 0 {
		return fmt.Sprintf("%s X:%d Y:%d %s", p.Name, p.X, p.Y, c)
	}
	return fmt.Sprintf("X:%d Y:%d %s", p.X, p.Y, c)
}

type Agent struct {
	mu         *sync.Mutex
	screenshot *string
	PathPrefix string
	SnapGap    time.Duration
	Log        *log.Logger
	snapCmd    string
}

func NewAgent(pathPrefix string, snapGap time.Duration) *Agent {
	empty := ""
	a := &Agent{
		mu:         new(sync.Mutex),
		screenshot: &empty,
		PathPrefix: pathPrefix,
		SnapGap:    snapGap,
		Log:        log.New(os.Stderr, "[autoclick] ", log.LstdFlags|log.Lshortfile),
		snapCmd:    "scrot",
	}
	if runtime.GOOS == "darwin" {
		a.snapCmd = "screencapture"
	}
	a.Scrot()
	go func() {
		for {
			a.Scrot()
			time.Sleep(a.SnapGap)
		}
	}()
	return a
}

func (a *Agent) Scrot() {
	last := *a.screenshot
	now := time.Now().UnixNano()
	path := filepath.Join(a.PathPrefix, fmt.Sprintf("%d.png", now))
	cmd := fmt.Sprintf("%s %s", a.snapCmd, path)
	a.mu.Lock()
	c := exec.Command("/bin/sh", "-c", cmd)
	c.Env = []string{
		"DISPLAY=:0.0",
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Run()
	a.screenshot = &path
	if len(last) > 0 {
		os.Remove(last)
	}
	a.mu.Unlock()
}

func (a *Agent) Close() {
	files, _ := ioutil.ReadDir(a.PathPrefix)
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".png" {
			os.Remove(filepath.Join(a.PathPrefix, f.Name()))
		}
	}
}

func (a *Agent) IsColor(p *Point) bool {
	co := a.GetColor(p)
	if co == p.Color {
		a.Log.Printf(p.String())
		return true
	}
	rgb, _ := co.RGB()
	c := rgbterm.FgString(string(co), rgb.Red, rgb.Green, rgb.Blue)
	a.Log.Printf("%s != %s\n", p.String(), c)
	return false
}

func (a *Agent) GetColor(p *Point) Hex {
	a.mu.Lock()
	defer a.mu.Unlock()
	cmd := fmt.Sprintf("convert %s -depth 8 -crop 1x1+%d+%d txt:- | grep -om1 '#\\w\\+'", *a.screenshot, p.X, p.Y)
	c := exec.Command("/bin/sh", "-c", cmd)
	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = os.Stderr
	c.Run()
	return Hex(strings.Trim(b.String(), "\n"))
}

func (a *Agent) GetMouse() (int, int) {
	cmd := fmt.Sprintf("xdotool getmouselocation --shell")
	c := exec.Command("/bin/sh", "-c", cmd)
	c.Env = []string{
		"DISPLAY=:0.0",
	}
	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = os.Stderr
	c.Run()
	var x, y int
	for _, v := range strings.Split(b.String(), "\n") {
		if strings.HasPrefix(v, "X") {
			x, _ = strconv.Atoi(strings.Split(v, "=")[1])
		}
		if strings.HasPrefix(v, "Y") {
			y, _ = strconv.Atoi(strings.Split(v, "=")[1])
		}
	}
	return x, y
}

func (a *Agent) MoveMouse(p *Point) {
	a.Log.Printf("move to %s X:%d Y:%d\n", p.Name, p.X, p.Y)
	cmd := fmt.Sprintf("xdotool mousemove %d %d", p.X, p.Y)
	c := exec.Command("/bin/sh", "-c", cmd)
	c.Env = []string{
		"DISPLAY=:0.0",
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Run()
}

func (a *Agent) MoveAndClick(p *Point) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.MoveMouse(p)
	time.Sleep(10 * time.Millisecond)
	cmd := fmt.Sprintf("xdotool click 1")
	c := exec.Command("/bin/sh", "-c", cmd)
	c.Env = []string{
		"DISPLAY=:0.0",
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Run()
}
