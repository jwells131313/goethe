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
	"testing"
)

const (
	expectedError = "Expected Error"
)

func TestBasicCache(t *testing.T) {
	c, err := NewCache(newTestEchoCalculator(), nil)
	assert.Nil(t, err, "no new cache")

	reply, err := getStringValue(c, "hi")
	assert.Nil(t, err, "getting value (1)")
	assert.Equal(t, reply, "hi", "echo cache value is not the same")

	reply, err = getStringValue(c, "there")
	assert.Nil(t, err, "getting value (2)")
	assert.Equal(t, reply, "there", "echo cache value is not the same")

	reply, err = getStringValue(c, "there")
	assert.Nil(t, err, "getting value (3)")
	assert.Equal(t, reply, "there", "echo cache value is not the same")

	reply, err = getStringValue(c, "hi")
	assert.Nil(t, err, "getting value (4)")
	assert.Equal(t, reply, "hi", "echo cache value is not the same")
}

func TestClear(t *testing.T) {
	c, err := NewCache(newTestEchoCalculator(), nil)
	assert.Nil(t, err, "no new cache")

	c.Compute("hi")
	assert.True(t, c.HasKey("hi"), "should have the hi value")
	assert.False(t, c.HasKey("there"), "should not have the there value")
	assert.Equal(t, 1, c.Size(), "Size should be one")

	c.Clear()
	assert.False(t, c.HasKey("hi"), "should not have the hi value after clear")
	assert.False(t, c.HasKey("there"), "should not have the there value after clear")
	assert.Equal(t, 0, c.Size(), "should be zero size after clear")
}

func TestRemove(t *testing.T) {
	c, err := NewComputeFunctionCache(func(key interface{}) (interface{}, error) {
		return key, nil
	})

	assert.Nil(t, err, "no new cache")

	c.Compute("hi")
	c.Compute("there")
	c.Compute("sailor")

	assert.True(t, c.HasKey("hi"), "should have the hi value")
	assert.True(t, c.HasKey("there"), "should have the there value")
	assert.True(t, c.HasKey("sailor"), "should have the sailor value")
	assert.Equal(t, 3, c.Size(), "size should be three prior to remove")

	c.Remove(func(key interface{}, value interface{}) bool {
		if "there" == key || "sailor" == key {
			return true
		}

		return false
	})

	assert.True(t, c.HasKey("hi"), "should have the hi value")
	assert.False(t, c.HasKey("there"), "should not have the there value")
	assert.False(t, c.HasKey("sailor"), "should not have the sailor value")
	assert.Equal(t, 1, c.Size(), "should be one after two elements removed")

}

func TestNestedCache(t *testing.T) {
	nester := newTestNestedCalculator()
	cache, err := NewCache(nester, nil)
	assert.Nil(t, err, "could not create cache")

	nester.cache = cache

	values := []int{1, 2, 3, 4, 5, 6}
	for _, value := range values {
		reply, err := cache.Compute(value)
		assert.Nil(t, err, "could not get value")
		assert.Equal(t, reply, value, "cache returned incorrect value")
	}
}

func TestCycleCacheWithRemediator(t *testing.T) {
	called := false

	remediator := func(in interface{}) error {
		called = true
		return fmt.Errorf(expectedError)
	}

	cycler := newTestCyclicCalculator()
	cache, err := NewCache(cycler, remediator)
	assert.Nil(t, err, "could not create cache")

	cycler.cache = cache

	value, err := cache.Compute(1)
	assert.NotNil(t, err, "Did not get an expected error, instead got %v", value)
	assert.Equal(t, expectedError, err.Error(), "Did not get expected error from remediator")
	assert.True(t, called, "The remediator was not called")
}

func TestCycleCacheWithoutRemediator(t *testing.T) {
	cycler := newTestCyclicCalculator()
	cache, err := NewCache(cycler, nil)
	assert.Nil(t, err, "could not create cache")

	cycler.cache = cache

	value, err := cache.Compute(1)
	assert.NotNil(t, err, "Did not get an expected error, instead got %v", value)
}

func getStringValue(c Computable, in string) (string, error) {
	raw, err := c.Compute(in)
	if err != nil {
		return "", err
	}

	asString, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("reply was not a string")
	}

	return asString, nil
}

type testEchoCalculator struct {
}

func newTestEchoCalculator() Computable {
	return &testEchoCalculator{}
}

func (echo *testEchoCalculator) Compute(key interface{}) (interface{}, error) {
	return key, nil
}

type testNestedCalculator struct {
	cache Computable
}

func newTestNestedCalculator() *testNestedCalculator {
	return &testNestedCalculator{}
}

func (nest *testNestedCalculator) Compute(in interface{}) (interface{}, error) {
	key, err := itoi(in)
	if err != nil {
		return nil, err
	}

	if key < 5 {
		nextKey := key + 1
		nest.cache.Compute(nextKey)
	}

	return key, nil
}

type testCyclicCalculator struct {
	cache Computable
}

func newTestCyclicCalculator() *testCyclicCalculator {
	return &testCyclicCalculator{}
}

func (cycle *testCyclicCalculator) Compute(in interface{}) (interface{}, error) {
	key, err := itoi(in)
	if err != nil {
		return nil, err
	}

	if key < 5 {
		nextKey := key + 1
		_, err = cycle.cache.Compute(nextKey)
		if err != nil {
			return nil, err
		}
	} else if key >= 5 {
		// So if we start at 1, we should get a cycle
		_, err = cycle.cache.Compute(1)
		if err != nil {
			return nil, err
		}
	}

	return key, nil

}

func itoi(in interface{}) (int, error) {
	if in == nil {
		return -1, fmt.Errorf("incoming integer was nil")
	}

	retVal, ok := in.(int)
	if !ok {
		return -1, fmt.Errorf("incoming integer was not an integer %v", in)
	}

	return retVal, nil
}
