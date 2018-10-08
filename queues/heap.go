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
	"strings"
	"sync"
)

// Comparator is a function that returns
// 1 if a should be returned before b
// 0 if a is the same as b so they can be returned in any order
// -1 if a should be returned after b
type Comparator func(a interface{}, b interface{}) int

// Heap is a data structure that has log-n complexity insertion
// and removal.  You can only remove from the top of a heap, you
// cannot remove from the middle
type Heap interface {
	Add(interface{}) error
	Get() (interface{}, bool)
	Peek() (interface{}, bool)
	GetComparator() Comparator
}

type heapNode struct {
	parent  *heapData
	payload interface{}
}

type heapData struct {
	mux        sync.Mutex
	comparator Comparator
	queue      []*heapNode
}

func NewHeap(c Comparator) Heap {
	return &heapData{
		comparator: c,
		queue:      make([]*heapNode, 0),
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

func (node *heapNode) After(bNode *heapNode) bool {
	a := node.payload
	b := bNode.payload

	return node.parent.comparator(a, b) < 0
}

func (node *heapNode) AfterOrEqual(bNode *heapNode) bool {
	a := node.payload
	b := bNode.payload

	return node.parent.comparator(a, b) <= 0
}

func (heap *heapData) GetComparator() Comparator {
	return heap.comparator
}

func (heap *heapData) Add(payload interface{}) error {
	heap.mux.Lock()
	defer heap.mux.Unlock()

	nextIndex := len(heap.queue)
	node := &heapNode{
		parent:  heap,
		payload: payload,
	}
	heap.queue = append(heap.queue, node)

	// bubble it up
	parentIndex := getParent(nextIndex)
	currentIndex := nextIndex

	for (parentIndex >= 0) && (heap.queue[parentIndex].After(node)) {
		// Must swap
		heap.queue[parentIndex], heap.queue[currentIndex] = heap.queue[currentIndex], heap.queue[parentIndex]

		currentIndex = parentIndex
		parentIndex = getParent(currentIndex)
	}

	return nil
}

func (heap *heapData) Get() (interface{}, bool) {
	heap.mux.Lock()
	defer heap.mux.Unlock()

	originalLen := len(heap.queue)
	if originalLen <= 0 {
		return nil, false
	}

	retPayload := heap.queue[0].payload

	if originalLen == 1 {
		// Last one, leave queue empty
		heap.queue = make([]*heapNode, 0)
		return retPayload, true
	}

	currentLength := originalLen - 1

	// Move last index to top
	heap.queue[0] = heap.queue[currentLength]
	// Cut off last node
	heap.queue = heap.queue[0:currentLength]

	currentIndex := 0
	bubbleNode := heap.queue[0]

	for {
		leftChild, rightChild := getChildren(currentIndex)

		if leftChild >= currentLength {
			// We are at the end, no more swaps available
			return retPayload, true
		}

		if rightChild >= currentLength {
			// Only one more swap available, do it if we need it and leave
			if bubbleNode.After(heap.queue[leftChild]) {
				// need to swap
				heap.queue[currentIndex], heap.queue[leftChild] = heap.queue[leftChild], heap.queue[currentIndex]
			}

			// no need to continue no more swaps possible
			return retPayload, true
		}

		// battle royale, both children are there
		swapIndex := rightChild // The default is to descend to the right if equal
		if heap.queue[rightChild].After(heap.queue[leftChild]) {
			swapIndex = leftChild
		}

		swapNode := heap.queue[swapIndex]
		if swapNode.AfterOrEqual(bubbleNode) {
			// Done bubbling down, we can leave
			return retPayload, true
		}

		// Must swap and continue
		heap.queue[currentIndex], heap.queue[swapIndex] = heap.queue[swapIndex], heap.queue[currentIndex]

		currentIndex = swapIndex
	}
}

func (heap *heapData) Peek() (interface{}, bool) {
	heap.mux.Lock()
	defer heap.mux.Unlock()

	if len(heap.queue) <= 0 {
		return nil, false
	}

	node := heap.queue[0]

	return node.payload, true
}

func (heap *heapData) String() string {
	retVal := make([]string, 0)

	retVal = append(retVal, "[")

	first := true
	for _, node := range heap.queue {
		u := node.payload

		if first {
			appendMe := fmt.Sprintf("%v", u)

			retVal = append(retVal, appendMe)

			first = false
		} else {
			appendMe := fmt.Sprintf(",%v", u)

			retVal = append(retVal, appendMe)
		}
	}

	retVal = append(retVal, "]")

	return strings.Join(retVal, "")
}
