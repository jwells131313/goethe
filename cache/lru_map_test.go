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

func TestBasicLRU(t *testing.T) {
	lkm := newLRUKeyMap()
	assert.Equal(t, 0, lkm.GetCurrentSize())

	lkm.AddMRU(1)
	assert.Equal(t, 1, lkm.GetCurrentSize())

	assert.True(t, lkm.Contains(1))
	assert.False(t, lkm.Contains(2))

	lkm.AddMRU(2)
	assert.Equal(t, 2, lkm.GetCurrentSize())

	assert.True(t, lkm.Contains(1))
	assert.True(t, lkm.Contains(2))

	lkm.RemoveLRU()

	assert.Equal(t, 1, lkm.GetCurrentSize())

	assert.False(t, lkm.Contains(1))
	assert.True(t, lkm.Contains(2))

	lkm.RemoveLRU()

	assert.Equal(t, 0, lkm.GetCurrentSize())

	assert.False(t, lkm.Contains(1))
	assert.False(t, lkm.Contains(2))

	// Should be no-op
	lkm.RemoveLRU()

	assert.Equal(t, 0, lkm.GetCurrentSize())

	assert.False(t, lkm.Contains(1))
	assert.False(t, lkm.Contains(2))
}

func TestRemoveLRU(t *testing.T) {
	lkm := newLRUKeyMap()

	lkm.AddMRU(1)
	lkm.AddMRU(2)
	lkm.AddMRU(3)
	lkm.AddMRU(4)
	lkm.AddMRU(5)

	if !assert.True(t, checkOneThroughFive(t, lkm, []bool{true, true, true, true, true})) {
		return
	}

	lkm.Remove(6)

	if !assert.True(t, checkOneThroughFive(t, lkm, []bool{true, true, true, true, true})) {
		return
	}

	lkm.Remove(3)

	if !assert.True(t, checkOneThroughFive(t, lkm, []bool{true, true, false, true, true})) {
		return
	}

	lkm.Remove(5)

	if !assert.True(t, checkOneThroughFive(t, lkm, []bool{true, true, false, true, false})) {
		return
	}

	lkm.Remove(1)

	if !assert.True(t, checkOneThroughFive(t, lkm, []bool{false, true, false, true, false})) {
		return
	}

	lkm.Remove(2)

	if !assert.True(t, checkOneThroughFive(t, lkm, []bool{false, false, false, true, false})) {
		return
	}

	lkm.Remove(4)

	if !assert.True(t, checkOneThroughFive(t, lkm, []bool{false, false, false, false, false})) {
		return
	}

	// Should do nothing
	lkm.Remove(4)

	if !assert.True(t, checkOneThroughFive(t, lkm, []bool{false, false, false, false, false})) {
		return
	}
}

func checkOneThroughFive(t *testing.T, lru lruKeyMap, checkMe []bool) bool {
	numExpected := 0
	for index, check := range checkMe {
		val := index + 1

		if !assert.Equal(t, check, lru.Contains(val), "Value %d", val) {
			return false
		}
		if check {
			numExpected++
		}
	}

	if !assert.Equal(t, numExpected, lru.GetCurrentSize(), "Invalid lru size") {
		return false
	}

	return true
}

func TestRemoveAllLRU(t *testing.T) {
	lkm := newLRUKeyMap()

	for lcv := 0; lcv < 10; lcv++ {
		lkm.AddMRU(lcv)
	}

	assert.Equal(t, 10, lkm.GetCurrentSize())

	lkm.RemoveAll(func(key interface{}, value interface{}) bool {
		ikey := key.(int)
		if ikey%2 == 0 {
			return true
		}

		return false
	})

	assert.Equal(t, 5, lkm.GetCurrentSize())

	for lcv := 0; lcv < 10; lcv++ {
		expectedResult := lcv%2 != 0

		assert.Equal(t, expectedResult, lkm.Contains(lcv), "Expected: %v, lcv: %d", expectedResult, lcv)
	}

	for lcv := 0; lcv < 10; lcv++ {
		if lcv%2 == 0 {
			lkm.AddMRU(lcv)
		}
	}

	assert.Equal(t, 10, lkm.GetCurrentSize())

	for lcv := 0; lcv < 10; lcv++ {
		assert.True(t, lkm.Contains(lcv))
	}
}
