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
	"github.com/jwells131313/goethe/utilities"
	"testing"
	"time"
)

type Local struct {
	count int
}

var locals = make([]*Local, 0)

const (
	threadLocalName = "TestThreadLocal"
)

func TestThreadLocalStorage(t *testing.T) {
	goethe := utilities.GetGoethe()

	funcQueue := goethe.NewBoundedFunctionQueue(100)

	pool, err := goethe.NewPool("OneOnlyPool", 2, 2, 1*time.Minute, funcQueue, nil)
	if err != nil {
		t.Errorf("%v", err)
		return
	}
	defer pool.Close()

	err = goethe.EstablishThreadLocal(threadLocalName, initializer, nil)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	err = pool.Start()
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	for lcv := 0; lcv < 100; lcv++ {
		funcQueue.Enqueue(adder)
	}

	for getCurrentCount() < 100 {
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(2 * time.Second)

	finalCount := getCurrentCount()

	if finalCount != 100 {
		t.Errorf("expected 100, got %d", finalCount)
		return
	}
}

func getCurrentCount() int {
	var retVal = 0

	for _, local := range locals {
		retVal += local.count
	}

	return retVal
}

func initializer(tl goethe.ThreadLocal) {
	retVal := &Local{}
	tl.Set(retVal)

	locals = append(locals, retVal)
}

func adder() error {
	goethe := utilities.GetGoethe()

	tl, err := goethe.GetThreadLocal(threadLocalName)
	if err != nil {
		return err
	}

	iface, err := tl.Get()
	if err != nil {
		return err
	}

	local := iface.(*Local)

	local.count++

	return nil
}
