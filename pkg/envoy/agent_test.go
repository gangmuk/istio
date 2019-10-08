// Copyright 2017 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package envoy

import (
	"context"
	"testing"
	"time"
)

// TestProxy sample struct for proxy
type TestProxy struct {
	run     func(interface{}, int, <-chan error) error
	cleanup func(int)
}

func (tp TestProxy) Run(config interface{}, epoch int, stop <-chan error) error {
	return tp.run(config, epoch, stop)
}

func (tp TestProxy) Cleanup(epoch int) {
	if tp.cleanup != nil {
		tp.cleanup(epoch)
	}
}

// TestStartExit starts a proxy and ensures the agent exits once the proxy exits
func TestStartExit(t *testing.T) {
	ctx := context.Background()
	done := make(chan struct{})
	a := NewAgent(TestProxy{
		func(config interface{}, i2 int, abort <-chan error) error { return nil },
		func(i int) {}},
		0)
	go func() {
		a.Run(ctx)
		done <- struct{}{}
	}()
	a.Restart("config")
	<-done
}

// TestStartDrain tests basic start, termination sequence
//   * Runs with passed config
//   * Terminate is called
//   * Runs with drain config
//   * Aborts all proxies
func TestStartDrain(t *testing.T) {
	wantEpoch := 0
	proxiesStarted, wantProxiesStarted := 0, 2
	blockChan := make(chan interface{})
	ctx, cancel := context.WithCancel(context.Background())
	startConfig := "start config"
	start := func(config interface{}, currentEpoch int, _ <-chan error) error {
		t.Logf("Start called with config: %v", config)
		proxiesStarted++
		if currentEpoch != wantEpoch {
			t.Errorf("start wanted epoch %v, got %v", wantEpoch, currentEpoch)
		}
		wantEpoch = currentEpoch + 1
		blockChan <- "unblock"
		if currentEpoch == 0 {
			<-ctx.Done()
			if config != startConfig {
				t.Errorf("start wanted config %q, got %q", startConfig, config)
			}
			time.Sleep(time.Second * 2) // ensure initial proxy doesn't terminate too quickly
		} else if currentEpoch == 1 {
			if _, ok := config.(DrainConfig); !ok {
				t.Errorf("start expected draining config, got %q", config)
			}
		}
		return nil
	}
	a := NewAgent(TestProxy{start, nil}, -10*time.Second)
	go a.Run(ctx)
	a.Restart(startConfig)
	<-blockChan
	cancel()
	<-blockChan
	<-ctx.Done()

	if proxiesStarted != wantProxiesStarted {
		t.Errorf("expected %v proxies to be started, got %v", wantProxiesStarted, proxiesStarted)
	}
}

// TestApplyTwice tests that scheduling the same config does not trigger a restart
func TestApplyTwice(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	desired := "config"
	applyCount := 0
	start := func(config interface{}, epoch int, _ <-chan error) error {
		if epoch == 1 && applyCount < 2 {
			t.Error("Should start only once for same config")
		}
		<-ctx.Done()
		return nil
	}
	cleanup := func(epoch int) {}
	a := NewAgent(TestProxy{start, cleanup}, -10*time.Second)
	go a.Run(ctx)
	a.Restart(desired)
	applyCount++
	a.Restart(desired)
	applyCount++
	cancel()
}

// TestStartTwiceStop applies three configs and validates that cleanups are called in order
func TestStartTwiceStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	stop0 := make(chan struct{})
	stop1 := make(chan struct{})
	desired0 := "config0"
	desired1 := "config1"
	desired2 := "config2"
	start := func(config interface{}, epoch int, _ <-chan error) error {
		if config == desired0 && epoch == 0 {
			<-stop0
		} else if config == desired1 && epoch == 1 {
			<-stop1
		} else if config == desired2 && epoch == 2 {
			close(stop1)
			<-ctx.Done()
		} else if _, ok := config.(DrainConfig); !ok { // don't need to validate draining proxy here
			t.Errorf("Unexpected start %v, epoch %d", config, epoch)
			cancel()
		}
		return nil
	}
	finished0, finished1, finished2 := false, false, false
	cleanup := func(epoch int) {
		// epoch 1 finishes before epoch 0
		if epoch == 1 {
			finished1 = true
			if finished0 || finished2 {
				t.Errorf("Expected epoch 1 to be first to finish")
			}
			close(stop0)
		} else if epoch == 0 {
			finished0 = true
			if !finished1 || finished2 {
				t.Errorf("Expected epoch 0 to be second to finish")
			}
			cancel()
		} else if epoch == 2 {
			finished2 = true
			if !finished0 || !finished1 {
				t.Errorf("Expected epoch 2 to be last to finish")
			}
		} else {
			// epoch 3 is the drain epoch
			if epoch != 3 {
				t.Errorf("Unexpected epoch %d in cleanup", epoch)
			}
			cancel()
		}
	}
	a := NewAgent(TestProxy{start, cleanup}, 0)
	go a.Run(ctx)
	a.Restart(desired0)
	a.Restart(desired1)
	a.Restart(desired2)
	<-ctx.Done()
}

// TestRecovery tests that recovery is applied once
func TestRecovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	desired := "config"
	failed := false
	start := func(config interface{}, epoch int, _ <-chan error) error {
		if epoch == 0 && !failed {
			failed = true
			return nil
		}
		if epoch > 0 {
			t.Errorf("Should not reconcile after success")
		}
		<-ctx.Done()
		return nil
	}
	a := NewAgent(TestProxy{start, func(_ int) {}}, 0)
	go a.Run(ctx)
	a.Restart(desired)

	// make sure we don't try to reconcile twice
	<-time.After(100 * time.Millisecond)
	cancel()
}
