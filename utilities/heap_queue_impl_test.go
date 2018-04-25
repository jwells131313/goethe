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

package utilities

import (
	"testing"
	"time"
)

func add(addMe int) *time.Time {
	t := time.Unix(int64(addMe), 0)

	return &t
}

func TestHeapAddsAndRemoves(t *testing.T) {
	heap := NewHeap()

	heap.Add(add(5), nil)
	heap.Add(add(12), nil)
	heap.Add(add(10), nil)
	heap.Add(add(5), nil)
	heap.Add(add(8), nil)
	heap.Add(add(10), nil)
	heap.Add(add(13), nil)
	heap.Add(add(14), nil)

	returns := []*time.Time{
		add(5),
		add(5),
		add(8),
		add(10),
		add(10),
		add(12),
		add(13),
		add(14),
	}

	for i, d := range returns {
		dFound, _, there := heap.Peek()
		if !there {
			t.Errorf("(1) Should have found %v on iteration %d, but found %v", d, i, dFound)
			return
		}

		if *dFound != *d {
			t.Errorf("(2) Should have found %v on iteration %d, but found %v", d, i, dFound)
			return
		}

		dFound, _, there = heap.Get()
		if !there {
			t.Errorf("(3) Should have found %v on iteration %d, but found %v", d, i, dFound)
			return
		}

		if *dFound != *d {
			t.Errorf("(4) Should have found %v on iteration %d, but found %v", d, i, dFound)
			return
		}
	}

	_, _, found := heap.Peek()
	if found {
		t.Errorf("Peek should have returned false")
		return
	}

	_, _, found = heap.Get()
	if found {
		t.Errorf("Get should have returned false")
		return
	}

}

func TestHeapAddsAndRemovesToGetOnlyLeftSwap(t *testing.T) {
	heap := NewHeap()

	heap.Add(add(8), nil)
	heap.Add(add(9), nil)
	heap.Add(add(10), nil)
	heap.Add(add(12), nil)
	heap.Add(add(13), nil)

	returns := []*time.Time{
		add(8),
		add(9),
		add(10),
		add(12),
		add(13),
	}

	for i, d := range returns {
		dFound, _, there := heap.Peek()
		if !there {
			t.Errorf("(1) Should have found %v on iteration %d, but found %v", d, i, dFound)
			return
		}

		if *dFound != *d {
			t.Errorf("(2) Should have found %v on iteration %d, but found %v", d, i, dFound)
			return
		}

		dFound, _, there = heap.Get()
		if !there {
			t.Errorf("(3) Should have found %v on iteration %d, but found %v", d, i, dFound)
			return
		}

		if *dFound != *d {
			t.Errorf("(4) Should have found %v on iteration %d, but found %v", d, i, dFound)
			return
		}
	}

	_, _, found := heap.Peek()
	if found {
		t.Errorf("Peek should have returned false")
		return
	}

	_, _, found = heap.Get()
	if found {
		t.Errorf("Get should have returned false")
		return
	}

}
