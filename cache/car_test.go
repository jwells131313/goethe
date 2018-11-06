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
	"fmt"
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
	takeOffOfB2 = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1,
	}

	accessT2 = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1, 1, 5,
	}

	takeOffOfB1 = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1, 0,
	}

	equalT1T2 = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1, 0, 11, 12, 11, 12, 15, 16, 17, 18, 19,
	}

	maxOutB2KeysPlusOne = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1, 0, 11, 12, 11, 12, 15, 16, 17, 18, 19,
		15, 20, 16, 21, 17, 22,
	}

	cycleAccessedT2ToFindDemotion = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 1,
		11,
	}

	addToT1WithValueInB2 = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		11,
	}

	pushPToMax = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1, 0, 11, 12, 11, 12, 15, 16, 17, 18, 19,
		15, 20, 16, 21, 17, 22, 23, 18, 19,
	}

	pTo5BackTo2 = []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
		10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		1, 0, 11, 12, 11, 12, 15, 16, 17, 18, 19,
		9, 8, 7,
	}
)

func TestAddElevenToCacheSizeTen(t *testing.T) {
	carCache, err := NewComputeFunctionCARCache(10, func(key interface{}) (interface{}, error) {
		skey := key.(string)

		return strconv.Atoi(skey)
	})
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

func TestCARDestructor(t *testing.T) {
	var deletedKey, deletedValue interface{}

	carCache, err := NewCARCacheWithDestructor(5, &iConversionType{}, nil,
		func(k, v interface{}) error {
			deletedKey = k
			deletedValue = v
			return nil
		})
	if !assert.Nil(t, err, "could not create car cache") {
		return
	}

	carCache.Compute(zero)
	carCache.Compute(one)
	carCache.Compute(two)
	carCache.Compute(three)
	carCache.Compute(four)
	carCache.Compute(five)

	// Should have deleted zero
	assert.Equal(t, zero, deletedKey)
	assert.Equal(t, 0, deletedValue)
}

func TestCARFunctionDestructor(t *testing.T) {
	var deletedKey, deletedValue interface{}

	carCache, err := NewComputeFunctionCARCacheWithDestructor(5,
		func(key interface{}) (interface{}, error) {
			skey := key.(string)

			return strconv.Atoi(skey)
		},
		func(k, v interface{}) error {
			deletedKey = k
			deletedValue = v
			return nil
		})
	if !assert.Nil(t, err, "could not create car cache") {
		return
	}

	carCache.Compute(zero)
	carCache.Compute(one)
	carCache.Compute(two)
	carCache.Compute(three)
	carCache.Compute(four)
	carCache.Compute(five)

	// Should have deleted zero
	assert.Equal(t, zero, deletedKey)
	assert.Equal(t, 0, deletedValue)
}

func TestCARDestructorWithError(t *testing.T) {
	var deletedKey, deletedValue interface{}
	expectedError := fmt.Errorf("expected error")

	carCache, err := NewCARCacheWithDestructor(5, &iConversionType{}, nil,
		func(k, v interface{}) error {
			deletedKey = k
			deletedValue = v
			return expectedError
		})
	if !assert.Nil(t, err, "could not create car cache") {
		return
	}

	val, err := carCache.Compute(zero)
	if !assert.Nil(t, err, "compute failed") {
		return
	}
	if !assert.Equal(t, 0, val, "incorrect value") {
		return
	}
	val, err = carCache.Compute(one)
	if !assert.Nil(t, err, "compute failed") {
		return
	}
	if !assert.Equal(t, 1, val, "incorrect value") {
		return
	}
	val, err = carCache.Compute(two)
	if !assert.Nil(t, err, "compute failed") {
		return
	}
	if !assert.Equal(t, 2, val, "incorrect value") {
		return
	}
	val, err = carCache.Compute(three)
	if !assert.Nil(t, err, "compute failed") {
		return
	}
	if !assert.Equal(t, 3, val, "incorrect value") {
		return
	}
	val, err = carCache.Compute(four)
	if !assert.Nil(t, err, "compute failed") {
		return
	}
	if !assert.Equal(t, 4, val, "incorrect value") {
		return
	}
	val, err = carCache.Compute(five)
	if !assert.NotNil(t, err, "compute failed") {
		return
	}
	if !assert.Equal(t, 5, val, "incorrect value") {
		return
	}

	// Should have deleted zero
	assert.Equal(t, zero, deletedKey)
	assert.Equal(t, 0, deletedValue)
	assert.Equal(t, expectedError, err, "error does not match")
}

func TestTakeOffOfB2(t *testing.T) {
	runTest(t, takeOffOfB2, 10, 11, 0, 10, 1, 0, 0)
}

func TestAccessT2(t *testing.T) {
	runTest(t, accessT2, 10, 11, 0, 10, 1, 0, 0)
}

func TestTakeOffOfB1(t *testing.T) {
	runTest(t, takeOffOfB1, 10, 11, 0, 10, 0, 1, 1)
}

func TestEqualT1T2(t *testing.T) {
	runTest(t, equalT1T2, 10, 18, 5, 5, 0, 8, 5)
}

func TestMaxOutB2KeysPlusOne(t *testing.T) {
	runTest(t, maxOutB2KeysPlusOne, 10, 20, 5, 5, 0, 10, 5)
}

func TestCycleAccessedT2ToFindDemotion(t *testing.T) {
	runTest(t, cycleAccessedT2ToFindDemotion, 10, 12, 1, 9, 1, 1, 0)
}

func TestPushPToMax(t *testing.T) {
	runTest(t, pushPToMax, 10, 20, 4, 6, 0, 10, 10)
}

func TestPushTo5BackTo2(t *testing.T) {
	runTest(t, pTo5BackTo2, 10, 18, 2, 8, 3, 5, 2)
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

	checkOutputs(t, c, vs, ks, t1, t2, b1, b2, ep)
}

func TestAddToT2WithValueInB2(t *testing.T) {
	c, err := NewCARCache(10, newTestEchoCalculator(), nil)
	if !assert.Nil(t, err, "could not create cache %v", err) {
		return
	}

	for _, i := range addToT1WithValueInB2 {
		r, err := c.Compute(i)
		if !assert.Nil(t, err, "could not calculate %v", err) {
			return
		}

		if !assert.Equal(t, i, r, "did not get expected return") {
			return
		}
	}

	// At this point 0 is in B1, 1 is in B2, 7 is in T2
	// We remove 0 to make the size of B1 less than B2,
	// and remove a value from T2 to allow the cache to
	// just take a new one, then take it from B2
	c.Remove(func(rkey interface{}, _ interface{}) bool {
		key := rkey.(int)
		if key == 0 {
			return true
		}
		if key == 7 {
			return true
		}
		return false
	})
	c.Compute(1)

	checkOutputs(t, c, 10, 10, 1, 9, 0, 0, 0)
}

func checkOutputs(t *testing.T, c Cache, vs int, ks int, t1 int, t2 int, b1 int, b2 int, ep int) {
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
