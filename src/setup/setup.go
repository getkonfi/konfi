package setup

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/getkonfi/konfi/setup/cst"
	"github.com/getkonfi/konfi/theme"

	"github.com/rs/zerolog"
)

type App struct {
	AppName    string
	AppVersion string

	Config   *KonfConfig
	Logger   *zerolog.Logger
	Theme    *theme.Theme
	Detected []Konfable
	Versions map[string]string

	closeFns []Fn
}

// Konfable is the minimal interface setup needs for detected apps.
// full interface lives in konfables/ — this avoids an import cycle.
type Konfable interface {
	Name() string
	ConfigPath() string
}

type Fn func(ctx context.Context, app *App) error

type Unit struct {
	Name    string
	InitFn  Fn
	CloseFn Fn
}

type initStat struct {
	Name       string
	Status     string
	Elapsed    time.Duration
	Heap       int64
	Goroutines int
}

func InitApp(ctx context.Context, units []Unit) (*App, error) {
	app := &App{
		AppName:    cst.AppName,
		AppVersion: cst.AppVersion,
		Versions:   make(map[string]string),
	}

	var stats []initStat
	for _, u := range units {
		m := startMeasurement()
		sts := "OK"

		err := u.InitFn(ctx, app)
		if err != nil {
			sts = "FAIL"
		}

		elapsed, heap, gCount := m.delta()
		stats = append(stats, initStat{
			Name:       u.Name,
			Status:     sts,
			Elapsed:    elapsed,
			Heap:       heap,
			Goroutines: gCount,
		})

		if u.CloseFn != nil {
			app.closeFns = append(app.closeFns, u.CloseFn)
		}

		if err != nil {
			if app.Logger != nil {
				app.Logger.Error().Err(err).Str("unit", u.Name).Msg("init failed")
			}
			app.cleanup()
			return nil, fmt.Errorf("init %s: %w", u.Name, err)
		}
	}

	if app.Logger != nil {
		app.Logger.Info().
			Str("go", runtime.Version()).
			Str("name", app.AppName).
			Str("version", app.AppVersion).
			Msg("init complete")

		for _, s := range stats {
			app.Logger.Debug().
				Str("unit", s.Name).
				Str("status", s.Status).
				Str("elapsed", readableDur(s.Elapsed)).
				Str("heap", readableByte(uint64(s.Heap))).
				Int("goroutines", s.Goroutines).
				Msg("unit init")
		}
	}

	return app, nil
}

// Shutdown runs close functions in reverse order.
func (app *App) Shutdown() {
	if app.Logger != nil {
		app.Logger.Info().Msg("shutting down")
	}
	app.cleanup()
}

// cleanup runs close functions in reverse order.
func (app *App) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for i := len(app.closeFns) - 1; i >= 0; i-- {
		if err := app.closeFns[i](ctx, app); err != nil && app.Logger != nil {
			app.Logger.Error().Err(err).Msg("shutdown error")
		}
	}
}

// --- measurement ---

type measurement struct {
	start  time.Time
	heap   uint64
	gCount int
}

func startMeasurement() *measurement {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return &measurement{
		start:  time.Now(),
		heap:   mem.HeapAlloc,
		gCount: runtime.NumGoroutine(),
	}
}

func (m *measurement) delta() (elapsed time.Duration, heapDelta int64, goroutineDelta int) {
	elapsed = time.Since(m.start)
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	//nolint:gosec // uint64 -> int64 overflow is acceptable for heap deltas
	return elapsed, int64(mem.HeapAlloc - m.heap), runtime.NumGoroutine() - m.gCount
}

// --- formatting ---

func readableByte(b uint64) string {
	if b < 1024 {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := 1024, 0
	for n := b / 1024; n >= 1024; n /= 1024 {
		div *= 1024
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func readableDur(d time.Duration) string {
	switch {
	case d >= time.Second:
		return fmt.Sprintf("%.2fs", d.Seconds())
	case d >= time.Millisecond:
		return fmt.Sprintf("%.1fms", float64(d.Nanoseconds())/1e6)
	case d >= time.Microsecond:
		return fmt.Sprintf("%.1fμs", float64(d.Nanoseconds())/1e3)
	default:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	}
}
