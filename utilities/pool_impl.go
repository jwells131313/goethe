/*
 * DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS HEADER.
 *
 * Copyright (c) 2018 Oracle and/or its affiliates. All rights reserved.
 *
 * The contents of this file are subject to the terms of either the GNU
 * General Public License Version 2 only ("GPL") or the Common Development
 * and Distribution License("CDDL") (collectively, the "License").  You
 * may not use this file except in compliance with the License.  You can
 * obtain a copy of the License at
 * https://glassfish.dev.java.net/public/CDDL+GPL_1_1.html
 * or packager/legal/LICENSE.txt.  See the License for the specific
 * language governing permissions and limitations under the License.
 *
 * When distributing the software, include this License Header Notice in each
 * file and include the License file at packager/legal/LICENSE.txt.
 *
 * GPL Classpath Exception:
 * Oracle designates this particular file as subject to the "Classpath"
 * exception as provided by Oracle in the GPL Version 2 section of the License
 * file that accompanied this code.
 *
 * Modifications:
 * If applicable, add the following below the License Header, with the fields
 * enclosed by brackets [] replaced by your own identifying information:
 * "Portions Copyright [year] [name of copyright owner]"
 *
 * Contributor(s):
 * If you wish your version of this file to be governed by only the CDDL or
 * only the GPL Version 2, indicate your decision by adding "[Contributor]
 * elects to include this software in this distribution under the [CDDL or GPL
 * Version 2] license."  If you don't indicate a single choice of license, a
 * recipient has the option to distribute your version of this file under
 * either the CDDL, the GPL Version 2 or to extend the choice of license to
 * its licensees as provided above.  However, if you add GPL Version 2 code
 * and therefore, elected the GPL Version 2 license, then the option applies
 * only if the new code is made subject to such option by the copyright
 * holder.
 */

package utilities

import (
	"fmt"
	"github.com/jwells131313/goethe"
	"reflect"
	"sync"
	"time"
)

type threadPool struct {
	mux                    sync.Mutex
	name                   string
	started, closed        bool
	minThreads, maxThreads int32
	idleDecay              time.Duration
	functionalQueue        goethe.FunctionQueue
	errorQueue             goethe.ErrorQueue
	parent                 *goetheData

	currentThreads int32
	threadState    map[int64]int
	closeChannel   chan bool
	decayChannel   chan bool
	changeChannel  chan int
	decayTimer     goethe.Timer
}

// states for each thread in the pool
const (
	// WAITING currently waiting on the queue
	WAITING = 0

	// RUNNING current running user code
	RUNNING = 1
)

var (
	errorInterface = reflect.TypeOf((*error)(nil)).Elem()
)

// NewThreadPool creates a thread pool
func newThreadPool(par *goetheData, name string, min, max int32, idle time.Duration,
	fq goethe.FunctionQueue, eq goethe.ErrorQueue) (goethe.Pool, error) {
	if min < 0 {
		return nil, fmt.Errorf("minimum thread count less than zero %d", min)
	}
	if max < 1 {
		return nil, fmt.Errorf("maximum thread count less than one %d", max)
	}
	if min > max {
		return nil, fmt.Errorf("minimum (%d) is greater than maximum (%d)", min, max)
	}
	if fq == nil {
		return nil, fmt.Errorf("pool must have a functional queue")
	}

	retVal := &threadPool{
		name:            name,
		minThreads:      min,
		maxThreads:      max,
		idleDecay:       idle,
		functionalQueue: fq,
		errorQueue:      eq,
		threadState:     make(map[int64]int),
		parent:          par,
		closeChannel:    make(chan bool),
		decayChannel:    make(chan bool),
		changeChannel:   make(chan int),
	}

	timer, err := par.ScheduleWithFixedDelay(0, 1*time.Minute,
		retVal.errorQueue, retVal.ringBell)
	if err != nil {
		return nil, err
	}

	retVal.decayTimer = timer

	return retVal, nil
}

func (threadPool *threadPool) ringBell() {
	defer func() {
		recover()
	}()

	threadPool.decayChannel <- true
}

func (threadPool *threadPool) IsStarted() bool {
	threadPool.mux.Lock()
	defer threadPool.mux.Unlock()

	return threadPool.started
}

func (threadPool *threadPool) Start() error {
	threadPool.mux.Lock()
	defer threadPool.mux.Unlock()

	if threadPool.closed {
		return goethe.ErrPoolClosed
	}

	if threadPool.started {
		return nil
	}

	goether := GetGoethe()

	var lcv int32
	for lcv = 0; lcv < threadPool.minThreads; lcv++ {
		goether.GoWithArgs(threadRunner, threadPool)
		threadPool.currentThreads++
	}

	goether.GoWithArgs(threadPool.monitor)
	threadPool.functionalQueue.SetStateChangeCallback(threadPool.functionalQueueChanged)

	threadPool.started = true

	return nil
}

