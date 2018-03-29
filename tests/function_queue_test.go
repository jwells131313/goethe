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
	"sync"
	"testing"
	"time"

	ethe "github.com/jwells131313/goethe"
	"github.com/jwells131313/goethe/utilities"
)

func TestBasicFQFunctionality(t *testing.T) {
	goethe := utilities.GetGoethe()

	funcQueue := goethe.NewBoundedFunctionQueue(10)

	info, err := funcQueue.Dequeue(0)
	if err == nil {
		t.Errorf("should not have found anything in newly created queue")
		return
	}
	if err != ethe.ErrEmptyQueue {
		t.Errorf("unexpected error returned %v", err)
		return
	}
	if info != nil {
		t.Errorf("info should have been nil")
		return
	}

	var fout int
	f := func() error {
		fout = 13
		return nil
	}

	err = funcQueue.Enqueue(f)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	info, err = funcQueue.Dequeue(0)
	if err != nil {
		t.Errorf("we just added a value, it should be there %v", err)
		return
	}
	if info == nil {
		t.Errorf("we just added a value, the value itself must not be nil")
		return
	}

	err = info()
	if err != nil {
		t.Errorf("function returned an error")
	}

	retVal := fout

	if retVal != 13 {
		t.Errorf("function returned unexpected value %d", retVal)
	}

	info, err = funcQueue.Dequeue(0)
	if err == nil {
		t.Errorf("after dequing message there should be none left %v", err)
		return
	}
	if err != ethe.ErrEmptyQueue {
		t.Errorf("unexpected error returned %v", err)
		return
	}
	if info != nil {
		t.Errorf("after dequeing message there should be no more functions")
		return
	}

	// The basics work
}

func TestFQCapacityWorks(t *testing.T) {
	goethe := utilities.GetGoethe()

	funcQueue := goethe.NewBoundedFunctionQueue(5)

	var a0, a1, a2, a3, a4 int

	f0 := func() error {
		a0 = 100
		return nil
	}
	f1 := func() error {
		a1 = 101
		return nil
	}
	f2 := func() error {
		a2 = 102
		return nil
	}
	f3 := func() error {
		a3 = 103
		return nil
	}
	f4 := func() error {
		a4 = 104
		return nil
	}

	funcArray := []func() error{f0, f1, f2, f3, f4}

	for lcv := 0; lcv < 5; lcv++ {
		// All of these enqueues should work
		err := funcQueue.Enqueue(funcArray[lcv])
		if err != nil {
			t.Errorf("unexpected failure enqueing up to capacity %v", err)
			return
		}
	}

	fx := func() error {
		return nil
	}

	err := funcQueue.Enqueue(fx)
	if err == nil {
		t.Errorf("should have been an error, we are one past capacity")
		return
	}
	if err != ethe.ErrAtCapacity {
		t.Errorf("unexpected error when adding past capacity: %v", err)
		return
	}

	for lcv := 0; lcv < 5; lcv++ {
		// Make sure we can dequeue multiple messages
		info, err := funcQueue.Dequeue(0)
		if err != nil {
			t.Errorf("should have found item on iteration %d %v", lcv, err)
			return
		}

		info()

		var result int
		switch lcv {
		case 0:
			result = a0
			break
		case 1:
			result = a1
			break
		case 2:
			result = a2
			break
		case 3:
			result = a3
			break
		case 4:
			result = a4
			break
		}

		expected := 100 + lcv

		if result != expected {
			t.Errorf("Did not get expected result on iteration %d, expected %d, got %d", lcv, expected, result)
		}
	}

}

func TestFQEmptyQueueBlocks(t *testing.T) {
	goethe := utilities.GetGoethe()

	funcQueue := goethe.NewBoundedFunctionQueue(10)

	current := time.Now()

	_, err := funcQueue.Dequeue(2 * time.Second)
	if err == nil {
		t.Error("Expected an error after waiting two seconds")
		return
	}
	if err != ethe.ErrEmptyQueue {
		t.Errorf("unexpected exception %v", err)
		return
	}

	elapsed := time.Since(current)
	if elapsed < (2 * time.Second) {
		t.Errorf("should have waited two seconds, only waited %d", elapsed)
		return
	}
}

func TestFQQueueBlocksUntilDataEnqueued(t *testing.T) {
	goethe := utilities.GetGoethe()

	funcQueue := goethe.NewBoundedFunctionQueue(10)

	mux := sync.Mutex{}
	cond := sync.NewCond(&mux)

	finished := false

	f := func() error {
		mux.Lock()
		defer mux.Unlock()

		finished = true

		cond.Broadcast()
		return nil
	}

	current := time.Now()

	errorOutput := make(chan error)

	goethe.Go(func() error {
		g, err := funcQueue.Dequeue(10 * time.Second)
		if err != nil {
			errorOutput <- err

			mux.Lock()
			defer mux.Unlock()

			finished = true
			cond.Broadcast()
			return err
		}

		err = g()
		if g != nil {
			errorOutput <- err

			mux.Lock()
			defer mux.Unlock()

			finished = true
			cond.Broadcast()

			return err
		}

		errorOutput <- nil

		return nil
	})

	time.Sleep(2 * time.Second)

	// nap time is over, send the function
	err := funcQueue.Enqueue(f)
	if err != nil {
		t.Errorf("Could not enqueue function %v", err)
		return
	}

	mux.Lock()
	for !finished {
		cond.Wait()
	}
	mux.Unlock()

	err = <-errorOutput
	if err != nil {
		t.Errorf("error from the thread %v", err)
		return
	}

	elapsed := time.Since(current)

	if elapsed < (2 * time.Second) {
		t.Errorf("Should have waited at least two seconds %d", elapsed)
		return
	}

	if elapsed >= (10 * time.Second) {
		t.Errorf("should have exited due to data way longer ago %d", elapsed)
	}

	t.Logf("Actual elapsedTime %d", elapsed)
}
