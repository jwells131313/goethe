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
	"testing"
)

type bBB struct {
	a, b, c int
}

func TestGetValues(t *testing.T) {
	input := make([]interface{}, 0)
	input = append(input, "a")
	input = append(input, "b")
	input = append(input, "c")

	v, err := GetValues(aAa, input)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	if len(v) != 3 {
		t.Errorf("unexpected number of return values %d", len(v))
	}
}

func TestInvokeWithNil(t *testing.T) {
	bb0 := &bBB{
		a: 1,
		b: 2,
		c: 3,
	}
	bb2 := &bBB{
		a: 4,
		b: 5,
		c: 6,
	}
	bb3 := bBB{
		a: 7,
		b: 8,
		c: 9,
	}

	input := make([]interface{}, 0)
	input = append(input, bb0)
	input = append(input, nil)
	input = append(input, bb2)
	input = append(input, bb3)

	rChan := make(chan *bBB)

	input = append(input, rChan)

	v, err := GetValues(bbB, input)
	if err != nil {
		t.Errorf("%v", err)
		return
	}

	go func() {
		Invoke(bbB, v, nil)
	}()

	r0 := <-rChan
	err = checkbBB(r0, 1, 2, 3)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	r1 := <-rChan
	if r1 != nil {
		t.Errorf("expected nil second paramater got %v", r1)
		return
	}

	r2 := <-rChan
	err = checkbBB(r2, 4, 5, 6)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	r3 := <-rChan
	err = checkbBB(r3, 7, 8, 9)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

}

func checkbBB(cMe *bBB, a, b, c int) error {
	if cMe.a != a {
		return fmt.Errorf("Invalid a, expected %d got %d", a, cMe.a)
	}

	if cMe.b != b {
		return fmt.Errorf("Invalid b, expected %d got %d", b, cMe.b)
	}

	if cMe.c != c {
		return fmt.Errorf("Invalid b, expected %d got %d", c, cMe.c)
	}

	return nil
}

func aAa(a, b, c string) {

}

func bbB(a, b, c *bBB, d bBB, rChan chan *bBB) {
	rChan <- a
	rChan <- b
	rChan <- c
	rChan <- &d
	close(rChan)
}
