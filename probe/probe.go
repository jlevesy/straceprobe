package probe

import (
	"fmt"
	"runtime"
	"syscall"
)

type Sample map[uint64]int

type Probe interface {
	Attach(pid int) error
	Collect() (Sample, error)
	Stop()
}

type event struct {
	CallID uint64
	Error  error
}

type collectResult struct {
	Sample Sample
	Error  error
}

type ptraceProbe struct {
	in   chan *event
	out  chan chan *collectResult
	stop chan struct{}
}

func New(inSize int) Probe {
	return &ptraceProbe{
		in:   make(chan *event, inSize),
		out:  make(chan chan *collectResult),
		stop: make(chan struct{}),
	}
}

func (p *ptraceProbe) Attach(pid int) error {
	go p.collect(pid)
	go p.listen()

	return nil
}

func (p *ptraceProbe) Collect() (Sample, error) {
	out := make(chan *collectResult)
	p.out <- out
	res := <-out
	return res.Sample, res.Error
}

func (p *ptraceProbe) Stop() {
	close(p.stop)
}

func (p *ptraceProbe) collect(pid int) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	err := syscall.PtraceAttach(pid)

	if err != nil {
		p.in <- &event{0, err}
		return
	}

	defer syscall.PtraceDetach(pid)

	_, err = syscall.Wait4(pid, nil, 0, nil)

	if err != nil {
		p.in <- &event{0, err}
		return
	}

	exit := false
	for {
		select {
		case <-p.stop:
			return
		default:
		}

		var regs syscall.PtraceRegs

		if exit {
			err := syscall.PtraceGetRegs(pid, &regs)

			if err != nil {
				p.in <- &event{0, err}
				return
			}

			p.in <- &event{regs.Orig_rax, nil}
		}

		err := syscall.PtraceSyscall(pid, 0)

		if err != nil {
			p.in <- &event{0, err}
			return
		}

		_, err = syscall.Wait4(pid, nil, 0, nil)

		if err != nil {
			p.in <- &event{0, err}
			return
		}

		exit = !exit
	}
}

func (p *ptraceProbe) listen() {
	sample := make(Sample)
	var err error
	for {
		select {
		case evt := <-p.in:
			err = evt.Error
			if evt.Error == nil {
				sample[evt.CallID]++
			}

		case out := <-p.out:
			out <- &collectResult{sample, err}
			sample = make(Sample)
			err = nil
		case <-p.stop:
			return
		}
	}
}
