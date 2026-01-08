package utils

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Spinner represents an animated spinner
type Spinner struct {
	frames   []string
	message  string
	interval time.Duration
	active   bool
	mu       sync.Mutex
	done     chan bool
}

// SpinnerStyle represents different spinner styles
type SpinnerStyle int

const (
	SpinnerDots SpinnerStyle = iota
	SpinnerLine
	SpinnerCircle
	SpinnerArrow
	SpinnerBounce
	SpinnerGrow
)

var spinnerFrames = map[SpinnerStyle][]string{
	SpinnerDots:   {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	SpinnerLine:   {"-", "\\", "|", "/"},
	SpinnerCircle: {"◐", "◓", "◑", "◒"},
	SpinnerArrow:  {"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
	SpinnerBounce: {"⠁", "⠂", "⠄", "⠂"},
	SpinnerGrow:   {"▁", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃"},
}

// NewSpinner creates a new spinner with default style
func NewSpinner(message string) *Spinner {
	return NewSpinnerWithStyle(message, SpinnerDots)
}

// NewSpinnerWithStyle creates a new spinner with specified style
func NewSpinnerWithStyle(message string, style SpinnerStyle) *Spinner {
	frames, ok := spinnerFrames[style]
	if !ok {
		frames = spinnerFrames[SpinnerDots]
	}

	return &Spinner{
		frames:   frames,
		message:  message,
		interval: 80 * time.Millisecond,
		done:     make(chan bool),
	}
}

// Start starts the spinner
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go func() {
		frameIdx := 0
		for {
			select {
			case <-s.done:
				return
			default:
				s.mu.Lock()
				if !s.active {
					s.mu.Unlock()
					return
				}
				frame := s.frames[frameIdx]
				message := s.message
				s.mu.Unlock()

				fmt.Fprintf(os.Stderr, "\r%s %s", ColorCyan(frame), message)
				frameIdx = (frameIdx + 1) % len(s.frames)
				time.Sleep(s.interval)
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	s.active = false
	s.done <- true
	fmt.Fprintf(os.Stderr, "\r%s\r", clearLine(len(s.message)+3))
}

// Success stops the spinner with a success message
func (s *Spinner) Success(message string) {
	s.Stop()
	if message == "" {
		message = s.message
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", ColorGreen(IconSuccess), message)
}

// Error stops the spinner with an error message
func (s *Spinner) Error(message string) {
	s.Stop()
	if message == "" {
		message = s.message
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", ColorRed(IconError), message)
}

// Warning stops the spinner with a warning message
func (s *Spinner) Warning(message string) {
	s.Stop()
	if message == "" {
		message = s.message
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", ColorYellow(IconWarning), message)
}

// UpdateMessage updates the spinner message
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// SetInterval sets the animation interval
func (s *Spinner) SetInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interval = interval
}

func clearLine(length int) string {
	return fmt.Sprintf("%*s", length, "")
}

// Progress represents a progress bar
type Progress struct {
	total     int64
	current   int64
	width     int
	message   string
	startTime time.Time
	mu        sync.Mutex
}

// NewProgress creates a new progress bar
func NewProgress(total int64, message string) *Progress {
	return &Progress{
		total:     total,
		current:   0,
		width:     40,
		message:   message,
		startTime: time.Now(),
	}
}

// Update updates the progress
func (p *Progress) Update(current int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = current
	p.render()
}

// Increment increments the progress
func (p *Progress) Increment(amount int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current += amount
	if p.current > p.total {
		p.current = p.total
	}
	p.render()
}

// SetMessage updates the message
func (p *Progress) SetMessage(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.message = message
}

// Finish completes the progress bar
func (p *Progress) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = p.total
	p.render()
	fmt.Fprintln(os.Stderr)
}

func (p *Progress) render() {
	percentage := float64(p.current) / float64(p.total)
	if p.total == 0 {
		percentage = 0
	}

	filled := int(float64(p.width) * percentage)
	empty := p.width - filled

	bar := fmt.Sprintf("[%s%s]",
		ColorCyan(repeatString("█", filled)),
		repeatString("░", empty))

	elapsed := time.Since(p.startTime)
	speed := float64(p.current) / elapsed.Seconds()
	eta := time.Duration(0)
	if speed > 0 {
		remaining := float64(p.total - p.current)
		eta = time.Duration(remaining/speed) * time.Second
	}

	fmt.Fprintf(os.Stderr, "\r%s %s %3.0f%% %s/%s %s",
		p.message,
		bar,
		percentage*100,
		formatBytes(p.current),
		formatBytes(p.total),
		formatDuration(eta))
}

func repeatString(s string, count int) string {
	if count <= 0 {
		return ""
	}
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "0s"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// MultiProgress manages multiple progress bars
type MultiProgress struct {
	bars []*Progress
	mu   sync.Mutex
}

// NewMultiProgress creates a new multi-progress manager
func NewMultiProgress() *MultiProgress {
	return &MultiProgress{
		bars: make([]*Progress, 0),
	}
}

// AddBar adds a new progress bar
func (mp *MultiProgress) AddBar(total int64, message string) *Progress {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	bar := NewProgress(total, message)
	mp.bars = append(mp.bars, bar)
	return bar
}

// Finish finishes all progress bars
func (mp *MultiProgress) Finish() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	for _, bar := range mp.bars {
		bar.Finish()
	}
}
