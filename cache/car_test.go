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
	"strconv"
	"testing"
)

const (
	zero  = "0"
	one   = "1"
	two   = "2"
	three = "3"
	four  = "4"
	five  = "5"
	six   = "6"
	seven = "7"
	eight = "8"
	nine  = "9"
	ten   = "10"
)

var (
	take_off_of_b2 = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1,
	}

	access_t2 = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1, 1, 5,
	}

	take_off_of_b1 = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1, 0,
	}

	equal_t1_t2 = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1, 0, 11, 12, 11, 12, 15, 16, 17, 18, 19,
	}

	max_out_b2_keys_plus_one = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1, 0, 11, 12, 11, 12, 15, 16, 17, 18, 19,
		15, 20, 16, 21, 17, 22,
	}

	cycle_accessed_t2_to_find_demotion = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1,
		11,
	}
)

func TestAddElevenToCacheSizeTen(t *testing.T) {
	carCache, err := NewCARCache(10, &iConversionType{}, nil)
	if !assert.Nil(t, err, "could not create car cache") {
		return
	}

	for lcv := 0; lcv < 2; lcv++ {
		carCache.Compute(zero)
		carCache.Compute(one)
		carCache.Compute(two)
		carCache.Compute(three)
		carCache.Compute(four)
		carCache.Compute(five)
		carCache.Compute(six)
		carCache.Compute(seven)
		carCache.Compute(eight)
		carCache.Compute(nine)
		carCache.Compute(ten)
	}

	assert.Equal(t, 10, getValueSize(carCache))
	assert.Equal(t, 10, getKeySize(carCache))
	assert.Equal(t, 0, getP(carCache))
}

func TestAddElevenToCacheSizeTenForwardThenBackward(t *testing.T) {
	carCache, err := NewCARCache(10, &iConversionType{}, nil)
	if !assert.Nil(t, err, "could not create car cache") {
		return
	}

	carCache.Compute(zero)
	carCache.Compute(one)
	carCache.Compute(two)
	carCache.Compute(three)
	carCache.Compute(four)
	carCache.Compute(five)
	carCache.Compute(six)
	carCache.Compute(seven)
	carCache.Compute(eight)
	carCache.Compute(nine)
	carCache.Compute(ten)

	carCache.Compute(ten)
	carCache.Compute(nine)
	carCache.Compute(eight)
	carCache.Compute(seven)
	carCache.Compute(six)
	carCache.Compute(five)
	carCache.Compute(four)
	carCache.Compute(three)
	carCache.Compute(two)
	carCache.Compute(one)
	carCache.Compute(zero)

	assert.Equal(t, 10, getValueSize(carCache))
	assert.Equal(t, 11, getKeySize(carCache))

	assert.Equal(t, 1, getT1Size(carCache))
	assert.Equal(t, 9, getT2Size(carCache))
	assert.Equal(t, 0, getB1Size(carCache))
	assert.Equal(t, 1, getB2Size(carCache))

	assert.Equal(t, 0, getP(carCache))
}

func TestTakeOffOfB2(t *testing.T) {
	runTest(t, take_off_of_b2, 10, 11, 0, 10, 1, 0, 0)
}

func TestAccessT2(t *testing.T) {
	runTest(t, access_t2, 10, 11, 0, 10, 1, 0, 0)
}

func TestTakeOffOfB1(t *testing.T) {
	runTest(t, take_off_of_b1, 10, 11, 0, 10, 0, 1, 1)
}

func TestEqualT1T2(t *testing.T) {
	runTest(t, equal_t1_t2, 10, 18, 5, 5, 0, 8, 5)
}

func TestMaxOutB2KeysPlusOne(t *testing.T) {
	runTest(t, max_out_b2_keys_plus_one, 10, 20, 5, 5, 0, 10, 5)
}

func TestCycleAccessedT2ToFindDemotion(t *testing.T) {
	runTest(t, cycle_accessed_t2_to_find_demotion, 10, 12, 1, 9, 1, 1, 0)
}

func runTest(t *testing.T, input []int, vs int, ks int, t1 int, t2 int, b1 int, b2 int, ep int) {
	c, err := NewCARCache(10, newTestEchoCalculator(), nil)
	if !assert.Nil(t, err, "could not create cache %v", err) {
		return
	}

	for _, i := range input {
		r, err := c.Compute(i)
		if !assert.Nil(t, err, "could not calculate %v", err) {
			return
		}

		if !assert.Equal(t, i, r, "did not get expected return") {
			return
		}
	}

	assert.Equal(t, vs, getValueSize(c))
	assert.Equal(t, ks, getKeySize(c))

	assert.Equal(t, t1, getT1Size(c))
	assert.Equal(t, t2, getT2Size(c))
	assert.Equal(t, b1, getB1Size(c))
	assert.Equal(t, b2, getB2Size(c))

	assert.Equal(t, ep, getP(c))
}

type iConversionType struct {
}

func (ict *iConversionType) Compute(key interface{}) (interface{}, error) {
	skey := key.(string)

	return strconv.Atoi(skey)
}

func getKeySize(c Cache) int {
	cc := c.(*carCache)
	return cc.T1.Size() + cc.T2.Size() + cc.B1.GetCurrentSize() + cc.B2.GetCurrentSize()
}

func getValueSize(c Cache) int {
	cc := c.(*carCache)
	return cc.T1.Size() + cc.T2.Size()
}

func getP(c Cache) int {
	cc := c.(*carCache)
	return cc.p
}

func getT1Size(c Cache) int {
	cc := c.(*carCache)
	return cc.T1.Size()
}

func getT2Size(c Cache) int {
	cc := c.(*carCache)
	return cc.T2.Size()
}

func getB1Size(c Cache) int {
	cc := c.(*carCache)
	return cc.B1.GetCurrentSize()
}

func getB2Size(c Cache) int {
	cc := c.(*carCache)
	return cc.B2.GetCurrentSize()
}