func (threadPool *threadPool) functionalQueueChanged(fq goethe.FunctionQueue) {
	defer func() {
		recover()
	}()

	if threadPool.IsClosed() {
		return
	}
	if !threadPool.IsStarted() {
		return
	}

	threadPool.changeChannel <- fq.GetSize()
}

func (threadPool *threadPool) GetName() string {
	return threadPool.name
}

func (threadPool *threadPool) GetMinThreads() int32 {
	return threadPool.minThreads
}

func (threadPool *threadPool) GetMaxThreads() int32 {
	return threadPool.maxThreads
}

func (threadPool *threadPool) GetIdleDecayDuration() time.Duration {
	return threadPool.idleDecay
}

func (threadPool *threadPool) GetCurrentThreadCount() int32 {
	threadPool.mux.Lock()
	defer threadPool.mux.Unlock()

	return threadPool.currentThreads
}

func (threadPool *threadPool) GetFunctionQueue() goethe.FunctionQueue {
	return threadPool.functionalQueue
}

func (threadPool *threadPool) GetErrorQueue() goethe.ErrorQueue {
	return threadPool.errorQueue
}

func (threadPool *threadPool) IsClosed() bool {
	threadPool.mux.Lock()
	defer threadPool.mux.Unlock()

	return threadPool.closed
}

func (threadPool *threadPool) Close() {
	threadPool.mux.Lock()
	defer threadPool.mux.Unlock()

	if threadPool.closed {
		return
	}

	threadPool.closed = true

	threadPool.functionalQueue.SetStateChangeCallback(nil)

	threadPool.parent.removePool(threadPool.name)

	threadPool.decayTimer.Cancel()

	close(threadPool.closeChannel)
	close(threadPool.decayChannel)
	close(threadPool.changeChannel)
}

func (threadPool *threadPool) monitor() {
	for {
		if threadPool.IsClosed() {
			return
		}

		threadPool.monitorOnce()

		select {
		case <-threadPool.changeChannel:
			break
		case <-threadPool.closeChannel:
			return
		case <-threadPool.decayChannel:
			break
		}
	}
}

func (threadPool *threadPool) monitorOnce() {
	threadPool.mux.Lock()
	defer threadPool.mux.Unlock()

	if threadPool.functionalQueue.IsEmpty() {
		// nothing to do, individual threads will die at their own rate
		return
	}

	if threadPool.currentThreads >= threadPool.maxThreads {
		// already at limit
		return
	}

	// Are all threads busy?
	allBusy := true
	for _, state := range threadPool.threadState {
		if state == WAITING {
			allBusy = false
			break
		}
	}

	if allBusy {
		// We have to grow!
		goether := GetGoethe()

		// But only do one at a time so as not to
		// get too crazy with uping and downing
		// threads
		goether.GoWithArgs(threadRunner, threadPool)
		threadPool.currentThreads++
	}
}

func threadRunner(threadPool *threadPool) {
	goether := GetGoethe()
	tid := goether.GetThreadID()

	defer deleteMapTid(threadPool, tid)

	for {
		if threadPool.IsClosed() {
			threadPool.mux.Lock()
			threadPool.currentThreads--
			threadPool.mux.Unlock()

			return
		}

		changeMapState(threadPool, tid, WAITING)

		descriptor, err := threadPool.functionalQueue.Dequeue(threadPool.idleDecay)
		if err != nil {
			if err == goethe.ErrEmptyQueue {
				threadPool.mux.Lock()
				if threadPool.currentThreads > threadPool.minThreads {
					// Reduce size of thread pool, but not below minimum
					threadPool.currentThreads--

					threadPool.mux.Unlock()
					return
				}
				threadPool.mux.Unlock()
			} else {
				// Todo: log this error or something?
				threadPool.mux.Lock()
				threadPool.currentThreads--
				threadPool.mux.Unlock()

				return
			}
		} else {
			changeMapState(threadPool, tid, RUNNING)

			argsAsVals, err := GetValues(descriptor.UserCall, descriptor.Args)
			if err != nil {
				// Todo: log this error or something?
				return
			}

			Invoke(descriptor.UserCall, argsAsVals, threadPool.errorQueue)
		}
	}
}

func changeMapState(threadPool *threadPool, tid int64, newState int) {
	threadPool.mux.Lock()
	defer threadPool.mux.Unlock()

	threadPool.threadState[tid] = newState
}

func deleteMapTid(threadPool *threadPool, tid int64) {
	threadPool.mux.Lock()
	defer threadPool.mux.Unlock()

	delete(threadPool.threadState, tid)
}
