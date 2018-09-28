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

package cache

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddCarClock(t *testing.T) {
	cc := newCarClock()
	if !assert.Equal(t, 0, cc.Size()) {
		return
	}

	cc.AddTail(1, -1)
	if !assert.Equal(t, 1, cc.Size()) {
		return
	}

	if !assert.False(t, cc.GetPageReferenceOfHead(), "initial page ref of head should be false") {
		return
	}

	cc.SetPageReference(1)

	if !assert.True(t, cc.GetPageReferenceOfHead(), "page ref of key should be true") {
		return
	}

	raw, found := cc.Get(1)
	if !assert.True(t, found, "1 is there") {
		return
	}

	retVal := raw.(int)
	if !assert.Equal(t, -1, retVal) {
		return
	}

	_, found = cc.Get(1)
	if !assert.True(t, found, "1 is there") {
		return
	}

	_, found = cc.Get(2)
	if !assert.False(t, found, "2 is not there yet") {
		return
	}

	cc.AddTail(2, -2)
	if !assert.Equal(t, 2, cc.Size()) {
		return
	}

	// 1 is at head and has page bit set.  2 is tail and does not have page bit set
	if !assert.True(t, cc.GetPageReferenceOfHead(), "page ref of key should be true") {
		return
	}

	if !checkRemove(t, cc, 1) {
		return
	}

	// Now 2 should be at head with bit not set
	if !assert.False(t, cc.GetPageReferenceOfHead(), "page ref of key should be true") {
		return
	}
}

func TestInternalPageReferences(t *testing.T) {
	cc := newCarClock()

	cc.AddTail(1, -1)
	cc.AddTail(2, -2)
	cc.AddTail(3, -3)
	cc.AddTail(4, -4)

	cc.SetPageReference(2)
	cc.SetPageReference(3)

	if !assert.Equal(t, 4, cc.Size()) {
		return
	}

	// 1, 2, 3, 4
	if !assert.False(t, cc.GetPageReferenceOfHead()) {
		return
	}

	if !checkRemove(t, cc, 1) {
		return
	}

	// 2, 3, 4
	if !assert.True(t, cc.GetPageReferenceOfHead()) {
		return
	}

	if !checkRemove(t, cc, 2) {
		return
	}

	// 3, 4
	if !assert.True(t, cc.GetPageReferenceOfHead()) {
		return
	}

	if !checkRemove(t, cc, 3) {
		return
	}

	// 4
	if !assert.False(t, cc.GetPageReferenceOfHead()) {
		return
	}

	if !checkRemove(t, cc, 4) {
		return
	}

	assert.Equal(t, 0, cc.Size())

}

func TestRemoveAll(t *testing.T) {
	cc := newCarClock()

	cc.AddTail(0, 0)
	cc.AddTail(1, -1)
	cc.AddTail(2, -2)
	cc.AddTail(3, -3)
	cc.AddTail(4, -4)
	cc.AddTail(5, -5)
	cc.AddTail(6, -6)
	cc.AddTail(7, -7)
	cc.AddTail(8, -8)
	cc.AddTail(9, -9)

	if !assert.Equal(t, 10, cc.Size()) {
		return
	}

	cc.RemoveAll(func(key interface{}, value interface{}) bool {
		ikey := key.(int)

		if ikey%2 == 0 {
			return true
		}

		return false
	})

	if !assert.Equal(t, 5, cc.Size()) {
		return
	}

	cc.AddTail(0, 0)
	cc.AddTail(2, 2)

	if !assert.Equal(t, 7, cc.Size()) {
		return
	}

	for cc.Size() > 0 {
		cc.RemoveHead()
	}
}

func checkRemove(t *testing.T, cc carClock, key int) bool {
	k, v, found := cc.RemoveHead()

	success := true

	if !assert.True(t, found) {
		success = false
	}
	if !assert.Equal(t, key, k) {
		success = false
	}
	expectedValue := -1 * key

	if !assert.Equal(t, expectedValue, v) {
		success = false
	}

	return success
}
