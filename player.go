package main

import (
	"fmt"
	"sync"
	"time"
)

type PlayerOptions struct {
	LeadIn       time.Duration
	TapDuration  time.Duration
	InterKeyGap  time.Duration
	ReleaseDelay time.Duration
}

type Player struct {
	song *Song
	opts PlayerOptions

	mu      sync.Mutex
	playing bool
	stop    chan struct{}
	held    map[string]heldStroke
}

type heldStroke struct {
	Stroke KeyStroke
	Count  int
}

func NewPlayer(song *Song, opts PlayerOptions) *Player {
	if opts.TapDuration <= 0 {
		opts.TapDuration = 22 * time.Millisecond
	}
	if opts.InterKeyGap < 0 {
		opts.InterKeyGap = 0
	}
	if opts.ReleaseDelay <= 0 {
		opts.ReleaseDelay = time.Millisecond
	}

	return &Player{
		song: song,
		opts: opts,
		held: make(map[string]heldStroke),
	}
}

func (p *Player) Toggle() {
	if p.IsPlaying() {
		p.Stop()
		return
	}
	p.Start()
}

func (p *Player) Start() {
	p.mu.Lock()
	if p.playing {
		p.mu.Unlock()
		return
	}
	stop := make(chan struct{})
	p.stop = stop
	p.playing = true
	p.mu.Unlock()

	fmt.Printf("Starting in %s. Focus Roblox now.\n", p.opts.LeadIn.Round(time.Millisecond))
	go p.run(stop)
}

func (p *Player) Stop() {
	p.mu.Lock()
	if !p.playing {
		p.mu.Unlock()
		return
	}
	stop := p.stop
	p.playing = false
	p.stop = nil
	p.mu.Unlock()

	close(stop)
	p.releaseAll()
	fmt.Println("Stopped.")
}

func (p *Player) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

func (p *Player) run(stop <-chan struct{}) {
	defer p.finish(stop)

	start := time.Now().Add(p.opts.LeadIn)
	for index := 0; index < len(p.song.Actions); {
		at := p.song.Actions[index].At
		if !sleepUntil(start.Add(at), stop) {
			return
		}

		end := index + 1
		for end < len(p.song.Actions) && p.song.Actions[end].At == at {
			end++
		}

		if !p.playGroup(p.song.Actions[index:end], stop) {
			return
		}
		index = end
	}

	fmt.Println("Finished.")
}

func (p *Player) finish(stop <-chan struct{}) {
	p.releaseAll()

	p.mu.Lock()
	if p.stop == stop {
		p.playing = false
		p.stop = nil
	}
	p.mu.Unlock()
}

func (p *Player) playGroup(actions []Action, stop <-chan struct{}) bool {
	taps := make([]Action, 0, len(actions))
	for _, action := range actions {
		switch action.Kind {
		case ActionTap:
			taps = append(taps, action)
		case ActionUp:
			p.release(action.Stroke)
			if !sleepInterruptible(p.opts.InterKeyGap, stop) {
				return false
			}
		case ActionDown:
			p.press(action.Stroke)
			if !sleepInterruptible(p.opts.InterKeyGap, stop) {
				return false
			}
		}
	}

	if len(taps) == 0 {
		return true
	}

	for _, action := range taps {
		p.press(action.Stroke)
		if !sleepInterruptible(p.opts.InterKeyGap, stop) {
			return false
		}
	}

	if !sleepInterruptible(p.opts.TapDuration, stop) {
		return false
	}

	for i := len(taps) - 1; i >= 0; i-- {
		p.release(taps[i].Stroke)
		if !sleepInterruptible(p.opts.ReleaseDelay, stop) {
			return false
		}
	}

	return true
}

func (p *Player) press(stroke KeyStroke) {
	pressStroke(stroke)
	p.mu.Lock()
	id := strokeID(stroke)
	held := p.held[id]
	held.Stroke = stroke
	held.Count++
	p.held[id] = held
	p.mu.Unlock()
}

func (p *Player) release(stroke KeyStroke) {
	p.mu.Lock()
	id := strokeID(stroke)
	held, ok := p.held[id]
	if ok && held.Count > 1 {
		held.Count--
		p.held[id] = held
		p.mu.Unlock()
		return
	}
	if ok {
		delete(p.held, id)
	}
	p.mu.Unlock()

	releaseStroke(stroke)
}

func (p *Player) releaseAll() {
	p.mu.Lock()
	held := make([]KeyStroke, 0, len(p.held))
	for _, entry := range p.held {
		held = append(held, entry.Stroke)
	}
	p.held = make(map[string]heldStroke)
	p.mu.Unlock()

	for _, stroke := range held {
		releaseStroke(stroke)
		time.Sleep(p.opts.ReleaseDelay)
	}
}

func strokeID(stroke KeyStroke) string {
	return fmt.Sprintf("%d:%t", stroke.KeyCode, stroke.Shift)
}

func sleepUntil(deadline time.Time, stop <-chan struct{}) bool {
	return sleepInterruptible(time.Until(deadline), stop)
}

func sleepInterruptible(duration time.Duration, stop <-chan struct{}) bool {
	if duration <= 0 {
		select {
		case <-stop:
			return false
		default:
			return true
		}
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-stop:
		return false
	}
}
