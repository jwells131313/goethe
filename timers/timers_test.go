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

package timers

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
	"time"
)

func TestTimers(t *testing.T) {
	timer := NewTimerHeap(nil)
	assert.NotNil(t, timer)

	assert.True(t, timer.IsRunning())

	timer.Cancel()

	assert.False(t, timer.IsRunning())
}

func getMillisecondDuration(nMillis int) time.Duration {
	return time.Duration(nMillis) * time.Millisecond
}

func TestTimersGoOff(t *testing.T) {
	timer := NewTimerHeap(nil)
	assert.NotNil(t, timer)

	var ring1, ring2, ring3 int32

	var count int32
	job1, err := timer.AddJobByDuration(getMillisecondDuration(400), func() {
		ring1 = atomic.AddInt32(&count, 1)
	})
	if !assert.Nil(t, err, "Failed due to %s", err) {
		return
	}

	job2, err := timer.AddJobByDuration(getMillisecondDuration(300), func() {
		ring2 = atomic.AddInt32(&count, 1)
	})

	job3, err := timer.AddJobByDuration(getMillisecondDuration(500), func() {
		ring3 = atomic.AddInt32(&count, 1)
	})

	time.Sleep(getMillisecondDuration(1000))

	assert.Equal(t, int32(2), ring1)
	assert.Equal(t, int32(1), ring2)
	assert.Equal(t, int32(3), ring3)

	assert.False(t, job1.IsRunning())
	assert.False(t, job2.IsRunning())
	assert.False(t, job3.IsRunning())
}

const expectedErrorString = "Expected One"

func TestErrorChannel(t *testing.T) {
	eChan := make(chan error)
	timer := NewTimerHeap(eChan)

	timer.AddJobByTime(time.Now(), func() error {
		return fmt.Errorf(expectedErrorString)
	})

	err := <-timer.GetErrorChannel()
	if !assert.NotNil(t, err) {
		return
	}

	assert.Equal(t, expectedErrorString, err.Error())
}

const iterations int32 = 20

func TestLotsAtOnce(t *testing.T) {
	eChan := make(chan error)
	timer := NewTimerHeap(eChan)

	timeToFire := time.Now().Add(time.Duration(500) * time.Millisecond)
	for lcv := int32(0); lcv < iterations; lcv++ {
		timer.AddJobByTime(timeToFire, doIncrement)
	}

	// Should really all be done in 1 second
	time.Sleep(getMillisecondDuration(1000))

	assert.Equal(t, iterations, incrementMe)
}

var incrementMe int32

func doIncrement() error {
	atomic.AddInt32(&incrementMe, 1)
	return nil
}

func TestCancelledNotRun(t *testing.T) {
	timer := NewTimerHeap(nil)

	var didRun bool
	job, err := timer.AddJobByDuration(time.Duration(1)*time.Second, func() {
		didRun = true
	})
	if !assert.Nil(t, err) {
		return
	}

	job.Cancel()

	time.Sleep(time.Duration(2) * time.Second)

	assert.False(t, didRun)
}
