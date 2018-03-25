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
