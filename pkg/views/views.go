package views

import (
	"fmt"
	"sync/atomic"

	"github.com/pterm/pterm"
)

type TerminalView struct {
	nbTotalLogStreams      atomic.Int64
	nbLogStreamsFound      atomic.Int64
	nbStreamsScanned       atomic.Int64
	spinnerRetrieveStreams *pterm.SpinnerPrinter
	spinnerScanStreams     *pterm.SpinnerPrinter
}

func NewTerminalView() *TerminalView {
	return &TerminalView{}
}

func (v *TerminalView) IncNbLogStreams() {
	v.nbTotalLogStreams.Add(1)
	v.UpdateSpinnerRetrieveLogStreams()
}

func (v *TerminalView) IncNbStreamsScanned() {
	v.nbStreamsScanned.Add(1)
	v.UpdateSpinnerScanLogStreams()
}

func (v *TerminalView) IncNbLogStreamsFound() {
	v.nbLogStreamsFound.Add(1)
	v.UpdateSpinnerRetrieveLogStreams()
}

func (v *TerminalView) StartSpinnerRetrieveLogStreams() {
	v.spinnerRetrieveStreams, _ = pterm.DefaultSpinner.Start("Retrieve log streams")
}

func (v *TerminalView) UpdateSpinnerRetrieveLogStreams() {
	v.spinnerRetrieveStreams.UpdateText(fmt.Sprintf("Retrieve log streams... %d - found %d", v.nbTotalLogStreams.Load(), v.nbLogStreamsFound.Load()))
}

func (v *TerminalView) StartSpinnerScanLogStreams() error {
	var err error
	v.spinnerScanStreams, err = pterm.DefaultSpinner.Start("Retrieve events")
	return err
}

func (v *TerminalView) UpdateSpinnerScanLogStreams() {
	v.spinnerScanStreams.UpdateText(fmt.Sprintf("Retrieve events of log streams... %d/%d", v.nbStreamsScanned.Load(), v.nbLogStreamsFound.Load()))
}

func (v *TerminalView) UpdateSpinnerRetrieveLogStreamsWithText(text string) {
	if v.spinnerRetrieveStreams != nil {
		v.spinnerRetrieveStreams.UpdateText(text)
	}
}

func (v *TerminalView) StopSpinnerRetrieveLogStreams() {
	v.spinnerRetrieveStreams.Success("Log streams retrieved")
}

func (v *TerminalView) StopSpinnerScanLogStreams() {
	v.spinnerScanStreams.Success("Log streams scanned")
}
