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

package tests

import (
	"testing"
	"sync"

	"github.com/jwells131313/goethe/utilities"
	"time"
	"github.com/jwells131313/goethe"
)

type simpleValue struct {
	mux sync.Mutex

	value int
}

type throttler struct {
	mux sync.Mutex
	cond *sync.Cond

	proceed bool
}

func TestTwoWritersMutex(t *testing.T) {
	waiter := newSimpleValue()
	throttle := newThrottler()

	goethe := utilities.GetGoethe()
	lock := goethe.NewGoetheLock()

	goethe.Go(func() error {
		incrementValueByOne(lock, waiter, throttle)

		return nil
	})

	goethe.Go(func() error {
		incrementValueByOne(lock, waiter, throttle)

		return nil
	})

	received, gotValue := waiter.waitForValue(5, 1)
	if gotValue != true {
		t.Error("should have gotten to 1 very quickly, got ", received)
	}

	// Only ONE of the threads should get this, so after waiting
	// the value should only be one
	received, gotValue = waiter.waitForValue(2, 2)
	if gotValue {
		t.Error("should not have gotten the value 2", received)
		return
	}

	// Now, let the other thread go
	throttle.release()

	received, gotValue = waiter.waitForValue(5, 2)
	if !gotValue {
		t.Error("should have gotten the value 2", received)
		return
	}

	throttle.release()
}

func incrementValueByOne(lock goethe.Lock, waiter *simpleValue, throttle *throttler) {
	lock.WriteLock()
	defer lock.WriteUnlock()

	waiter.value++

	throttle.wait()
}

func newSimpleValue() *simpleValue {
	retVal := &simpleValue{}

	return retVal
}

func (waiter *simpleValue) waitForValue(seconds, expected int) (int, bool) {
	iterations := seconds * 10

	waiter.mux.Lock()

	for lcv := 0; lcv < iterations; lcv++ {
		if waiter.value == expected {
			waiter.mux.Unlock()
			return waiter.value, true
		}

		waiter.mux.Unlock()
		time.Sleep(100 * time.Millisecond)
		waiter.mux.Lock()
	}

	retVal := waiter.value
	waiter.mux.Unlock()

	if retVal == expected {
		return retVal, true
	}

	return retVal, false
}

func newThrottler() *throttler {
	retVal := &throttler {
		proceed: false,
	}

	retVal.cond = sync.NewCond(&retVal.mux)
	return retVal
}

func (throttle *throttler) release() {
	throttle.mux.Lock()
	defer throttle.mux.Unlock()

	throttle.proceed = true
	throttle.cond.Broadcast()
}

func (throttle *throttler) reset() {
	throttle.mux.Lock()
	defer throttle.mux.Unlock()

	throttle.proceed = false
}

func (throttle *throttler) wait() {
	throttle.mux.Lock()
	defer throttle.mux.Unlock()

	throttle.cond.Wait()
}

