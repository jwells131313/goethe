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
	"errors"
	"testing"
)

func TestGoetheFactory(t *testing.T) {
	goethe := GetGoethe()
	if goethe == nil {
		t.Error("We did not get a goth")
		return
	}

	channel1 := make(chan int64)
	channel2 := make(chan int64)

	retTid1, err := goethe.Go(func() {
		tid := goethe.GetThreadID()
		channel1 <- tid
	})
	if err != nil {
		t.Errorf("error running thread %v", err)
		return
	}

	retTid2, err := goethe.Go(func() {
		tid := goethe.GetThreadID()
		channel2 <- tid
	})
	if err != nil {
		t.Errorf("error running thread %v", err)
		return
	}

	foundTid1 := <-channel1
	if foundTid1 <= 9 {
		t.Error("The tid for the function was less than or equal to 9, an error", foundTid1)
		return
	}

	foundTid2 := <-channel2
	if foundTid2 <= 9 {
		t.Error("The tid for the function was less than or equal to 9, an error", foundTid2)
		return
	}

	if foundTid1 == foundTid2 {
		t.Error("Tids should not be the same, got", foundTid1)
		return
	}

	if foundTid1 != retTid1 {
		t.Errorf("tid returned from go was %d, tid from inside was %d", retTid1, foundTid1)
		return
	}

	if foundTid2 != retTid2 {
		t.Errorf("tid returned from go(2) was %d, tid from inside was %d", retTid1, foundTid1)
		return
	}

	t.Logf("Got tids %v,%v", foundTid1, foundTid2)
}

func TestReturnNegativeOneOnNormalThread(t *testing.T) {
	goethe := GetGoethe()
	if goethe == nil {
		t.Error("We did not get a goth")
		return
	}

	nonGothTid := goethe.GetThreadID()
	if nonGothTid != -1 {
		t.Error("Non-goth tid must be -1, we got ", nonGothTid)
	}
}

func TestGoWithArgs(t *testing.T) {
	goethe := GetGoethe()

	ret := make(chan int)

	goethe.Go(addMe, 1, 2, 3, ret)

	val := <-ret

	if val != (1 + 2 + 3) {
		t.Errorf("did not get expected result, got %d", val)
		return
	}
}

func TestGoWithArgsFunctionReturnsError(t *testing.T) {
	goethe := GetGoethe()

	err := errors.New("an error")

	goethe.Go(returnError, err)
}

func addMe(a, b, c int, ret chan int) {
	ret <- a + b + c
}

func returnError(echo error) error {
	return echo
}
