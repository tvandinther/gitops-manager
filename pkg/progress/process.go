package progress

import (
	"context"
	"sync"
	"time"
)

type ProcessReporter struct {
	*Reporter
	ProgressCount     int
	TotalCount        int
	Template          ProcessTemplate
	doneContext       context.Context
	cancel            context.CancelFunc
	progressionPeriod time.Duration
	wg                sync.WaitGroup
}

type ProcessReporterOptions struct {
	ReportPeriod   time.Duration
	Template       ProcessTemplate
	TotalFileCount int
}

type ProcessTemplate struct {
	PresentAction string // Present tense of the action e.g. "processing"
	PastAction    string // Past tense of the action e.g. "processed"
	Subject       string // The subject being processed in plural form e.g. "files"
}

func (r *Reporter) NewProcess(opts *ProcessReporterOptions) *ProcessReporter {
	processReporter := &ProcessReporter{
		ProgressCount:     0,
		TotalCount:        opts.TotalFileCount,
		Template:          opts.Template,
		Reporter:          r,
		progressionPeriod: opts.ReportPeriod,
		wg:                sync.WaitGroup{},
	}

	return processReporter
}

func (p *ProcessReporter) Start(ctx context.Context) {
	p.doneContext, p.cancel = context.WithCancel(ctx)

	p.Reporter.Progress("%s %d %s", p.Template.PresentAction, p.TotalCount, p.Template.Subject)
	ticker := time.NewTicker(5 * time.Second)
	p.wg.Add(1)

	go func() {
		defer ticker.Stop()
		defer p.wg.Done()

		for {
			select {
			case <-ticker.C:
				p.sendProgress()
			case <-p.doneContext.Done():
				p.sendProgress() // Send final processed count
				return
			}
		}
	}()
}

func (p *ProcessReporter) Done() {
	p.cancel()
	p.wg.Wait()
}

func (p *ProcessReporter) Increment(delta int) {
	p.ProgressCount += delta
}

func (p *ProcessReporter) sendProgress() {
	p.Reporter.Progress("%s %d/%d %s", p.Template.PastAction, p.ProgressCount, p.TotalCount, p.Template.Subject)
}
