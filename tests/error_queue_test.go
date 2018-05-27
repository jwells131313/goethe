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
	"errors"
	"github.com/jwells131313/goethe"
	"testing"
)

func TestBasicErrorFunctionality(t *testing.T) {
	errorQueue := goethe.NewBoundedErrorQueue(10)

	info, found := errorQueue.Dequeue()
	if found {
		t.Errorf("should not have found anything in newly created queue %v", info)
		return
	}
	if info != nil {
		t.Errorf("info should have been nil, it was %v", info)
		return
	}

	errorInfo := &dummyErrorInformation{
		tid: 10,
		err: errors.New("an error"),
	}

	err := errorQueue.Enqueue(errorInfo)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	info, found = errorQueue.Dequeue()
	if !found {
		t.Errorf("we just added a value, it should be there %v", info)
		return
	}
	if info == nil {
		t.Errorf("we just added a value, the value itself must not be nil %v", info)
		return
	}

	if info.GetThreadID() != 10 {
		t.Errorf("expected 10 as tid, got %d", info.GetThreadID())
		return
	}
	if info.GetError() == nil {
		t.Errorf("expected a non-nil error")
		return
	}

	if info.GetError().Error() != "an error" {
		t.Errorf("expected \"an error\" but got %s", info.GetError().Error())
		return
	}

	info, found = errorQueue.Dequeue()
	if found {
		t.Errorf("after dequing message there should be none left %v", info)
		return
	}
	if info != nil {
		t.Errorf("after dequeing message there should be no error info %v", info)
		return
	}

	// The basics work
}

func TestCapacityWorks(t *testing.T) {
	errorQueue := goethe.NewBoundedErrorQueue(10)

	errorInfo := &dummyErrorInformation{
		tid: 10,
		err: errors.New("an error"),
	}

	for lcv := 0; lcv < 10; lcv++ {
		// All of these enqueues should work
		err := errorQueue.Enqueue(errorInfo)
		if err != nil {
			t.Errorf("unexpected failure enqueing up to capacity %v", err)
			return
		}
	}

	err := errorQueue.Enqueue(errorInfo)
	if err == nil {
		t.Errorf("should have been an error, we are one past capacity")
		return
	}
	if err != goethe.ErrAtCapacity {
		t.Errorf("unexpected error when adding past capacity: %v", err)
		return
	}

	for lcv := 0; lcv < 10; lcv++ {
		// Make sure we can dequeue multiple messages
		info, found := errorQueue.Dequeue()
		if !found {
			t.Errorf("should have found item on iteration %d", lcv)
			return
		}

		if info.GetThreadID() != 10 {
			t.Errorf("unknown value %d on iteration %d", info.GetThreadID(), lcv)
			return
		}
		if info.GetError().Error() != "an error" {
			t.Errorf("unexpected error %s on iteration %d", info.GetError().Error(), lcv)
			return
		}
	}

}
