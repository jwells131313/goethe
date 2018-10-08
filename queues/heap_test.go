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

package queues

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

func add(addMe int) *time.Time {
	t := time.Unix(int64(addMe), 0)

	return &t
}

func cmp(araw interface{}, braw interface{}) int {
	a := araw.(int)
	b := braw.(int)

	if a < b {
		return 1
	}
	if a == b {
		return 0
	}

	return -1
}

func TestHeapAddsAndRemoves(t *testing.T) {
	heap := NewHeap(cmp)

	heap.Add(5)
	heap.Add(12)
	heap.Add(10)
	heap.Add(5)
	heap.Add(8)
	heap.Add(10)
	heap.Add(13)
	heap.Add(14)

	returns := []int{5, 5, 8, 10, 10, 12, 13, 14}

	assert.True(t, checkReturns(t, heap, returns))
}

func TestAllTheSameStopsBubblingDown(t *testing.T) {
	heap := NewHeap(cmp)

	heap.Add(1)
	heap.Add(1)
	heap.Add(1)
	heap.Add(1)
	heap.Add(1)
	heap.Add(1)
	heap.Add(1)

	returns := []int{1, 1, 1, 1, 1, 1, 1}

	assert.True(t, checkReturns(t, heap, returns))
}

func TestHeapAddsAndRemovesToGetOnlyLeftSwap(t *testing.T) {
	heap := NewHeap(cmp)

	heap.Add(8)
	heap.Add(9)
	heap.Add(10)
	heap.Add(12)
	heap.Add(13)

	returns := []int{8, 9, 10, 12, 13}

	assert.True(t, checkReturns(t, heap, returns))
}

func TestHeapString(t *testing.T) {
	heap := NewHeap(cmp)

	heap.Add(1)
	heap.Add(3)
	heap.Add(4)
	heap.Add(2)

	stringNow := fmt.Sprintf("%v", heap)

	assert.True(t, strings.Contains(stringNow, "1"))
	assert.True(t, strings.Contains(stringNow, "2"))
	assert.True(t, strings.Contains(stringNow, "3"))
	assert.True(t, strings.Contains(stringNow, "4"))

	returns := []int{1, 2, 3, 4}

	assert.True(t, checkReturns(t, heap, returns))

	stringAfter := fmt.Sprintf("%v", heap)

	assert.Equal(t, "[]", stringAfter)
}

func TestGetComparator(t *testing.T) {
	heap := NewHeap(cmp)

	f := heap.GetComparator()

	assert.Equal(t, -1, f(3, 2))
	assert.Equal(t, 0, f(2, 2))
	assert.Equal(t, 1, f(2, 3))
}

func checkReturns(t *testing.T, heap Heap, expecteds []int) bool {
	for i, d := range expecteds {
		dFound, there := heap.Peek()
		if !assert.True(t, there,
			"(1) Should have found %v on iteration %d, but found %v", d, i, dFound) {
			return false
		}

		if !assert.Equal(t, d, dFound,
			"(2) Should have found %v on iteration %d, but found %v", d, i, dFound) {
			return false
		}

		dFound, there = heap.Get()
		if !assert.True(t, there, "(3) Should have found %v on iteration %d, but found %v", d, i, dFound) {
			return false
		}

		if !assert.Equal(t, d, dFound, "(4) Should have found %v on iteration %d, but found %v", d, i, dFound) {
			return false
		}
	}

	_, found := heap.Peek()
	if !assert.False(t, found, "Peek should have returned false") {
		return false
	}

	_, found = heap.Get()
	if !assert.False(t, found, "Get should have returned false") {
		return false
	}

	return true
}
