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

package internal

import (
	"testing"
	"time"
)

func TestHeapAddsAndRemoves(t *testing.T) {
	heap := NewHeap()

	heap.Add(5)
	heap.Add(12)
	heap.Add(10)
	heap.Add(5)
	heap.Add(8)
	heap.Add(10)
	heap.Add(13)
	heap.Add(14)

	returns := []time.Duration{5, 5, 8, 10, 10, 12, 13, 14}

	for i, d := range returns {
		dFound, there := heap.Peek()
		if !there {
			t.Errorf("(1) Should have found %d on iteration %d, but found %d", d, i, dFound)
			return
		}

		if dFound != d {
			t.Errorf("(2) Should have found %d on iteration %d, but found %d", d, i, dFound)
			return
		}

		dFound, there = heap.Get()
		if !there {
			t.Errorf("(3) Should have found %d on iteration %d, but found %d", d, i, dFound)
			return
		}

		if dFound != d {
			t.Errorf("(4) Should have found %d on iteration %d, but found %d", d, i, dFound)
			return
		}
	}

	_, found := heap.Peek()
	if found {
		t.Errorf("Peek should have returned false")
		return
	}

	_, found = heap.Get()
	if found {
		t.Errorf("Get should have returned false")
		return
	}

}

func TestHeapAddsAndRemovesToGetOnlyLeftSwap(t *testing.T) {
	heap := NewHeap()

	heap.Add(8)
	heap.Add(9)
	heap.Add(10)
	heap.Add(12)
	heap.Add(13)

	returns := []time.Duration{8, 9, 10, 12, 13}

	for i, d := range returns {
		dFound, there := heap.Peek()
		if !there {
			t.Errorf("(1) Should have found %d on iteration %d, but found %d", d, i, dFound)
			return
		}

		if dFound != d {
			t.Errorf("(2) Should have found %d on iteration %d, but found %d", d, i, dFound)
			return
		}

		dFound, there = heap.Get()
		if !there {
			t.Errorf("(3) Should have found %d on iteration %d, but found %d", d, i, dFound)
			return
		}

		if dFound != d {
			t.Errorf("(4) Should have found %d on iteration %d, but found %d", d, i, dFound)
			return
		}
	}

	_, found := heap.Peek()
	if found {
		t.Errorf("Peek should have returned false")
		return
	}

	_, found = heap.Get()
	if found {
		t.Errorf("Get should have returned false")
		return
	}

}
