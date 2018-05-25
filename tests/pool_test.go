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
	"github.com/jwells131313/goethe"
	"testing"
	"time"
)

func TestOneFixedPoolFunctionality(t *testing.T) {
	ethe := goethe.GetGoethe()

	funcQueue := ethe.NewBoundedFunctionQueue(10)

	pool, err := ethe.NewPool("OneOnlyPool", 1, 1, 1*time.Minute, funcQueue, nil)
	if err != nil {
		t.Errorf("could not create pool %v", err)
		return
	}
	defer pool.Close()

	if pool.IsStarted() {
		t.Error("A pool just gotten should not be started")
		return
	}

	if pool.GetMinThreads() != 1 {
		t.Errorf("Expected minimum 1 got %d", pool.GetMinThreads())
		return
	}

	if pool.GetMaxThreads() != 1 {
		t.Errorf("Expected maximum 1 got %d", pool.GetMinThreads())
		return
	}

	if pool.GetErrorQueue() != nil {
		t.Errorf("expected nil error queue got %v", pool.GetErrorQueue())
		return
	}

	if pool.GetFunctionQueue() == nil {
		t.Errorf("expected non-nil functional queue")
		return
	}

	if pool.GetCurrentThreadCount() != 0 {
		t.Errorf("should be 0 threads before being started, there were %d", pool.GetCurrentThreadCount())
		return
	}

	if pool.IsClosed() {
		t.Error("pool has not yet been closed")
		return
	}

	err = pool.Start()
	if err != nil {
		t.Errorf("error starting pool %v", err)
		return
	}

	retVals := make(chan int64)

	funcQueue.Enqueue(getTID, retVals)
	funcQueue.Enqueue(getTID, retVals)

	if pool.GetCurrentThreadCount() != 1 {
		t.Errorf("should be 1 threads after being started, there were %d", pool.GetCurrentThreadCount())
		return
	}

	firstReturn := <-retVals
	secondReturn := <-retVals

	if firstReturn != secondReturn {
		t.Errorf("the tids should have been the same, but they were different %d/%d", firstReturn, secondReturn)
		return
	}
}

func TestZeroToOneFixedPoolFunctionality(t *testing.T) {
	ethe := goethe.GetGoethe()

	funcQueue := ethe.NewBoundedFunctionQueue(10)

	pool, err := ethe.NewPool("ZeroOnePool", 0, 1, 1*time.Minute, funcQueue, nil)
	if err != nil {
		t.Errorf("could not create pool %v", err)
		return
	}
	defer pool.Close()

	if pool.IsStarted() {
		t.Error("A pool just gotten should not be started")
		return
	}

	if pool.GetMinThreads() != 0 {
		t.Errorf("Expected minimum 1 got %d", pool.GetMinThreads())
		return
	}

	if pool.GetMaxThreads() != 1 {
		t.Errorf("Expected maximum 1 got %d", pool.GetMinThreads())
		return
	}

	if pool.GetErrorQueue() != nil {
		t.Errorf("expected nil error queue got %v", pool.GetErrorQueue())
		return
	}

	if pool.GetFunctionQueue() == nil {
		t.Errorf("expected non-nil functional queue")
		return
	}

	if pool.GetCurrentThreadCount() != 0 {
		t.Errorf("should be 0 threads before being started, there were %d", pool.GetCurrentThreadCount())
		return
	}

	if pool.IsClosed() {
		t.Error("pool has not yet been closed")
		return
	}

	err = pool.Start()
	if err != nil {
		t.Errorf("error starting pool %v", err)
		return
	}

	retVals := make(chan int64)

	funcQueue.Enqueue(getTID, retVals)
	funcQueue.Enqueue(getTID, retVals)

	firstReturn := <-retVals
	secondReturn := <-retVals

	if firstReturn != secondReturn {
		t.Errorf("the tids should have been the same, but they were different %d/%d", firstReturn, secondReturn)
		return
	}
}

func TestGetMapFunctionality(t *testing.T) {
	ethe := goethe.GetGoethe()

	funcQueue := ethe.NewBoundedFunctionQueue(10)

	poolA, found := ethe.GetPool("Pool A")
	if found {
		t.Error("Should not have found pool not yet created")
		return
	}
	if poolA != nil {
		t.Errorf("poolA should have been nil %v", poolA)
		return
	}

	poolA, err := ethe.NewPool("Pool A", 1, 1, 1*time.Minute, funcQueue, nil)
	if err != nil {
		t.Errorf("should have been able to create pool")
		return
	}
	if poolA == nil {
		t.Errorf("pool a should not have been nil")
		return
	}

	// Run it again
	poolA1, err := ethe.NewPool("Pool A", 1, 1, 1*time.Minute, funcQueue, nil)
	if err == nil {
		t.Errorf("should have gotten exception from existing pool")
		return
	}
	if err != goethe.ErrPoolAlreadyExists {
		t.Errorf("got unexpected error %v", err)
		return
	}
	if poolA1 != poolA {
		t.Errorf("got a different pool? %v/%v", poolA1, poolA)
		return
	}

	poolA1.Close()
	if !poolA1.IsClosed() {
		t.Errorf("We just closed it, but its not closed?")
		return
	}

	poolA, found = ethe.GetPool("Pool A")
	if found {
		t.Error("Should not have found pool not yet created")
		return
	}
	if poolA != nil {
		t.Errorf("poolA should have been nil %v", poolA)
		return
	}
}

func getTID(ret chan int64) {
	ethe := goethe.GetGoethe()

	ret <- ethe.GetThreadID()
}
