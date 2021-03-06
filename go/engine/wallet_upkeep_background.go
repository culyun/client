// Copyright 2018 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

// WalletUpkeepBackground keeps the wallet bundle encrypted for the latest puk.

package engine

import (
	"sync"
	"time"

	"github.com/keybase/client/go/libkb"
	"github.com/keybase/client/go/stellar"
)

var WalletUpkeepBackgroundSettings = BackgroundTaskSettings{
	// Wait after starting the app
	Start: 40 * time.Second,
	// When waking up on mobile lots of timers will go off at once. We wait an additional
	// delay so as not to add to that herd and slow down the mobile experience when opening the app.
	WakeUp: 20 * time.Second,
	// Wait between checks
	Interval: 24 * time.Hour,
	// Time limit on each round
	Limit: 10 * time.Minute,
}

// WalletUpkeepBackground is an engine.
type WalletUpkeepBackground struct {
	libkb.Contextified
	sync.Mutex

	args *WalletUpkeepBackgroundArgs
	task *BackgroundTask
}

type WalletUpkeepBackgroundArgs struct {
	// Channels used for testing. Normally nil.
	testingMetaCh     chan<- string
	testingRoundResCh chan<- error
}

// NewWalletUpkeepBackground creates a WalletUpkeepBackground engine.
func NewWalletUpkeepBackground(g *libkb.GlobalContext, args *WalletUpkeepBackgroundArgs) *WalletUpkeepBackground {
	task := NewBackgroundTask(g, &BackgroundTaskArgs{
		Name:     "WalletUpkeepBackground",
		F:        WalletUpkeepBackgroundRound,
		Settings: WalletUpkeepBackgroundSettings,

		testingMetaCh:     args.testingMetaCh,
		testingRoundResCh: args.testingRoundResCh,
	})
	return &WalletUpkeepBackground{
		Contextified: libkb.NewContextified(g),
		args:         args,
		// Install the task early so that Shutdown can be called before RunEngine.
		task: task,
	}
}

// Name is the unique engine name.
func (e *WalletUpkeepBackground) Name() string {
	return "WalletUpkeepBackground"
}

// GetPrereqs returns the engine prereqs.
func (e *WalletUpkeepBackground) Prereqs() Prereqs {
	return Prereqs{}
}

// RequiredUIs returns the required UIs.
func (e *WalletUpkeepBackground) RequiredUIs() []libkb.UIKind {
	return []libkb.UIKind{}
}

// SubConsumers returns the other UI consumers for this engine.
func (e *WalletUpkeepBackground) SubConsumers() []libkb.UIConsumer {
	return []libkb.UIConsumer{}
}

// Run starts the engine.
// Returns immediately, kicks off a background goroutine.
func (e *WalletUpkeepBackground) Run(ctx *Context) (err error) {
	return RunEngine(e.task, ctx)
}

func (e *WalletUpkeepBackground) Shutdown() {
	e.task.Shutdown()
}

func WalletUpkeepBackgroundRound(g *libkb.GlobalContext, ectx *Context) error {
	if g.ConnectivityMonitor.IsConnected(ectx.GetNetContext()) == libkb.ConnectivityMonitorNo {
		g.Log.CDebugf(ectx.GetNetContext(), "WalletUpkeepBackgroundRound giving up offline")
		return nil
	}

	if !g.LocalSigchainGuard().IsAvailable(ectx.GetNetContext(), "WalletUpkeepBackgroundRound") {
		g.Log.CDebugf(ectx.GetNetContext(), "WalletUpkeepBackgroundRound yielding to guard")
		return nil
	}

	err := stellar.Upkeep(ectx.GetNetContext(), g)
	return err
}
