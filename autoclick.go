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
)

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

func (a *Agent) IsColor(x, y int, color string) bool {
	co := a.GetColor(x, y)
	if co == color {
		a.Log.Printf("X=%d && Y=%d && COLOR=%s\n", x, y, color)
		return true
	}
	a.Log.Printf("X=%d && Y=%d && COLOR=%s != %s\n", x, y, co, color)
	return false
}

func (a *Agent) GetColor(x, y int) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	cmd := fmt.Sprintf("convert %s -depth 8 -crop 1x1+%d+%d txt:- | grep -om1 '#\\w\\+'", *a.screenshot, x, y)
	c := exec.Command("/bin/sh", "-c", cmd)
	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = os.Stderr
	c.Run()
	return strings.Trim(b.String(), "\n")
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

func (a *Agent) MoveMouse(x, y int) {
	a.Log.Printf("move X=%d Y=%d\n", x, y)
	cmd := fmt.Sprintf("xdotool mousemove %d %d", x, y)
	c := exec.Command("/bin/sh", "-c", cmd)
	c.Env = []string{
		"DISPLAY=:0.0",
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Run()
}

func (a *Agent) MoveAndClick(x, y int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.MoveMouse(x, y)
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
