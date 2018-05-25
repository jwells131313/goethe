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
	"fmt"
	"strings"
	"sync"
	"time"
)

// heapQueue is a queue of durations sorted as least duration first
// order of add is log n, order of remove is log n
type heapQueue interface {
	Add(*time.Time, interface{}) error
	Get() (*time.Time, interface{}, bool)
	Peek() (*time.Time, interface{}, bool)
	String() string
}

type heapNode struct {
	time    *time.Time
	payload interface{}
}

type heapQueueData struct {
	mux   sync.Mutex
	queue []*heapNode
}

func newHeap() heapQueue {
	return &heapQueueData{
		queue: make([]*heapNode, 0),
	}
}

func getParent(index int) int {
	if index <= 0 {
		return -1
	}

	return (index - 1) / 2
}

func getChildren(index int) (int, int) {
	tree := index * 2

	return tree + 1, tree + 2
}

func (heap *heapQueueData) Add(time *time.Time, payload interface{}) error {
	heap.mux.Lock()
	defer heap.mux.Unlock()

	nextIndex := len(heap.queue)
	node := &heapNode{
		time:    time,
		payload: payload,
	}
	heap.queue = append(heap.queue, node)

	// bubble it up
	parentIndex := getParent(nextIndex)
	currentIndex := nextIndex

	for (parentIndex >= 0) && (heap.getTime(parentIndex).After(*time)) {
		// Must swap
		heap.queue[parentIndex], heap.queue[currentIndex] = heap.queue[currentIndex], heap.queue[parentIndex]

		currentIndex = parentIndex
		parentIndex = getParent(currentIndex)
	}

	return nil
}

func (heap *heapQueueData) getTime(index int) *time.Time {
	node := heap.queue[index]
	return node.time
}

func (heap *heapQueueData) Get() (*time.Time, interface{}, bool) {
	heap.mux.Lock()
	defer heap.mux.Unlock()

	originalLen := len(heap.queue)
	if originalLen <= 0 {
		return nil, nil, false
	}

	retVal := heap.queue[0].time
	retPayload := heap.queue[0].payload

	if originalLen == 1 {
		// Last one, leave queue empty
		heap.queue = make([]*heapNode, 0)
		return retVal, retPayload, true
	}

	currentLength := originalLen - 1

	// Move last index to top
	heap.queue[0] = heap.queue[currentLength]
	// Cut off last node
	heap.queue = heap.queue[0:currentLength]

	currentIndex := 0
	bubbleTime := heap.queue[0].time

	for {
		leftChild, rightChild := getChildren(currentIndex)

		if leftChild >= currentLength {
			// We are at the end, no more swaps available
			return retVal, retPayload, true
		}

		if rightChild >= currentLength {
			// Only one more swap available, do it if we need it and leave
			if bubbleTime.After(*(heap.getTime(leftChild))) {
				// need to swap
				heap.queue[currentIndex], heap.queue[leftChild] = heap.queue[leftChild], heap.queue[currentIndex]
			}

			// no need to continue no more swaps possible
			return retVal, retPayload, true
		}

		// battle royale, both children are there
		swapIndex := rightChild // The default is to descend to the right if equal
		if heap.getTime(rightChild).After(*(heap.getTime(leftChild))) {
			swapIndex = leftChild
		}

		swapTime := heap.getTime(swapIndex)
		if swapTime.After(*bubbleTime) {
			// Done bubbling down, we can leave
			return retVal, retPayload, true
		}

		// Must swap and continue
		heap.queue[currentIndex], heap.queue[swapIndex] = heap.queue[swapIndex], heap.queue[currentIndex]

		currentIndex = swapIndex
	}
}

func (heap *heapQueueData) Peek() (*time.Time, interface{}, bool) {
	heap.mux.Lock()
	defer heap.mux.Unlock()

	if len(heap.queue) <= 0 {
		return nil, nil, false
	}

	node := heap.queue[0]

	return node.time, node.payload, true
}

func (heap *heapQueueData) String() string {
	retVal := make([]string, 0)

	retVal = append(retVal, "[")

	first := true
	for _, node := range heap.queue {
		u := node.time.Unix()

		if first {
			appendMe := fmt.Sprintf("%x", u)

			retVal = append(retVal, appendMe)

			first = false
		} else {
			appendMe := fmt.Sprintf(",%x", u)

			retVal = append(retVal, appendMe)
		}
	}

	retVal = append(retVal, "]")

	return strings.Join(retVal, "")
}
