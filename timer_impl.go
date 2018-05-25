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

package goethe

import (
	"fmt"
	"reflect"
	"sync"
	"time"
)

const (
	fudgeFactor time.Duration = 10 * time.Millisecond
)

type timerImpl interface {
	run()

	addJob(
		initialDelay time.Duration,
		period time.Duration,
		errorQueue ErrorQueue,
		method interface{},
		arguments []reflect.Value,
		fixed bool) (Timer, error)
}

type timerData struct {
	mux           Lock
	cond          *sync.Cond
	heap          heapQueue
	nextJobNumber int64
	sleepy        sleeper
	nextJob       uint64
}

type nextJob struct {
	jobNumber uint64
}

type timerJob struct {
	mux         sync.Mutex
	initialTime *time.Time
	cancelled   bool
	delay       time.Duration
	fixed       bool
	method      interface{}
	args        []reflect.Value
	errors      ErrorQueue

	next *nextJob
}

// NewTimer creates a timer for use with the goethe scheduler
func newTimer() timerImpl {
	goethe := GetGoethe()

	retVal := &timerData{
		mux:    goethe.NewGoetheLock(),
		heap:   newHeap(),
		sleepy: newSleeper(),
	}

	retVal.cond = sync.NewCond(retVal.mux)

	return retVal
}

func (timer *timerData) run() {
	for timer.runOne() {
	}
}

func (timer *timerData) runOne() bool {
	goethe := GetGoethe()

	timer.mux.Lock()
	defer timer.mux.Unlock()

	peek, peekNode, found := timer.heap.Peek()
	if !found {
		timer.cond.Wait()

		return true
	}

	now := time.Now()
	now = now.Add(fudgeFactor)

	pNode := peekNode.(*timerJob)

	// Is it inside the range
	until := (*peek).Sub(now)
	if until >= 0 {
		timer.sleepy.sleep(until, timer.cond, pNode.next.jobNumber)

		timer.cond.Wait()

		return true
	}

	_, payload, found := timer.heap.Get()
	if !found {
		// should never happen because peek said it was there
		return true
	}

	job := payload.(*timerJob)

	// Ok, time to actually run the job!
	if !job.IsRunning() {
		// cancelled, schedule next and go
		timer.scheduleNextWakeUp()

		return true
	}

	if job.fixed {
		// calculate the next time
		sinceInitial := now.Sub(*job.initialTime)

		numRuns := sinceInitial / job.delay

		nextIteration := numRuns + 1

		nextOffset := nextIteration * job.delay

		nextRunTime := job.initialTime.Add(nextOffset)

		timer.scheduleNext(job, &nextRunTime)
	}

	goethe.Go(timer.invoke, goethe, job)

	// schedule next guy to go
	timer.scheduleNextWakeUp()

	return true
}

func (timer *timerData) scheduleNextWakeUp() {
	timer.mux.Lock()
	defer timer.mux.Unlock()

	peek, peekNode, found := timer.heap.Peek()
	if !found {
		return
	}

	pNode := peekNode.(*timerJob)

	if pNode.next == nil {
		// Should not be possible, but account for it
		return
	}

	until := time.Until(*peek)
	timer.sleepy.sleep(until, timer.cond, pNode.next.jobNumber)
}

func (timer *timerData) invoke(ethe Goethe, job *timerJob) {
	tl, err := ethe.GetThreadLocal(TimerThreadLocal)
	if err != nil {
		if job.errors != nil {
			ei := newErrorinformation(ethe.GetThreadID(), fmt.Errorf("could not find TimerThreadLocal"))
			job.errors.Enqueue(ei)
		}

		return
	}

	tl.Set(job)

	invoke(job.method, job.args, job.errors)

	if job.fixed {
		// parent put new job on
		return
	}

	nextRun := time.Now().Add(job.delay)

	timer.scheduleNext(job, &nextRun)
}

func (timer *timerData) addJob(
	initialDelay time.Duration,
	period time.Duration,
	errorQueue ErrorQueue,
	method interface{},
	arguments []reflect.Value,
	fixed bool) (Timer, error) {
	ethe := GetGoethe()

	now := time.Now()
	added := now.Add(initialDelay)

	retVal := &timerJob{
		initialTime: &added,
		delay:       period,
		fixed:       fixed,
		method:      method,
		args:        arguments,
		errors:      errorQueue,
	}

	_, err := ethe.Go(timer.scheduleNext, retVal, &added)
	if err != nil {
		return nil, err
	}

	return retVal, nil
}

func (timer *timerData) scheduleNext(job *timerJob, nextRingTime *time.Time) error {
	timer.mux.Lock()
	defer timer.mux.Unlock()

	nextJobNumber := timer.nextJob
	timer.nextJob++

	nextRing := &nextJob{
		jobNumber: nextJobNumber,
	}

	job.next = nextRing

	err := timer.heap.Add(nextRingTime, job)
	if err != nil {
		return err
	}

	timer.cond.Broadcast()

	return nil
}

// Cancel cancels the timer
func (job *timerJob) Cancel() {
	job.mux.Lock()
	defer job.mux.Unlock()

	job.cancelled = true
}

// IsRunning true if this timer is running, false if it has been cancelled
func (job *timerJob) IsRunning() bool {
	job.mux.Lock()
	defer job.mux.Unlock()

	return !job.cancelled
}

// GetErrorQueue returns the error queue associated with this timer (may be nil)
func (job *timerJob) GetErrorQueue() ErrorQueue {
	return job.errors
}
